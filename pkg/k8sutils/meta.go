package k8sutils

import (
	"github.com/jelmersnoeck/kubekit"
	"k8s.io/apimachinery/pkg/runtime"
)

// Annotations returns a set of annotations annotated with the Heighliner
// defaults.
func Annotations(ann map[string]string, version string, resource runtime.Object) map[string]string {
	if ann == nil {
		ann = map[string]string{}
	}

	ann["hglnr.io/version"] = version
	ann["hglnr.io/component"] = kubekit.TypeName(resource)
	return ann
}
