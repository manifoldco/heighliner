package k8sutils

import (
	"fmt"

	"github.com/dchest/blake2b"
)

// ShortHash creates a shortened hash from the given string.
func ShortHash(data string, len int) string {
	b2b, _ := blake2b.New(&blake2b.Config{Size: uint8(len)})
	b2b.Write([]byte(data))
	return fmt.Sprintf("%x", b2b.Sum(nil))
}
