package registry

import "fmt"

// Registry represents the interface any registry needs to provide to query it.
type Registry interface {
	TagFor(string, string) (string, error)
}

type tagNotFoundError string

func (t tagNotFoundError) Error() string { return string(t) }

// NewTagNotFoundError returns an error that satisfies IsTagNotFoundError.
func NewTagNotFoundError(repository, release string) error {
	return tagNotFoundError(fmt.Sprintf("no suitable tag was found in '%s' for '%s'", repository, release))
}

// IsTagNotFoundError returns a bool indicating if the provided error is for
// no matching tag being found.
func IsTagNotFoundError(err error) bool {
	_, ok := err.(tagNotFoundError)
	return ok
}
