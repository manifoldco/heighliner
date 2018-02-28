package k8sutils

import (
	"log"
	"reflect"
	"time"

	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	apiextv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/apimachinery/pkg/util/wait"
)

// CustomResource describes all the values we need to create a CRD from scratch
// and register it with the Kubernetes API.
type CustomResource struct {
	Name    string
	Plural  string
	Group   string
	Version string
	Aliases []string
	Scope   v1beta1.ResourceScope
	Object  runtime.Object
}

// FullName returns the full name of the CustomResource.
func (c CustomResource) FullName() string {
	return c.Plural + "." + c.Group
}

// Kind returns the Type Name of the CR Object.
func (c CustomResource) Kind() string {
	return ObjectName(c.Object)
}

// Definition returns the CustomResourceDefinition that is linked to this
// CustomResource.
func (c CustomResource) Definition() *apiextv1beta1.CustomResourceDefinition {
	return &apiextv1beta1.CustomResourceDefinition{
		ObjectMeta: metav1.ObjectMeta{
			Name: c.FullName(),
		},
		Spec: apiextv1beta1.CustomResourceDefinitionSpec{
			Group:   c.Group,
			Version: c.Version,
			Scope:   c.Scope,
			Names: apiextv1beta1.CustomResourceDefinitionNames{
				Plural:     c.Plural,
				ShortNames: c.Aliases,
				Kind:       c.Kind(),
			},
		},
	}
}

// CreateCRD creates and registers a CRD with the k8s cluster.
func CreateCRD(cs clientset.Interface, c CustomResource) error {
	crd := c.Definition()

	if err := createCRD(cs, crd); err != nil {
		return err
	}

	return waitForCRD(cs, c.FullName(), crd)
}

func createCRD(cs clientset.Interface, crd *apiextv1beta1.CustomResourceDefinition) error {
	_, err := cs.ApiextensionsV1beta1().CustomResourceDefinitions().Create(crd)
	if apierrors.IsAlreadyExists(err) {
		return nil
	} else if err != nil {
		return err
	}

	return nil
}

func waitForCRD(cs clientset.Interface, fullName string, crd *apiextv1beta1.CustomResourceDefinition) error {
	err := wait.Poll(500*time.Millisecond, 60*time.Second, func() (bool, error) {
		crd, err := cs.ApiextensionsV1beta1().CustomResourceDefinitions().Get(
			fullName,
			metav1.GetOptions{},
		)
		if err != nil {
			return false, err
		}

		for _, cond := range crd.Status.Conditions {
			switch cond.Type {
			case apiextv1beta1.Established:
				if cond.Status == apiextv1beta1.ConditionTrue {
					return true, err
				}
			case apiextv1beta1.NamesAccepted:
				if cond.Status == apiextv1beta1.ConditionFalse {
					log.Printf("Name conflict: %v\n", cond.Reason)
				}
			}
		}
		return false, err
	})

	if err != nil {
		deleteErr := cs.ApiextensionsV1beta1().CustomResourceDefinitions().Delete(fullName, nil)
		if deleteErr != nil {
			return errors.NewAggregate([]error{err, deleteErr})
		}
		return err
	}

	return err
}

// ObjectName returns the CamelCased name of a given object.
func ObjectName(object interface{}) string {
	val := reflect.ValueOf(object)

	var name string
	switch val.Kind() {
	case reflect.Ptr:
		name = val.Elem().Type().Name()
	default:
		name = val.Type().Name()
	}

	return name
}
