package k8sutils

import (
	"flag"

	"github.com/jelmersnoeck/kubekit"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
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
func Labels(labels map[string]string, m metav1.ObjectMeta) map[string]string {
	if labels == nil {
		labels = map[string]string{}
	}

	labels[LabelServiceKey] = m.Name
	return labels
}

func init() {
	// we're getting a lot of errors about logging before flag parsing, this
	// should resolve that. Seeing that it's a common package, we don't need to
	// include this for every controller.
	flag.Parse()
}
