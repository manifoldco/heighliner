package svc

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/manifoldco/heighliner/pkg/api/v1alpha1"
	"github.com/manifoldco/heighliner/pkg/k8sutils"

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
	rc, err := kubekit.RESTClient(cfg, &v1alpha1.SchemeGroupVersion, v1alpha1.AddToScheme)
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

	imagePolicy, err := c.getImagePolicy(svc)
	if err != nil {
		log.Printf("Could not get ImagePolicy for Microservice %s: %s", svc.Name, err)
		return err
	}

	var deployedReleases []v1alpha1.Release
	for _, release := range imagePolicy.Status.Releases {
		vsvc, err := c.getVersionedMicroservice(svc, &release)
		if err != nil {
			log.Printf("Error generating the VersionedMicroservice object error=%s", err)
			// we don't need to return the error here, we want to be able to
			// deploy other releases still
			continue
		}

		patch, err := c.patcher.Apply(vsvc)
		if err != nil {
			log.Printf("Error applying VersionedMicroservice error=%s", err)
			// we don't need to return the error here, we want to be able to
			// deploy other releases still
			continue
		}

		patch, err = k8sutils.CleanupPatchAnnotations(patch, "hlnr-microservice")
		// doesn't matter if this errors, we just won't log the change if it
		// does
		if err == nil && !patcher.IsEmptyPatch(patch) {
			log.Printf("Synced Microservice %s %s with version %s", vsvc.Name, release.Name, release.Version)
		}

		deployedReleases = append(deployedReleases, release)
	}

	if err := deprecateReleases(c.patcher, svc, imagePolicy.Status.Releases); err != nil {
		log.Printf("Error deprecating releases for %s: %s", svc.Name, err)
		return err
	}

	// new release objects, store them
	if len(releaseDiff(svc.Status.Releases, deployedReleases)) > 0 {
		svc.Status.Releases = deployedReleases
		// need to specify types again until we resolve the mapping issue
		svc.TypeMeta = metav1.TypeMeta{
			Kind:       "Microservice",
			APIVersion: "hlnr.io/v1alpha1",
		}
		if _, err := c.patcher.Apply(svc); err != nil {
			log.Printf("Error syncing Microservice %s: %s", svc.Name, err)
			return err
		}
	}

	return nil
}

func (c *Controller) getVersionedMicroservice(crd *v1alpha1.Microservice, release *v1alpha1.Release) (*v1alpha1.VersionedMicroservice, error) {
	availabilityPolicySpec, err := c.getAvailabilityPolicySpec(crd)
	if err != nil {
		return nil, err
	}

	networkPolicySpec, err := c.getNetworkPolicySpec(crd)
	if err != nil {
		return nil, err
	}

	securityPolicySpec, err := c.getSecurityPolicySpec(crd)
	if err != nil {
		return nil, err
	}

	configPolicySpec, err := c.getConfigPolicySpec(crd)
	if err != nil {
		return nil, err
	}

	containers, err := c.getContainers(crd, release)
	if err != nil {
		return nil, err
	}

	annotations := crd.Annotations
	delete(annotations, "kubekit-hlnr-microservice/last-applied-configuration")
	delete(annotations, "kubectl.kubernetes.io/last-applied-configuration")

	name := fullReleaseName(crd.Name, release)
	labels := k8sutils.Labels(crd.Labels, crd.ObjectMeta)
	labels["hlnr.io/microservice.full_name"] = name
	labels["hlnr.io/microservice.name"] = crd.Name
	labels["hlnr.io/microservice.release"] = release.Name
	labels["hlnr.io/microservice.version"] = release.Version

	// TODO(jelmer): currently we need to specify the TypeMeta here. We need to
	// investigate a way to automate this depending on the passed in Object. The
	// issue lies within the passed in ClientSet. The ClientSet we've generated
	// is aware of the new types but this isn't used in the Factory that Kubekit
	// uses to pull out information.
	return &v1alpha1.VersionedMicroservice{
		TypeMeta: metav1.TypeMeta{
			Kind:       "VersionedMicroservice",
			APIVersion: "hlnr.io/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Annotations: annotations,
			Labels:      labels,
			Name:        name,
			Namespace:   crd.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(
					crd,
					v1alpha1.SchemeGroupVersion.WithKind(kubekit.TypeName(crd)),
				),
			},
		},
		Spec: v1alpha1.VersionedMicroserviceSpec{
			Availability: availabilityPolicySpec,
			Network:      networkPolicySpec,
			Config:       configPolicySpec,
			Security:     securityPolicySpec,
			Containers:   containers,
		},
	}, nil
}

func (c *Controller) getContainers(crd *v1alpha1.Microservice, release *v1alpha1.Release) ([]corev1.Container, error) {
	return []corev1.Container{
		{
			Name:            crd.Name,
			Image:           release.Image,
			ImagePullPolicy: corev1.PullIfNotPresent,
		},
	}, nil
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

	if len(imagePolicy.Status.Releases) == 0 {
		return nil, errors.New("Need a release to be set in the ImagePolicy Status")
	}

	return imagePolicy, nil
}

func (c *Controller) getAvailabilityPolicySpec(crd *v1alpha1.Microservice) (*v1alpha1.AvailabilityPolicySpec, error) {
	availabilityPolicy := &v1alpha1.AvailabilityPolicy{
		TypeMeta: metav1.TypeMeta{
			Kind:       "AvailabilityPolicy",
			APIVersion: "hlnr.io/v1alpha1",
		},
	}

	apName := crd.Spec.AvailabilityPolicy.Name
	if apName == "" {
		return nil, nil
	}

	if err := c.patcher.Get(availabilityPolicy, crd.Namespace, apName); err != nil {
		return nil, err
	}

	return &availabilityPolicy.Spec, nil
}

func (c *Controller) getNetworkPolicySpec(crd *v1alpha1.Microservice) (*v1alpha1.NetworkPolicySpec, error) {
	networkPolicy := &v1alpha1.NetworkPolicy{
		TypeMeta: metav1.TypeMeta{
			Kind:       "NetworkPolicy",
			APIVersion: "hlnr.io/v1alpha1",
		},
	}

	apName := crd.Spec.NetworkPolicy.Name
	if apName == "" {
		return nil, nil
	}

	if err := c.patcher.Get(networkPolicy, crd.Namespace, apName); err != nil {
		return nil, err
	}

	return &networkPolicy.Spec, nil
}

func (c *Controller) getConfigPolicySpec(crd *v1alpha1.Microservice) (*v1alpha1.ConfigPolicySpec, error) {
	configPolicy := &v1alpha1.ConfigPolicy{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ConfigPolicy",
			APIVersion: "hlnr.io/v1alpha1",
		},
	}

	apName := crd.Spec.ConfigPolicy.Name
	if apName == "" {
		return nil, nil
	}

	if err := c.patcher.Get(configPolicy, crd.Namespace, apName); err != nil {
		return nil, err
	}

	// TODO(jelmer): currently we're setting the last-updated annotation here.
	// This makes sure that the Deployment gets re-deployed when there is an
	// update to the configPolicy.
	// In the future, we'll roll out new versions on changes and we can drop
	// this annotation.
	crd.Annotations["hlnr-config-policy/last-updated"] = configPolicy.Status.LastUpdated.String()

	return &configPolicy.Spec, nil
}

func (c *Controller) getSecurityPolicySpec(crd *v1alpha1.Microservice) (*v1alpha1.SecurityPolicySpec, error) {
	securityPolicy := &v1alpha1.SecurityPolicy{
		TypeMeta: metav1.TypeMeta{
			Kind:       "SecurityPolicy",
			APIVersion: "hlnr.io/v1alpha1",
		},
	}

	apName := crd.Spec.SecurityPolicy.Name
	if apName == "" {
		return nil, nil
	}

	if err := c.patcher.Get(securityPolicy, crd.Namespace, apName); err != nil {
		return nil, err
	}

	return &securityPolicy.Spec, nil
}

type deleteClient interface {
	Delete(runtime.Object, ...patcher.OptionFunc) error
}

func deprecateReleases(cl deleteClient, crd *v1alpha1.Microservice, desired []v1alpha1.Release) error {
	deprecated := deprecatedReleases(desired, crd.Status.Releases)

	for _, release := range deprecated {
		name := fullReleaseName(crd.Name, &release)
		svc := &v1alpha1.VersionedMicroservice{
			TypeMeta: metav1.TypeMeta{
				Kind:       "VersionedMicroservice",
				APIVersion: "hlnr.io/v1alpha1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: name,
			},
		}

		if err := cl.Delete(svc); err != nil {
			log.Printf("Could not delete VersionedMicroservice %s: %s", name, err)
		}
	}

	return nil
}

func releaseDiff(desired, current []v1alpha1.Release) []v1alpha1.Release {
	// this could be done quicker/better but it's nice to reuse the same
	// function instead of implementing another algorithm.
	return append(deprecatedReleases(desired, current), deprecatedReleases(current, desired)...)
}

func deprecatedReleases(desired, current []v1alpha1.Release) []v1alpha1.Release {
	var deprecated []v1alpha1.Release

	desiredReleases := make([]string, len(desired))
	for i, release := range desired {
		desiredReleases[i] = release.String()
	}

CurrentReleaseLoop:
	for _, cRelease := range current {
		name := cRelease.String()
		for _, dRelease := range desiredReleases {
			if name == dRelease {
				continue CurrentReleaseLoop
			}
		}

		deprecated = append(deprecated, cRelease)
	}

	return deprecated
}

func fullReleaseName(crdName string, release *v1alpha1.Release) string {
	releaseName := ""
	if release.Name != crdName {
		releaseName = fmt.Sprintf("-%s", release.Name)
	}

	return strings.ToLower(fmt.Sprintf("%s%s-%s", crdName, releaseName, release.Version))
}
