package svc

import (
	"context"
	"errors"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/manifoldco/heighliner/pkg/api/v1alpha1"
	"github.com/manifoldco/heighliner/pkg/k8sutils"

	"github.com/jelmersnoeck/kubekit"
	"github.com/jelmersnoeck/kubekit/patcher"
	corev1 "k8s.io/api/core/v1"
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

// NewController returns a new Microservice Controller.
func NewController(cfg *rest.Config, cs kubernetes.Interface, namespace string) (*Controller, error) {
	rc, err := kubekit.RESTClient(cfg, &v1alpha1.SchemeGroupVersion, AddToScheme)
	if err != nil {
		return nil, err
	}

	return &Controller{
		cs:        cs,
		rc:        rc,
		namespace: namespace,
		patcher:   patcher.New("hlnr-microservice", cmdutil.NewFactory(nil)),
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
				c.patchMicroservice(obj)
			},
			UpdateFunc: func(old, new interface{}) {
				c.patchMicroservice(new)
			},
			DeleteFunc: func(obj interface{}) {
				svc := obj.(*v1alpha1.Microservice).DeepCopy()
				log.Printf("Deleting Microservice %s", svc.Name)
			},
		},
	)

	go watcher.Run(ctx.Done())
}

func (c *Controller) patchMicroservice(obj interface{}) error {
	svc := obj.(*v1alpha1.Microservice).DeepCopy()

	vsvc, err := c.getVersionedMicroservice(svc)
	if err != nil {
		log.Printf("Error generating the VersionedMicroservice object error=%s", err)
		return err
	}

	patch, err := c.patcher.Apply(vsvc)
	if err != nil {
		log.Printf("Error applying VersionedMicroservice error=%s", err)
		return err
	}

	if !patcher.IsEmptyPatch(patch) {
		cleanedPatch, err := k8sutils.CleanupPatchAnnotations(patch, "hlnr-microservice")
		if err != nil {
			cleanedPatch = patch
		}
		log.Printf("Synced Microservice %s with new data: %s", svc.Name, string(cleanedPatch))
	}

	return nil
}

func (c *Controller) getVersionedMicroservice(crd *v1alpha1.Microservice) (*v1alpha1.VersionedMicroservice, error) {
	imagePolicy, err := c.getImagePolicy(crd)
	if err != nil {
		return nil, err
	}

	// TODO(jelmer): currently we need to specify the TypeMeta here. We need to
	// investigate a way to automate this depending on the passed in Object. The
	// issue lies within the passed in ClientSet. The ClientSet we've generated
	// is aware of the new types but this isn't used in the Factory that Kubekit
	// uses to pull out information.
	vsvc := &v1alpha1.VersionedMicroservice{
		TypeMeta: metav1.TypeMeta{
			Kind:       "VersionedMicroservice",
			APIVersion: "hlnr.io/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Annotations: crd.Annotations,
			Labels:      crd.Labels,
			Name:        crd.Name,
			Namespace:   crd.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(
					crd,
					v1alpha1.SchemeGroupVersion.WithKind(kubekit.TypeName(crd)),
				),
			},
		},
		Spec: v1alpha1.VersionedMicroserviceSpec{
			Containers: []corev1.Container{
				{
					Name:  crd.Name,
					Image: imagePolicy.Status.Image,
				},
			},
		},
	}
	return vsvc, nil
}

func (c *Controller) getImagePolicy(crd *v1alpha1.Microservice) (*v1alpha1.ImagePolicy, error) {
	imagePolicy := &v1alpha1.ImagePolicy{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ImagePolicy",
			APIVersion: "hlnr.io/v1alpha1",
		},
	}

	ipName := crd.Spec.ImagePolicy.Name
	if err := c.patcher.Get(imagePolicy, crd.Namespace, ipName); err != nil {
		return nil, err
	}

	if imagePolicy.Status.Image == "" {
		return nil, errors.New("Need an image to be set in the ImagePolicy Status")
	}

	return imagePolicy, nil
}
