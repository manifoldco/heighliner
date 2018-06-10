package networkpolicy

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/manifoldco/heighliner/pkg/api/v1alpha1"

	"github.com/jelmersnoeck/kubekit"
	"github.com/jelmersnoeck/kubekit/patcher"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
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

// NewController returns a new NetworkPolicy Controller.
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
				c.syncNetworking(new)
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
	if len(releaseGroups) == 0 {
		log.Printf("No release groups to sync")
		return nil
	}

	var newDomains []v1alpha1.Domain
	for name, releaseGroup := range releaseGroups {
		if err := syncReleaseGroup(c.cs, c.patcher, ms, np, releaseGroup); err != nil {
			log.Printf("Error syncing release '%s': %s", name, err)
			continue
		}

		domains, err := syncSelectedRelease(c.cs, c.patcher, ms, np, releaseGroup)
		if err != nil {
			log.Printf("Error syncing selected release '%s': %s", name, err)
			continue
		}

		newDomains = append(newDomains, domains...)
	}

	if statusDomainsEqual(np.Status.Domains, newDomains) {
		return nil
	}

	// XXX: kubekit needs these fields set, but they aren't there when coming
	// through the watcher. Can we fix the watcher, or teach kubekit to
	// introspect them?
	np.TypeMeta = metav1.TypeMeta{
		Kind:       "NetworkPolicy",
		APIVersion: "hlnr.io/v1alpha1",
	}

	np.Status.Domains = newDomains
	if _, err := c.patcher.Apply(np); err != nil {
		log.Printf("Error syncing NetworkStatus %s: %s", np.Name, err)
		return err
	}

	return nil
}

type patchClient interface {
	getClient
	Apply(runtime.Object, ...patcher.OptionFunc) ([]byte, error)
}

func syncReleaseGroup(cs kubernetes.Interface, cl patchClient, svc *v1alpha1.Microservice, np *v1alpha1.NetworkPolicy, releases []v1alpha1.Release) error {
	if len(np.Spec.Ports) != 0 {
		for _, release := range releases {
			_, err := createOrReplaceService(cs, cl, svc, np, &release, release.FullName(svc.Name))
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func syncSelectedRelease(cs kubernetes.Interface, cl patchClient, ms *v1alpha1.Microservice, networkPolicy *v1alpha1.NetworkPolicy, releases []v1alpha1.Release) ([]v1alpha1.Domain, error) {
	np := networkPolicy.DeepCopy()

	// TODO(jelmer): this should come from a factory based on the update strategy.
	releaser := &LatestReleaser{}

	name := np.Name

	externalRelease, err := releaser.ExternalRelease(releases)
	if err != nil {
		log.Printf("Could not get ExternalRelease for %s: %s", name, err)
		return nil, err
	}

	srv, err := createOrReplaceService(cs, cl, ms, np, externalRelease, ms.Name)
	if err != nil {
		return nil, err
	}

	ing, err := buildIngressForRelease(ms, np, externalRelease, srv)
	if err != nil {
		log.Printf("Error building Ingress for %s: %s", name, err)
		return nil, err
	}

	if _, err := cl.Apply(ing); err != nil {
		log.Printf("Error syncing Ingress for release %s: %s", name, err)
		return nil, err
	}

	return buildNetworkStatusDomainsForRelease(ms, np, externalRelease)
}

// createOrReplaceService will either create a new service instance, or do a full
// replacement of one if it exists, performing changes on the existing one.
// it does not use PATCH, as we need to fully replace the OwnerReferences.
func createOrReplaceService(cs kubernetes.Interface, cl patchClient, svc *v1alpha1.Microservice, np *v1alpha1.NetworkPolicy, release *v1alpha1.Release, srvName string) (*v1.Service, error) {
	if len(np.Spec.Ports) == 0 {
		return nil, nil
	}

	srv := &v1.Service{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Service",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      srvName,
			Namespace: svc.Namespace,
		},
	}

	var create bool
	err := cl.Get(srv, srv.Namespace, srv.Name) // ignore errors, and let the PUT fail
	switch {
	case errors.IsNotFound(err):
		create = true
	case err != nil:
		log.Printf("Error fetching service for release %s: %s", np.Name, err)
		return nil, err
	}
	srv = buildServiceForRelease(srv, svc, np, release)

	if create {
		srv, err = cs.Core().Services(srv.Namespace).Create(srv)
	} else {
		srv, err = cs.Core().Services(srv.Namespace).Update(srv)
	}
	if err != nil {
		log.Printf("Error syncing service for release %s: %s", np.Name, err)
	}

	return srv, err
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
