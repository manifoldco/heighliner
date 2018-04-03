package k8sutils

import (
	"encoding/json"

	"k8s.io/apimachinery/pkg/util/intstr"
)

// PtrBool converts a boolean value to a pointer of that boolean value.
func PtrBool(b bool) *bool {
	return &b
}

// PtrIntOrString converts a value of the intstr.IntOrString value to a pointer
// of that value.
func PtrIntOrString(i intstr.IntOrString) *intstr.IntOrString {
	return &i
}

// PtrInt64 converts a value of int64 to the pointer of that value.
func PtrInt64(i int64) *int64 {
	return &i
}

// JSONBytes converts an interface value to a set of bytes encoded as JSON.
func JSONBytes(val interface{}) []byte {
	bts, err := json.Marshal(val)
	if err != nil {
		panic(err)
	}

	return bts
}
