package networkpolicy

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/manifoldco/heighliner/pkg/api/v1alpha1"
	"github.com/manifoldco/heighliner/pkg/k8sutils"

	"github.com/jelmersnoeck/kubekit"
	"github.com/jelmersnoeck/kubekit/patcher"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	cmdutil "k8s.io/kubernetes/pkg/kubectl/cmd/util"
)

// Controller represents the MicroserviceController. This controller
// takes care of creating, updating and deleting lower level Kubernetese
// components that are associated with a specific Microservice.
type Controller struct {
	rc        *rest.RESTClient
	cs        kubernetes.Interface
	namespace string
	patcher   *patcher.Patcher
}

// NewController returns a new ConfigPolicy Controller.
func NewController(cfg *rest.Config, cs kubernetes.Interface, namespace string) (*Controller, error) {
	rc, err := kubekit.RESTClient(cfg, &v1alpha1.SchemeGroupVersion, v1alpha1.AddToScheme)
	if err != nil {
		return nil, err
	}

	return &Controller{
		cs:        cs,
		rc:        rc,
		namespace: namespace,
		patcher:   patcher.New("hlnr-network-policy", cmdutil.NewFactory(nil)),
	}, nil
}

// Run runs the Controller in the background and sets up watchers to take action
// when the desired state is altered.
func (c *Controller) Run() error {
	log.Printf("Starting controller...")
	ctx, cancel := context.WithCancel(context.Background())

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
		&NetworkPolicyResource,
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				c.syncNetworking(obj)
			},
			UpdateFunc: func(old, new interface{}) {
				ok, err := k8sutils.ShouldSync(old, new)
				if err != nil {
					cp := old.(*v1alpha1.NetworkPolicy).DeepCopy()
					log.Printf("Error syncing networkpolicy %s: %s:", cp.Name, err)
				}

				if ok {
					c.syncNetworking(new)
				}
			},
			DeleteFunc: func(obj interface{}) {
				cp := obj.(*v1alpha1.NetworkPolicy).DeepCopy()
				log.Printf("Deleting NetworkPolicy %s", cp.Name)
			},
		},
	)

	go watcher.Run(ctx.Done())
}

func (c *Controller) syncNetworking(obj interface{}) error {
	np := obj.(*v1alpha1.NetworkPolicy)

	ms, err := getMicroservice(c.patcher, np)
	if err != nil {
		log.Printf("Could not retrieve Microservice for %s: %s", np.Name, err)
		return err
	}

	releaseGroups := groupReleases(ms.Name, ms.Status.Releases)
	for name, releaseGroup := range releaseGroups {
		if err := syncReleaseGroup(c.patcher, ms, np, releaseGroup); err != nil {
			log.Printf("Error syncing release '%s': %s", name, err)
			continue
		}

		if err := syncSelectedRelease(c.patcher, ms, np, releaseGroup); err != nil {
			log.Printf("Error syncing selected release '%s': %s", name, err)
			continue
		}
	}

	if len(releaseGroups) == 0 {
		log.Printf("No release groups to sync")
	}

	return nil
}

type patchClient interface {
	Apply(runtime.Object, ...patcher.OptionFunc) ([]byte, error)
}

func syncReleaseGroup(cl patchClient, svc *v1alpha1.Microservice, np *v1alpha1.NetworkPolicy, releases []v1alpha1.Release) error {
	if len(np.Spec.Ports) != 0 {
		for _, release := range releases {
			svc, err := buildServiceForRelease(svc, np, &release, true)
			if err != nil {
				return err
			}

			// we don't always want a service for each release
			if svc == nil {
				return nil
			}

			_, err = cl.Apply(svc)
			return err
		}
	}

	return nil
}

func syncSelectedRelease(cl patchClient, ms *v1alpha1.Microservice, networkPolicy *v1alpha1.NetworkPolicy, releases []v1alpha1.Release) error {
	np := networkPolicy.DeepCopy()
	if len(np.Spec.ExternalDNS) == 0 {
		return nil
	}

	// TODO(jelmer): this should come from a factory based on the update strategy.
	releaser := &LatestReleaser{}

	name := np.Name

	externalRelease, err := releaser.ExternalRelease(releases)
	if err != nil {
		log.Printf("Could not get ExternalRelease for %s: %s", name, err)
		return err
	}

	svc, err := buildServiceForRelease(ms, np, externalRelease, false)
	if err != nil {
		log.Printf("Error creating service for release %s: %s", name, err)
		return err
	}

	if _, err := cl.Apply(svc); err != nil {
		log.Printf("Error syncing service for release %s: %s", name, err)
		return err
	}

	ing, err := buildIngressForRelease(ms, np, externalRelease)
	if err != nil {
		log.Printf("Error building Ingress for %s: %s", name, err)
		return err
	}

	if _, err := cl.Apply(ing); err != nil {
		log.Printf("Error syncing Ingress for release %s: %s", name, err)
		return err
	}

	status, err := buildNetworkStatusForRelease(np, externalRelease)
	if err != nil {
		log.Printf("Error building Network Status for release %s: %s", name, err)
		return err
	}

	// need to specify types again until we resolve the mapping issue
	np.TypeMeta = metav1.TypeMeta{
		Kind:       "NetworkPolicy",
		APIVersion: "hlnr.io/v1alpha1",
	}
	np.Status = status
	if _, err := cl.Apply(np); err != nil {
		log.Printf("Error syncing NetworkStatus for release %s: %s", name, err)
		return err
	}

	return nil
}

func groupReleases(name string, releases []v1alpha1.Release) map[string][]v1alpha1.Release {
	grouped := map[string][]v1alpha1.Release{}

	for _, release := range releases {
		key := release.FullName(name)
		if _, ok := grouped[key]; !ok {
			grouped[key] = []v1alpha1.Release{}
		}

		grouped[key] = append(grouped[key], release)
	}

	return grouped
}

type getClient interface {
	Get(interface{}, string, string) error
}

func getMicroservice(cl getClient, np *v1alpha1.NetworkPolicy) (*v1alpha1.Microservice, error) {
	ms := &v1alpha1.Microservice{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Microservice",
			APIVersion: "hlnr.io/v1alpha1",
		},
	}

	name := np.Name
	if np.Spec.Microservice != nil {
		name = np.Spec.Microservice.Name
	}

	if err := cl.Get(ms, np.Namespace, name); err != nil {
		return nil, err
	}

	return ms, nil
}
