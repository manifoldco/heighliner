package k8sutils

import "flag"

func init() {
	// we're getting a lot of errors about logging before flag parsing, this
	// should resolve that. Seeing that it's a common package, we don't need to
	// include this for every controller.
	flag.Parse()
}
