package meta

import (
	"github.com/jelmersnoeck/kubekit"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/manifoldco/heighliner/pkg/api/v1alpha1"
)

// Annotations returns a set of annotations annotated with the Heighliner
// defaults.
func Annotations(ann map[string]string, version string, resource runtime.Object) map[string]string {
	if ann == nil {
		ann = map[string]string{}
	}

	ann["hlnr.io/version"] = version
	ann["hlnr.io/component"] = kubekit.TypeName(resource)
	return ann
}

// Labels returns a new set of labels annotated with Heighliner specific
// defaults.
func Labels(labels map[string]string, m metav1.Object) map[string]string {
	if labels == nil {
		labels = map[string]string{}
	}

	labels[LabelServiceKey] = m.GetName()
	return labels
}

// MicroserviceLabels returns a new set of labels annotated with Heighliner specific
// defaults (as from Label), and release specific values.
func MicroserviceLabels(ms *v1alpha1.Microservice, r *v1alpha1.Release, parent metav1.Object) map[string]string {
	labels := Labels(parent.GetLabels(), parent)

	labels["hlnr.io/microservice.full_name"] = r.FullName(ms.Name)
	labels["hlnr.io/microservice.name"] = ms.Name
	labels["hlnr.io/microservice.release"] = r.Name()
	labels["hlnr.io/microservice.version"] = r.Version()

	return labels
}
