package vsvc

import (
	"testing"

	"github.com/manifoldco/heighliner/apis/v1alpha1"
	"github.com/manifoldco/heighliner/internal/meta"

	"github.com/stretchr/testify/assert"
	"k8s.io/api/policy/v1beta1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func TestGetPodDisruptionBudget(t *testing.T) {
	resultFunc := func(t *testing.T, crd *v1alpha1.VersionedMicroservice, min, max *intstr.IntOrString) {
		obj, err := getPodDisruptionBudget(crd)
		pdb := obj.(*v1beta1.PodDisruptionBudget)
		assert.NoError(t, err)
		assert.Equal(t, min, pdb.Spec.MinAvailable)
		assert.Equal(t, max, pdb.Spec.MaxUnavailable)
		assert.Equal(t, crd.Name, pdb.Spec.Selector.MatchLabels[meta.LabelServiceKey])
	}

	t.Run("without config", func(t *testing.T) {
		crd := &v1alpha1.VersionedMicroservice{}
		crd.Name = "test-app"

		resultFunc(t, crd, ptrIntOrStringFromInt(1), nil)
	})

	t.Run("with minAvailable configured", func(t *testing.T) {
		crd := &v1alpha1.VersionedMicroservice{
			Spec: v1alpha1.VersionedMicroserviceSpec{
				Availability: &v1alpha1.AvailabilityPolicySpec{
					MinAvailable: ptrIntOrStringFromInt(5),
				},
			},
		}
		crd.Name = "my-test"

		resultFunc(t, crd, ptrIntOrStringFromInt(5), nil)
	})

	t.Run("with maxUnavailable configured", func(t *testing.T) {
		crd := &v1alpha1.VersionedMicroservice{
			Spec: v1alpha1.VersionedMicroserviceSpec{
				Availability: &v1alpha1.AvailabilityPolicySpec{
					MaxUnavailable: ptrIntOrStringFromInt(2),
				},
			},
		}
		crd.Name = "unavailable-test"

		resultFunc(t, crd, nil, ptrIntOrStringFromInt(2))
	})

	t.Run("with both values configured", func(t *testing.T) {
		crd := &v1alpha1.VersionedMicroservice{
			Spec: v1alpha1.VersionedMicroserviceSpec{
				Availability: &v1alpha1.AvailabilityPolicySpec{
					MaxUnavailable: ptrIntOrStringFromInt(2),
					MinAvailable:   ptrIntOrStringFromInt(2),
				},
			},
		}
		crd.Name = "invalid"
		_, err := getPodDisruptionBudget(crd)
		assert.Equal(t, ErrMinMaxAvailabilitySet, err)
	})
}
