package k8sutils

const (
	// LabelServiceKey is used to annotate the application with a specific
	// service key set by Heighliner. This way we always have at least one label
	// available for LabelSelectors.
	LabelServiceKey = "hlnr.io/service"
)
