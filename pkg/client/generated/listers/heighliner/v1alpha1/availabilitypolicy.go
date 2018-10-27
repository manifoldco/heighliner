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

// Code generated by lister-gen. DO NOT EDIT.

// This file was automatically generated by lister-gen

package v1alpha1

import (
	v1alpha1 "github.com/manifoldco/heighliner/apis/heighliner/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"
)

// AvailabilityPolicyLister helps list AvailabilityPolicies.
type AvailabilityPolicyLister interface {
	// List lists all AvailabilityPolicies in the indexer.
	List(selector labels.Selector) (ret []*v1alpha1.AvailabilityPolicy, err error)
	// AvailabilityPolicies returns an object that can list and get AvailabilityPolicies.
	AvailabilityPolicies(namespace string) AvailabilityPolicyNamespaceLister
	AvailabilityPolicyListerExpansion
}

// availabilityPolicyLister implements the AvailabilityPolicyLister interface.
type availabilityPolicyLister struct {
	indexer cache.Indexer
}

// NewAvailabilityPolicyLister returns a new AvailabilityPolicyLister.
func NewAvailabilityPolicyLister(indexer cache.Indexer) AvailabilityPolicyLister {
	return &availabilityPolicyLister{indexer: indexer}
}

// List lists all AvailabilityPolicies in the indexer.
func (s *availabilityPolicyLister) List(selector labels.Selector) (ret []*v1alpha1.AvailabilityPolicy, err error) {
	err = cache.ListAll(s.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*v1alpha1.AvailabilityPolicy))
	})
	return ret, err
}

// AvailabilityPolicies returns an object that can list and get AvailabilityPolicies.
func (s *availabilityPolicyLister) AvailabilityPolicies(namespace string) AvailabilityPolicyNamespaceLister {
	return availabilityPolicyNamespaceLister{indexer: s.indexer, namespace: namespace}
}

// AvailabilityPolicyNamespaceLister helps list and get AvailabilityPolicies.
type AvailabilityPolicyNamespaceLister interface {
	// List lists all AvailabilityPolicies in the indexer for a given namespace.
	List(selector labels.Selector) (ret []*v1alpha1.AvailabilityPolicy, err error)
	// Get retrieves the AvailabilityPolicy from the indexer for a given namespace and name.
	Get(name string) (*v1alpha1.AvailabilityPolicy, error)
	AvailabilityPolicyNamespaceListerExpansion
}

// availabilityPolicyNamespaceLister implements the AvailabilityPolicyNamespaceLister
// interface.
type availabilityPolicyNamespaceLister struct {
	indexer   cache.Indexer
	namespace string
}

// List lists all AvailabilityPolicies in the indexer for a given namespace.
func (s availabilityPolicyNamespaceLister) List(selector labels.Selector) (ret []*v1alpha1.AvailabilityPolicy, err error) {
	err = cache.ListAllByNamespace(s.indexer, s.namespace, selector, func(m interface{}) {
		ret = append(ret, m.(*v1alpha1.AvailabilityPolicy))
	})
	return ret, err
}

// Get retrieves the AvailabilityPolicy from the indexer for a given namespace and name.
func (s availabilityPolicyNamespaceLister) Get(name string) (*v1alpha1.AvailabilityPolicy, error) {
	obj, exists, err := s.indexer.GetByKey(s.namespace + "/" + name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(v1alpha1.Resource("availabilitypolicy"), name)
	}
	return obj.(*v1alpha1.AvailabilityPolicy), nil
}
