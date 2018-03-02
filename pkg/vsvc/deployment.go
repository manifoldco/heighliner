package vsvc

import (
	"github.com/manifoldco/heighliner/pkg/api/v1alpha1"
	"github.com/manifoldco/heighliner/pkg/k8sutils"

	"github.com/jelmersnoeck/kubekit"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// getDeployment creates the Deployment Object for a VersionedMicroservice.
func getDeployment(crd *v1alpha1.VersionedMicroservice) (*v1beta1.Deployment, error) {
	availability := crd.Spec.Availability
	if availability == nil {
		availability = &v1alpha1.DefaultAvailabilitySpec
	}

	labels := k8sutils.Labels(crd.Labels, crd.ObjectMeta)
	annotations := k8sutils.Annotations(crd.Annotations, v1alpha1.Version, crd)

	affinity := availability.Affinity
	if affinity == nil {
		affinity = DefaultAffinity("hlnr.io/service", crd.Name)
	}

	dpl := &v1beta1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:        crd.Name,
			Namespace:   crd.Namespace,
			Labels:      labels,
			Annotations: annotations,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(
					crd,
					v1alpha1.SchemeGroupVersion.WithKind(kubekit.TypeName(crd)),
				),
			},
		},
		Spec: v1beta1.DeploymentSpec{
			Replicas: availability.Replicas,
			Strategy: availability.DeploymentStrategy,
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:        crd.Name,
					Namespace:   crd.Namespace,
					Labels:      labels,
					Annotations: annotations,
				},
				Spec: corev1.PodSpec{
					// TODO(jelmer) make this configurable through a security
					// policy
					AutomountServiceAccountToken: func(b bool) *bool { return &b }(false),
					Affinity:                     affinity,
					RestartPolicy:                availability.RestartPolicy,
					Containers:                   crd.Spec.Containers,
					Volumes:                      crd.Spec.Volumes,
				},
			},
		},
	}

	return dpl, nil
}
