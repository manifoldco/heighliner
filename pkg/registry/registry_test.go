package registry

import (
	"errors"
	"testing"
)

func TestIsTagNotFoundError(t *testing.T) {
	t.Run("ok", func(t *testing.T) {
		err := NewTagNotFoundError("arigato/beans", "1.0.0")

		if !IsTagNotFoundError(err) {
			t.Error("Error not reported as tag not found")
		}
	})

	t.Run("not ok", func(t *testing.T) {
		err := errors.New("fake")

		if IsTagNotFoundError(err) {
			t.Error("Error incorrectly reported as tag not found")
		}
	})
}
