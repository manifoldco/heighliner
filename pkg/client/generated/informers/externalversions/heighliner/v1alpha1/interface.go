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

// Code generated by informer-gen. DO NOT EDIT.

// This file was automatically generated by informer-gen

package v1alpha1

import (
	internalinterfaces "github.com/manifoldco/heighliner/pkg/client/generated/informers/externalversions/internalinterfaces"
)

// Interface provides access to all the informers in this group version.
type Interface interface {
	// AvailabilityPolicies returns a AvailabilityPolicyInformer.
	AvailabilityPolicies() AvailabilityPolicyInformer
	// ConfigPolicies returns a ConfigPolicyInformer.
	ConfigPolicies() ConfigPolicyInformer
	// GitHubRepositories returns a GitHubRepositoryInformer.
	GitHubRepositories() GitHubRepositoryInformer
	// HealthPolicies returns a HealthPolicyInformer.
	HealthPolicies() HealthPolicyInformer
	// ImagePolicies returns a ImagePolicyInformer.
	ImagePolicies() ImagePolicyInformer
	// Microservices returns a MicroserviceInformer.
	Microservices() MicroserviceInformer
	// NetworkPolicies returns a NetworkPolicyInformer.
	NetworkPolicies() NetworkPolicyInformer
	// SecurityPolicies returns a SecurityPolicyInformer.
	SecurityPolicies() SecurityPolicyInformer
	// VersionedMicroservices returns a VersionedMicroserviceInformer.
	VersionedMicroservices() VersionedMicroserviceInformer
	// VersioningPolicies returns a VersioningPolicyInformer.
	VersioningPolicies() VersioningPolicyInformer
}

type version struct {
	factory          internalinterfaces.SharedInformerFactory
	namespace        string
	tweakListOptions internalinterfaces.TweakListOptionsFunc
}

// New returns a new Interface.
func New(f internalinterfaces.SharedInformerFactory, namespace string, tweakListOptions internalinterfaces.TweakListOptionsFunc) Interface {
	return &version{factory: f, namespace: namespace, tweakListOptions: tweakListOptions}
}

// AvailabilityPolicies returns a AvailabilityPolicyInformer.
func (v *version) AvailabilityPolicies() AvailabilityPolicyInformer {
	return &availabilityPolicyInformer{factory: v.factory, namespace: v.namespace, tweakListOptions: v.tweakListOptions}
}

// ConfigPolicies returns a ConfigPolicyInformer.
func (v *version) ConfigPolicies() ConfigPolicyInformer {
	return &configPolicyInformer{factory: v.factory, namespace: v.namespace, tweakListOptions: v.tweakListOptions}
}

// GitHubRepositories returns a GitHubRepositoryInformer.
func (v *version) GitHubRepositories() GitHubRepositoryInformer {
	return &gitHubRepositoryInformer{factory: v.factory, namespace: v.namespace, tweakListOptions: v.tweakListOptions}
}

// HealthPolicies returns a HealthPolicyInformer.
func (v *version) HealthPolicies() HealthPolicyInformer {
	return &healthPolicyInformer{factory: v.factory, namespace: v.namespace, tweakListOptions: v.tweakListOptions}
}

// ImagePolicies returns a ImagePolicyInformer.
func (v *version) ImagePolicies() ImagePolicyInformer {
	return &imagePolicyInformer{factory: v.factory, namespace: v.namespace, tweakListOptions: v.tweakListOptions}
}

// Microservices returns a MicroserviceInformer.
func (v *version) Microservices() MicroserviceInformer {
	return &microserviceInformer{factory: v.factory, namespace: v.namespace, tweakListOptions: v.tweakListOptions}
}

// NetworkPolicies returns a NetworkPolicyInformer.
func (v *version) NetworkPolicies() NetworkPolicyInformer {
	return &networkPolicyInformer{factory: v.factory, namespace: v.namespace, tweakListOptions: v.tweakListOptions}
}

// SecurityPolicies returns a SecurityPolicyInformer.
func (v *version) SecurityPolicies() SecurityPolicyInformer {
	return &securityPolicyInformer{factory: v.factory, namespace: v.namespace, tweakListOptions: v.tweakListOptions}
}

// VersionedMicroservices returns a VersionedMicroserviceInformer.
func (v *version) VersionedMicroservices() VersionedMicroserviceInformer {
	return &versionedMicroserviceInformer{factory: v.factory, namespace: v.namespace, tweakListOptions: v.tweakListOptions}
}

// VersioningPolicies returns a VersioningPolicyInformer.
func (v *version) VersioningPolicies() VersioningPolicyInformer {
	return &versioningPolicyInformer{factory: v.factory, namespace: v.namespace, tweakListOptions: v.tweakListOptions}
}
