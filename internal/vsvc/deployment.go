package vsvc

import (
	"github.com/manifoldco/heighliner/internal/api/v1alpha1"
	"github.com/manifoldco/heighliner/internal/k8sutils"
	"github.com/manifoldco/heighliner/internal/meta"

	"github.com/jelmersnoeck/kubekit"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// getDeployment creates the Deployment Object for a VersionedMicroservice.
func getDeployment(crd *v1alpha1.VersionedMicroservice) (runtime.Object, error) {
	availability := crd.Spec.Availability
	if availability == nil {
		availability = &v1alpha1.DefaultAvailabilityPolicySpec
	}

	labels := meta.Labels(crd.Labels, crd)
	annotations := meta.Annotations(crd.Annotations, v1alpha1.Version, crd)

	affinity := availability.Affinity
	if affinity == nil {
		affinity = DefaultAffinity("hlnr.io/service", crd.Name)
	}

	security := crd.Spec.Security
	if security == nil {
		security = &v1alpha1.SecurityPolicySpec{}
	}

	populateContainers(crd)

	dpl := &v1beta1.Deployment{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Deployment",
			APIVersion: "extensions/v1beta1",
		},
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
					ImagePullSecrets:             crd.Spec.ImagePullSecrets,
					ServiceAccountName:           security.ServiceAccountName,
					AutomountServiceAccountToken: k8sutils.PtrBool(security.AutomountServiceAccountToken),
					SecurityContext:              security.SecurityContext,
					Affinity:                     affinity,
					RestartPolicy:                availability.RestartPolicy,
					Containers:                   crd.Spec.Containers,
					Volumes:                      podVolumes(crd),
				},
			},
		},
	}

	return dpl, nil
}

func populateContainers(crd *v1alpha1.VersionedMicroservice) {
	if crd.Spec.Config == nil {
		return
	}

	for i, container := range crd.Spec.Containers {
		container.VolumeMounts = crd.Spec.Config.VolumeMounts
		container.EnvFrom = crd.Spec.Config.EnvFrom
		container.Env = crd.Spec.Config.Env
		container.Args = crd.Spec.Config.Args
		container.Command = crd.Spec.Config.Command

		// reassign the container in the CRD
		crd.Spec.Containers[i] = container
	}

	if crd.Spec.Security == nil {
		crd.Spec.Security = &v1alpha1.SecurityPolicySpec{}
	}
}

func podVolumes(crd *v1alpha1.VersionedMicroservice) []corev1.Volume {
	if crd.Spec.Config == nil {
		return nil
	}

	return crd.Spec.Config.Volumes
}
