/*
BSD 3-Clause License

Copyright (c) 2018, Arigato Machine Inc.
All rights reserved.

Redistribution and use in source and binary forms, with or without
modification, are permitted provided that the following conditions are met:

* Redistributions of source code must retain the above copyright notice, this
  list of conditions and the following disclaimer.

* Redistributions in binary form must reproduce the above copyright notice,
  this list of conditions and the following disclaimer in the documentation
  and/or other materials provided with the distribution.

* Neither the name of the copyright holder nor the names of its
  contributors may be used to endorse or promote products derived from
  this software without specific prior written permission.

THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS"
AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE
IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE
DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT HOLDER OR CONTRIBUTORS BE LIABLE
FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL
DAMAGES (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR
SERVICES; LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER
CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY,
OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
*/

// Code generated by client-gen. DO NOT EDIT.

package fake

import (
	v1alpha1 "github.com/manifoldco/heighliner/apis/heighliner/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakeVersioningPolicies implements VersioningPolicyInterface
type FakeVersioningPolicies struct {
	Fake *FakeHeighlinerV1alpha1
	ns   string
}

var versioningpoliciesResource = schema.GroupVersionResource{Group: "heighliner", Version: "v1alpha1", Resource: "versioningpolicies"}

var versioningpoliciesKind = schema.GroupVersionKind{Group: "heighliner", Version: "v1alpha1", Kind: "VersioningPolicy"}

// Get takes name of the versioningPolicy, and returns the corresponding versioningPolicy object, and an error if there is any.
func (c *FakeVersioningPolicies) Get(name string, options v1.GetOptions) (result *v1alpha1.VersioningPolicy, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewGetAction(versioningpoliciesResource, c.ns, name), &v1alpha1.VersioningPolicy{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.VersioningPolicy), err
}

// List takes label and field selectors, and returns the list of VersioningPolicies that match those selectors.
func (c *FakeVersioningPolicies) List(opts v1.ListOptions) (result *v1alpha1.VersioningPolicyList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewListAction(versioningpoliciesResource, versioningpoliciesKind, c.ns, opts), &v1alpha1.VersioningPolicyList{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &v1alpha1.VersioningPolicyList{}
	for _, item := range obj.(*v1alpha1.VersioningPolicyList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested versioningPolicies.
func (c *FakeVersioningPolicies) Watch(opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchAction(versioningpoliciesResource, c.ns, opts))

}

// Create takes the representation of a versioningPolicy and creates it.  Returns the server's representation of the versioningPolicy, and an error, if there is any.
func (c *FakeVersioningPolicies) Create(versioningPolicy *v1alpha1.VersioningPolicy) (result *v1alpha1.VersioningPolicy, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewCreateAction(versioningpoliciesResource, c.ns, versioningPolicy), &v1alpha1.VersioningPolicy{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.VersioningPolicy), err
}

// Update takes the representation of a versioningPolicy and updates it. Returns the server's representation of the versioningPolicy, and an error, if there is any.
func (c *FakeVersioningPolicies) Update(versioningPolicy *v1alpha1.VersioningPolicy) (result *v1alpha1.VersioningPolicy, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateAction(versioningpoliciesResource, c.ns, versioningPolicy), &v1alpha1.VersioningPolicy{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.VersioningPolicy), err
}

// Delete takes name of the versioningPolicy and deletes it. Returns an error if one occurs.
func (c *FakeVersioningPolicies) Delete(name string, options *v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteAction(versioningpoliciesResource, c.ns, name), &v1alpha1.VersioningPolicy{})

	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeVersioningPolicies) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	action := testing.NewDeleteCollectionAction(versioningpoliciesResource, c.ns, listOptions)

	_, err := c.Fake.Invokes(action, &v1alpha1.VersioningPolicyList{})
	return err
}

// Patch applies the patch and returns the patched versioningPolicy.
func (c *FakeVersioningPolicies) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1alpha1.VersioningPolicy, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(versioningpoliciesResource, c.ns, name, data, subresources...), &v1alpha1.VersioningPolicy{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.VersioningPolicy), err
}
