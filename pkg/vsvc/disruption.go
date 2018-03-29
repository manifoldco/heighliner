package vsvc

import (
	"errors"

	"github.com/manifoldco/heighliner/pkg/api/v1alpha1"
	"github.com/manifoldco/heighliner/pkg/k8sutils"

	"github.com/jelmersnoeck/kubekit"
	"k8s.io/api/policy/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
)

var (
	// ErrMinMaxAvailabilitySet is used when the Availability Configuration has
	// both MinAvailabe and MaxUnavailable set.
	ErrMinMaxAvailabilitySet = errors.New("Can't have both MinAvailable and MaxUnavailable configured")
)

func getPodDisruptionBudget(crd *v1alpha1.VersionedMicroservice) (runtime.Object, error) {
	budget := defaultDisruptionBudget.DeepCopy()

	labels := k8sutils.Labels(crd.Labels, crd.ObjectMeta)
	annotations := k8sutils.Annotations(crd.Annotations, v1alpha1.Version, crd)

	budget.ObjectMeta = metav1.ObjectMeta{
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
	}
	budget.Spec.Selector.MatchLabels[k8sutils.LabelServiceKey] = crd.Name
	if crd.Spec.Availability != nil {
		av := crd.Spec.Availability

		if av.MinAvailable != nil && av.MaxUnavailable != nil {
			return nil, ErrMinMaxAvailabilitySet
		}

		if av.MinAvailable != nil && av.MaxUnavailable == nil {
			budget.Spec.MinAvailable = av.MinAvailable
			budget.Spec.MaxUnavailable = nil
		}
		if av.MaxUnavailable != nil && av.MinAvailable == nil {
			budget.Spec.MaxUnavailable = av.MaxUnavailable
			budget.Spec.MinAvailable = nil
		}
	}

	return budget, nil
}

var defaultDisruptionBudget = &v1beta1.PodDisruptionBudget{
	TypeMeta: metav1.TypeMeta{
		Kind:       "PodDisruptionBudget",
		APIVersion: "policy/v1beta1",
	},
	Spec: v1beta1.PodDisruptionBudgetSpec{
		MinAvailable: ptrIntOrStringFromInt(1),
		Selector: &metav1.LabelSelector{
			MatchLabels: map[string]string{},
		},
	},
	Status: v1beta1.PodDisruptionBudgetStatus{
		DisruptedPods: map[string]metav1.Time{},
	},
}

func ptrIntOrStringFromInt(i int) *intstr.IntOrString {
	return ptrIntOrString(intstr.FromInt(i))
}

func ptrIntOrString(i intstr.IntOrString) *intstr.IntOrString {
	return &i
}
