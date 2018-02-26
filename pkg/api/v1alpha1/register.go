package v1alpha1

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
)

const (
	// GroupName defines the name of the group we'll use for our components.
	GroupName = "hglnr.io"

	// Version defines the version of this API.
	Version = "v1alpha1"
)

// SchemeGroupVersion is the Group Version used for this scheme.
var SchemeGroupVersion = schema.GroupVersion{Group: GroupName, Version: Version}
