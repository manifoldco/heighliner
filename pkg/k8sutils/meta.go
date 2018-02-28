package k8sutils

import (
	"flag"

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

func init() {
	// we're getting a lot of errors about logging before flag parsing, this
	// should resolve that. Seeing that it's a common package, we don't need to
	// include this for every controller.
	flag.Parse()
}
