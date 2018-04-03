package configpolicy

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/manifoldco/heighliner/pkg/api/v1alpha1"
	"github.com/manifoldco/heighliner/pkg/k8sutils"

	"github.com/jelmersnoeck/kubekit"
	"github.com/jelmersnoeck/kubekit/patcher"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
		patcher:   patcher.New("hlnr-configpolicy", cmdutil.NewFactory(nil)),
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
		&ConfigPolicyResource,
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				c.hashConfigValues(obj)
			},
			UpdateFunc: func(old, new interface{}) {
				c.hashConfigValues(new)
			},
			DeleteFunc: func(obj interface{}) {
				cp := obj.(*v1alpha1.ConfigPolicy).DeepCopy()
				log.Printf("Deleting ConfigPolicy %s", cp.Name)
			},
		},
	)

	go watcher.Run(ctx.Done())
}

func (c *Controller) hashConfigValues(obj interface{}) error {
	cp := obj.(*v1alpha1.ConfigPolicy).DeepCopy()

	hashedConfig, err := c.getHashedConfig(cp)
	if err != nil {
		log.Printf("Error getting hashed configuration for %s: %s", cp.Name, err)
		return err
	}
	hashedString := fmt.Sprintf("%x", hashedConfig)

	// some values of our config have changed, update the CRD status so
	// depending resources get notified.
	if hashedString != cp.Status.Hashed {
		cp.Status.LastUpdated = metav1.Now()
		cp.Status.Hashed = hashedString

		cp.TypeMeta = metav1.TypeMeta{
			Kind:       "ConfigPolicy",
			APIVersion: "hlnr.io/v1alpha1",
		}
		patch, err := c.patcher.Apply(cp)
		if err != nil {
			log.Printf("Could not update ConfigPolicy %s: %s", cp.Name, err)
		}

		patch, err = k8sutils.CleanupPatchAnnotations(patch, "hlnr-configpolicy")
		if err == nil && !patcher.IsEmptyPatch(patch) {
			log.Printf("Updated ConfigPolicy %s", cp.Name)
		}
	}

	return nil
}

func (c *Controller) getHashedConfig(crd *v1alpha1.ConfigPolicy) ([]byte, error) {
	envVarHash, err := getEnvVarHash(c.patcher, crd.Namespace, crd.Spec.Env)
	if err != nil {
		return nil, err
	}

	envFromSourceHash, err := getEnvFromSourceHash(c.patcher, crd.Namespace, crd.Spec.EnvFrom)
	if err != nil {
		return nil, err
	}

	return append(envVarHash, envFromSourceHash...), nil
}
