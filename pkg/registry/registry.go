package registry

// Registry represents the interface any registry needs to provide to query it.
type Registry interface {
	Ping() error
	Tags(string, int) ([]string, error)
}
