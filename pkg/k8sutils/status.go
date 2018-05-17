package k8sutils

import (
	"encoding/json"
)

func getStatuslessData(obj interface{}) ([]byte, error) {
	byteData, err := json.Marshal(obj)
	if err != nil {
		return nil, err
	}

	var mapData map[string]interface{}
	if err := json.Unmarshal(byteData, &mapData); err != nil {
		return nil, err
	}

	delete(mapData, "status")
	return json.Marshal(mapData)
}

func getSpec(obj interface{}) ([]byte, error) {
	byteData, err := json.Marshal(obj)
	if err != nil {
		return nil, err
	}

	var mapData map[string]interface{}
	if err := json.Unmarshal(byteData, &mapData); err != nil {
		return nil, err
	}

	return json.Marshal(mapData["spec"])
}

// ShouldSync will validate two objects and see if there are any updates worth
// syncing.
func ShouldSync(old, new interface{}) (bool, error) {
	oldData, err := getStatuslessData(old)
	if err != nil {
		return false, err
	}

	newData, err := getStatuslessData(new)
	if err != nil {
		return false, err
	}

	return string(oldData) != string(newData), nil
}

// SpecChanges will look at the `spec` in runtime objects and see if they
// differ.
func SpecChanges(old, new interface{}) (bool, error) {
	oldData, err := getSpec(old)
	if err != nil {
		return false, err
	}

	newData, err := getSpec(new)
	if err != nil {
		return false, err
	}

	return string(oldData) != string(newData), nil
}
