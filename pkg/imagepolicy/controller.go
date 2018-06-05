package imagepolicy

import (
	"context"
	"errors"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/manifoldco/heighliner/pkg/api/v1alpha1"
	"github.com/manifoldco/heighliner/pkg/registry"
	"github.com/manifoldco/heighliner/pkg/registry/hub"

	"github.com/jelmersnoeck/kubekit"
	"github.com/jelmersnoeck/kubekit/patcher"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	cmdutil "k8s.io/kubernetes/pkg/kubectl/cmd/util"
)

type (
	getClient interface {
		Get(interface{}, string, string) error
	}

	applyClient interface {
		Apply(runtime.Object, ...patcher.OptionFunc) ([]byte, error)
	}

	patchClient interface {
		getClient
		applyClient
	}
)

// Controller will take care of syncing the internal status of the ImagePolicy
// object with the available images filtered by repo, releases, and versions
type Controller struct {
	rc        *rest.RESTClient
	cs        kubernetes.Interface
	patcher   patchClient
	namespace string
}

// NewController returns a new ImagePolicy Controller.
func NewController(rcfg *rest.Config, cs kubernetes.Interface, namespace string) (*Controller, error) {
	rc, err := kubekit.RESTClient(rcfg, &v1alpha1.SchemeGroupVersion, v1alpha1.AddToScheme)
	if err != nil {
		return nil, err
	}

	return &Controller{
		cs:        cs,
		rc:        rc,
		patcher:   patcher.New("hlnr-image-policy", cmdutil.NewFactory(nil)),
		namespace: namespace,
	}, nil
}

// Run runs the Controller in the background and sets up watchers to take action
// when the desired state is altered.
func (c *Controller) Run() error {
	ctx, cancel := context.WithCancel(context.Background())

	log.Printf("Starting controller...")

	go c.run(ctx)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	<-quit
	log.Printf("Shutdown requested...")
	cancel()

	<-ctx.Done()
	log.Printf("Shutting down...")

	return nil
}

func (c *Controller) run(ctx context.Context) {
	watcher := kubekit.NewWatcher(
		c.rc,
		c.namespace,
		&ImagePolicyResource,
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				c.syncPolicy(obj)
			},
			UpdateFunc: func(old, new interface{}) {
				c.syncPolicy(new)
			},
			DeleteFunc: func(obj interface{}) {
				cp := obj.(*v1alpha1.ImagePolicy).DeepCopy()
				log.Printf("Deleting ImagePolicy %s", cp.Name)
			},
		},
	)

	go watcher.Run(ctx.Done())
}

func (c *Controller) syncPolicy(obj interface{}) error {
	ip := obj.(*v1alpha1.ImagePolicy).DeepCopy()

	// TODO: make this generic for various filters. Gitlab etc.
	repo, err := getGithubRepository(c.patcher, ip)
	if err != nil {
		log.Printf("Could not retrieve GithubRepository for %s: %s", ip.Name, err)
		return nil
	}

	registry, err := getRegistry(c.patcher, ip)
	if err != nil {
		log.Printf("Could not retrieve registry for %s: %s", ip.Name, err)
		return nil
	}

	vp, err := getVersioningPolicy(c.patcher, ip)
	if err != nil {
		log.Printf("Could not retrieve VersioningPolicy for %s: %s", ip.Name, err)
		return nil
	}

	ip.Status.Releases, err = filterImages(ip.Spec.Image, repo, registry, vp)
	if err != nil {
		log.Printf("Could not filter images for %s: %s", ip.Name, err)
		return nil
	}

	// need to specify types again until we resolve the mapping issue
	ip.TypeMeta = metav1.TypeMeta{
		Kind:       "ImagePolicy",
		APIVersion: "hlnr.io/v1alpha1",
	}

	if _, err := c.patcher.Apply(ip); err != nil {
		log.Printf("Error syncing ImagePolicy %s (%s): %s", ip.Name, ip.Namespace, err)
		return err
	}

	return nil
}

func getGithubRepository(cl patchClient, ip *v1alpha1.ImagePolicy) (*v1alpha1.GitHubRepository, error) {
	githubRepository := &v1alpha1.GitHubRepository{
		TypeMeta: metav1.TypeMeta{
			Kind:       "GitHubRepository",
			APIVersion: "hlnr.io/v1alpha1",
		},
	}

	if err := cl.Get(githubRepository, ip.Namespace, ip.Spec.Filter.GitHub.Name); err != nil {
		return nil, err
	}

	return githubRepository, nil
}

func getRegistry(cl patchClient, ip *v1alpha1.ImagePolicy) (registry.Registry, error) {

	pullSecrets := ip.Spec.ImagePullSecrets
	if len(pullSecrets) == 0 {
		return nil, errors.New("No ImagePullSecrets available")
	}
	registrySecrets := ip.Spec.ImagePullSecrets[0].Name

	secret := &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: "v1",
		},
	}

	if err := cl.Get(secret, ip.Namespace, registrySecrets); err != nil {
		return nil, err
	}

	registryClient, err := hub.New(secret, ip.Spec.Image) // TODO: make this generic to multiple container registries
	if err != nil {
		return nil, err
	}

	return registryClient, nil
}

func getVersioningPolicy(cl patchClient, ip *v1alpha1.ImagePolicy) (*v1alpha1.VersioningPolicy, error) {
	vp := &v1alpha1.VersioningPolicy{
		TypeMeta: metav1.TypeMeta{
			Kind:       "VersioningPolicy",
			APIVersion: "hlnr.io/v1alpha1",
		},
	}

	policyName := ip.Spec.VersioningPolicy.Name
	if err := cl.Get(vp, ip.Namespace, policyName); err != nil {
		return nil, err
	}

	return vp, nil
}

// filter images available on the image policy status by release level and image registry tags
func filterImages(image string, repo *v1alpha1.GitHubRepository, registry registry.Registry, vp *v1alpha1.VersioningPolicy) ([]v1alpha1.Release, error) {

	// define the source as an OwnerReference
	source := metav1.NewControllerRef(
		repo,
		v1alpha1.SchemeGroupVersion.WithKind(kubekit.TypeName(repo)),
	)

	releases := []v1alpha1.Release{}
	for _, release := range repo.Status.Releases {

		if release.Level != vp.Spec.SemVer.Level {
			continue
		}

		releaseStatus, err := registry.GetManifest(image, release.Tag)
		if err != nil {
			return nil, err
		}
		if !releaseStatus {
			log.Printf("Release %s with tag %s is not available in the registry", release.Name, release.Tag)
			continue
		}

		confirmedRelease := v1alpha1.Release{
			SemVer: &v1alpha1.SemVerRelease{
				Name:    release.Name,
				Version: release.Tag,
			},
			Released: release.ReleasedAt,
			Image:    image + ":" + release.Tag,
			Source:   source,
		}

		releases = append(releases, confirmedRelease)
	}

	return releases, nil
}
