package tester

import (
	"github.com/jelmersnoeck/kubekit/patcher"
	"k8s.io/apimachinery/pkg/runtime"
)

// PatchClient is a dummy kubekit client which is used for testing purposes.
type PatchClient struct {
	ApplyFunc  func(obj runtime.Object, opts ...patcher.OptionFunc) ([]byte, error)
	GetFunc    func(obj interface{}, namespace, name string) error
	DeleteFunc func(runtime.Object, ...patcher.OptionFunc) error
}

// Flush resets all the patcher functions so we can ensure it's not being used
// improperly from other test cases.
// One should use `defer PatchClient.Flush()` in every test case.
func (c *PatchClient) Flush() {
	c.ApplyFunc = nil
	c.GetFunc = nil
	c.DeleteFunc = nil
}

// Apply mimics the Apply behaviour of the patch client by calling the
// ApplyFunc.
func (c *PatchClient) Apply(obj runtime.Object, opts ...patcher.OptionFunc) ([]byte, error) {
	return c.ApplyFunc(obj, opts...)
}

// Get mimics the Get behaviour of the get client by calling the GetFunc.
func (c *PatchClient) Get(obj interface{}, namespace, name string) error {
	return c.GetFunc(obj, namespace, name)
}

// Delete mimics the Get behaviour of the clinet by calling DeleteFunc.
func (c *PatchClient) Delete(obj runtime.Object, ops ...patcher.OptionFunc) error {
	return c.DeleteFunc(obj, ops...)
}
