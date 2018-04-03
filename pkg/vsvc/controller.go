package vsvc

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/manifoldco/heighliner/pkg/api/v1alpha1"
	"github.com/manifoldco/heighliner/pkg/k8sutils"

	"github.com/jelmersnoeck/kubekit"
	"github.com/jelmersnoeck/kubekit/errors"
	"github.com/jelmersnoeck/kubekit/patcher"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	cmdutil "k8s.io/kubernetes/pkg/kubectl/cmd/util"
)

// Controller represents the VersionedMicroserviceController. This controller
// takes care of creating, updating and deleting lower level Kubernetese
// components that are associated with a specific VersionedMicroservice.
type Controller struct {
	rc        *rest.RESTClient
	cs        kubernetes.Interface
	namespace string
	patcher   *patcher.Patcher
}

// NewController returns a new VersionedMicroservice Controller.
func NewController(cfg *rest.Config, cs kubernetes.Interface, namespace string) (*Controller, error) {
	rc, err := kubekit.RESTClient(cfg, &v1alpha1.SchemeGroupVersion, v1alpha1.AddToScheme)
	if err != nil {
		return nil, err
	}

	return &Controller{
		cs:        cs,
		rc:        rc,
		namespace: namespace,
		patcher:   patcher.New("hlnr-versioned-microservice", cmdutil.NewFactory(nil)),
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
		&CustomResource,
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				c.applyCRD(obj, patcher.DisableUpdate())
			},
			UpdateFunc: func(old, new interface{}) {
				c.applyCRD(new, patcher.DisableCreate())
			},
			DeleteFunc: func(obj interface{}) {
				svc := obj.(*v1alpha1.VersionedMicroservice).DeepCopy()
				log.Printf("Deleting VersionedMicroservice %s", svc.Name)
			},
		},
	)

	go watcher.Run(ctx.Done())
}

func (c *Controller) applyCRD(obj interface{}, opts ...patcher.OptionFunc) error {
	vsvc := obj.(*v1alpha1.VersionedMicroservice).DeepCopy()

	// This is managed by a kubekit controller, lets remove that annotation.
	delete(vsvc.Annotations, "kubekit-hlnr-microservice/last-applied-configuration")

	if err := updateObject("Deployment", vsvc, c.patcher, getDeployment); err != nil {
		return err
	}

	if err := updateObject("Service", vsvc, c.patcher, getService); err != nil {
		return err
	}

	if err := updateObject("Ingress", vsvc, c.patcher, getIngress); err != nil && !errors.IsNoObjectGiven(err) {
		return err
	}

	if err := updateObject("PodDisruptionBudget", vsvc, c.patcher, getPodDisruptionBudget, patcher.WithDeleteFirst()); err != nil && !errors.IsNoObjectGiven(err) {
		return err
	}

	return nil
}

type objectFunc func(*v1alpha1.VersionedMicroservice) (runtime.Object, error)

// TODO: these errors should be logged with glog and at a higher level so we can
// cut down noise.
func updateObject(name string, vsvc *v1alpha1.VersionedMicroservice, p *patcher.Patcher, f objectFunc, opts ...patcher.OptionFunc) error {
	obj, err := f(vsvc)
	if err != nil {
		log.Printf("Could not configure %s for %s: %s", name, vsvc.Name, err)
		return err
	}

	patch, err := p.Apply(obj, opts...)
	if err != nil && !errors.IsNoObjectGiven(err) {
		log.Printf("Could not apply %s for %s: %s", name, vsvc.Name, err)
		return err
	}

	patch, _ = k8sutils.CleanupPatchAnnotations(patch, "hlnr-versioned-microservice")
	patch, err = k8sutils.CleanupPatchAnnotations(patch, "hlnr-microservice")
	if err == nil && !patcher.IsEmptyPatch(patch) {
		log.Printf("Synced %s %s with new data: %s", name, vsvc.Name, string(patch))
	}

	return err
}
