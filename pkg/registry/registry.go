package registry

// Registry represents the interface any registry needs to provide to query it.
type Registry interface {
	GetManifest(string, string) (bool, error)
}
