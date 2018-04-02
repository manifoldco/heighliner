package k8sutils

import (
	"encoding/json"
	"fmt"
)

// CleanupPatchAnnotations cleans up a patch diff to remove the kubekit
// annotations.
// This is useful for when a patch is applied and we don't want to print the
// annotations but just the actual diff.
func CleanupPatchAnnotations(patch []byte, name string) ([]byte, error) {
	data := map[string]interface{}{}
	if err := json.Unmarshal(patch, &data); err != nil {
		fmt.Println(err)
		return nil, err
	}

	data = cleanKeys(data, fmt.Sprintf("kubekit-%s/last-applied-configuration", name), "status", "$retainKeys")
	return json.Marshal(data)
}

// cleanKeys is a recursive function which cleans specific keys from a nested
// map.
func cleanKeys(data map[string]interface{}, keys ...string) map[string]interface{} {
	keyData := map[string]interface{}{}

	for k, v := range data {
		if cleanupKey(k, keys...) {
			continue
		}

		valueData := v
		if rawData, ok := v.(map[string]interface{}); ok {
			mappedData := cleanKeys(rawData, keys...)
			if len(mappedData) == 0 {
				continue
			}

			valueData = mappedData
		}

		if valueData != nil {
			keyData[k] = valueData
		}
	}

	return keyData
}

func cleanupKey(key string, keys ...string) bool {
	for _, k := range keys {
		if k == key {
			return true
		}
	}

	return false
}
