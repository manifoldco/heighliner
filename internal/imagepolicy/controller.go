package imagepolicy

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/manifoldco/heighliner/apis/heighliner/v1alpha1"
	"github.com/manifoldco/heighliner/internal/registry"

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

var (
	registriesMu sync.RWMutex
	registries   = make(map[string]registryGetter)
)

type registryGetter func(*corev1.Secret) (registry.Registry, error)

// AddRegistry allows a container registry to register itself as a possible
// option to retrieve images. If the func getter is nil or the same registry
// name has been used, the function panics.
func AddRegistry(name string, r registryGetter) {
	registriesMu.Lock()
	defer registriesMu.Unlock()
	if r == nil {
		panic("registry getter is nil")
	}
	if _, dup := registries[name]; dup {
		panic("register called twice for " + name)
	}
	registries[name] = r
}

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
	logger    *log.Logger
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
		logger:    log.New(os.Stderr, "", log.LstdFlags),
	}, nil
}

// Run runs the Controller in the background and sets up watchers to take action
// when the desired state is altered.
func (c *Controller) Run() error {
	ctx, cancel := context.WithCancel(context.Background())

	c.logger.Printf("Starting controller...")

	go c.run(ctx)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	<-quit
	c.logger.Printf("Shutdown requested...")
	cancel()

	<-ctx.Done()
	c.logger.Printf("Shutting down...")

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
				c.logger.Printf("Deleting ImagePolicy %s", cp.Name)
			},
		},
	)

	go watcher.Run(ctx.Done())
}

func (c *Controller) syncPolicy(obj interface{}) error {
	ip := obj.(*v1alpha1.ImagePolicy).DeepCopy()

	registry, err := getRegistry(c.patcher, ip)
	if err != nil {
		c.logger.Printf("Could not retrieve registry for %s: %s", ip.Name, err)
		return nil
	}

	vp, err := getVersioningPolicy(c.patcher, ip)
	if err != nil {
		c.logger.Printf("Could not retrieve VersioningPolicy for %s: %s", ip.Name, err)
		return nil
	}

	switch {
	case ip.Spec.Filter.GitHub != nil:
		repo, err := getGithubRepository(c.patcher, ip)
		if err != nil {
			c.logger.Printf("Could not retrieve GithubRepository for %s: %s", ip.Name, err)
			return nil
		}

		ip.Status.Releases, err = filterImages(ip.Spec.Image, ip.Spec.Match, repo, registry, vp)
		if err != nil {
			c.logger.Printf("Could not filter images for %s: %s", ip.Name, err)
			return nil
		}

	case ip.Spec.Filter.Pinned != nil:
		pinned := ip.Spec.Filter.Pinned

		r := v1alpha1.Release{
			SemVer: pinned,
			Level:  vp.Spec.SemVer.Level,
			Image:  ip.Spec.Image + ":" + pinned.Version,
		}

		ip.Status.Releases = []v1alpha1.Release{r}
	default:
		return errors.New("image spec filter not defined")
	}

	// need to specify types again until we resolve the mapping issue
	ip.TypeMeta = metav1.TypeMeta{
		Kind:       "ImagePolicy",
		APIVersion: "hlnr.io/v1alpha1",
	}

	if _, err := c.patcher.Apply(ip); err != nil {
		c.logger.Printf("Error syncing ImagePolicy %s (%s): %s", ip.Name, ip.Namespace, err)
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

	ghName := ip.Spec.Filter.GitHub.Name
	ghNamespace := ip.Spec.Filter.GitHub.Namespace
	if ghNamespace == "" {
		ghNamespace = ip.Namespace
	}

	if err := cl.Get(githubRepository, ghNamespace, ghName); err != nil {
		return nil, err
	}

	return githubRepository, nil
}

func getRegistry(cl patchClient, ip *v1alpha1.ImagePolicy) (registry.Registry, error) {

	var pullSecrets []corev1.LocalObjectReference
	if ip.Spec.ContainerRegistry != nil {
		pullSecrets = ip.Spec.ContainerRegistry.ImagePullSecrets
	}

	if len(pullSecrets) == 0 {
		return nil, errors.New("No ImagePullSecrets available")
	}
	registrySecrets := ip.Spec.ContainerRegistry.ImagePullSecrets[0].Name

	secret := &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: "v1",
		},
	}

	if err := cl.Get(secret, ip.Namespace, registrySecrets); err != nil {
		return nil, err
	}

	name := ip.Spec.ContainerRegistry.Registry()
	registriesMu.RLock()
	getter, ok := registries[name]
	registriesMu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("unknown %s registry", name)
	}

	return getter(secret)
}

func getVersioningPolicy(cl patchClient, ip *v1alpha1.ImagePolicy) (*v1alpha1.VersioningPolicy, error) {
	vp := &v1alpha1.VersioningPolicy{
		TypeMeta: metav1.TypeMeta{
			Kind:       "VersioningPolicy",
			APIVersion: "hlnr.io/v1alpha1",
		},
	}

	policyName := ip.Spec.VersioningPolicy.Name
	policyNamespace := ip.Spec.VersioningPolicy.Namespace
	if policyNamespace == "" {
		policyNamespace = ip.Namespace
	}

	if err := cl.Get(vp, policyNamespace, policyName); err != nil {
		return nil, err
	}

	return vp, nil
}

// filter images available on the image policy status by release level and image registry tags
func filterImages(image string, matcher *v1alpha1.ImagePolicyMatch, repo *v1alpha1.GitHubRepository, reg registry.Registry, vp *v1alpha1.VersioningPolicy) ([]v1alpha1.Release, error) {
	releases := []v1alpha1.Release{}
	for _, release := range repo.Status.Releases {

		if release.Level != vp.Spec.SemVer.Level {
			continue
		}

		tag, err := reg.TagFor(image, release.Tag, matcher)
		if registry.IsTagNotFoundError(err) {
			log.Printf("Release %s for tag %s is not available in the registry", release.Name, release.Tag)
			continue
		}

		if err != nil {
			return nil, err
		}

		confirmedRelease := v1alpha1.Release{
			SemVer: &v1alpha1.SemVerRelease{
				Name:    release.Name,
				Version: release.Tag,
			},
			Level:       release.Level,
			ReleaseTime: release.ReleaseTime,
			Image:       image + ":" + tag,
		}

		releases = append(releases, confirmedRelease)
	}

	return releases, nil
}
