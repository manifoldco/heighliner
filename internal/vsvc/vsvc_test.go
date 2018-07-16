package vsvc_test

import (
	"testing"

	"github.com/manifoldco/heighliner/apis/v1alpha1"
	"github.com/manifoldco/heighliner/internal/vsvc"

	"github.com/jelmersnoeck/kubekit/kubetest"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apiextensions-apiserver/pkg/apiserver/validation"
)

func TestCRD_VersionedMicroservice_Containers(t *testing.T) {
	validator, err := kubetest.GetValidator(vsvc.CustomResource)
	if err != nil {
		t.Fatalf("Couldn't get validator: %s", err)
	}

	t.Run("without any containers specified", func(t *testing.T) {
		crd := &v1alpha1.VersionedMicroservice{
			Spec: v1alpha1.VersionedMicroserviceSpec{
				Containers: []corev1.Container{},
			},
		}
		if err := validation.ValidateCustomResource(crd, validator); err == nil {
			t.Errorf("Expected error, got none")
		}
	})

	t.Run("with a single container specified", func(t *testing.T) {
		// TODO(jelmer): figure out a way to pull in the k8s validation for
		// these objects and make sure they're validated as well.
		crd := &v1alpha1.VersionedMicroservice{
			Spec: v1alpha1.VersionedMicroserviceSpec{
				Containers: []corev1.Container{
					{},
				},
			},
		}
		if err := validation.ValidateCustomResource(crd, validator); err != nil {
			t.Errorf("Expected no error, got '%s'", err)
		}
	})
}
