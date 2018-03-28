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

	if metaRaw, ok := data["metadata"]; ok {
		meta := metaRaw.(map[string]interface{})
		if annRaw, ok := meta["annotations"]; ok {
			ann := annRaw.(map[string]interface{})
			delete(ann, fmt.Sprintf("kubekit-%s/last-applied-configuration", name))

			meta["annotations"] = ann
			if len(ann) == 0 {
				delete(meta, "annotations")
			}
		}

		data["metadata"] = meta
		if len(meta) == 0 {
			delete(data, "metadata")
		}
	}

	return json.Marshal(data)
}
