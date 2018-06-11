package k8sutils

import (
	"encoding/base32"
	"strings"

	"github.com/dchest/blake2b"
)

var encoder = base32.HexEncoding.WithPadding(base32.NoPadding)

// ShortHash creates a shortened hash from the given string. The hash is
// lowercase base32 encoded, suitable for DNS use, and at most "len" characters
// long.
func ShortHash(data string, len int) string {
	b2b, _ := blake2b.New(&blake2b.Config{Size: uint8(len * 5 / 8)})
	b2b.Write([]byte(data))
	return strings.ToLower(encoder.EncodeToString(b2b.Sum(nil)))
}
