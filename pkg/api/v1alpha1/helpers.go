package v1alpha1

import (
	"encoding/json"

	"k8s.io/apimachinery/pkg/util/intstr"
)

func ptrIntOrString(i intstr.IntOrString) *intstr.IntOrString {
	return &i
}

func ptrInt64(i int64) *int64 {
	return &i
}

func jsonBytes(val interface{}) []byte {
	bts, err := json.Marshal(val)
	if err != nil {
		panic(err)
	}

	return bts
}
