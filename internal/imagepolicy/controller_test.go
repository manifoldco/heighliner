package imagepolicy

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"reflect"
	"testing"

	"github.com/jelmersnoeck/kubekit/patcher"
	"github.com/manifoldco/heighliner/apis/v1alpha1"
	"github.com/manifoldco/heighliner/internal/registry"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func TestController_SyncPolicy(t *testing.T) {
	AddRegistry("mock", func(*v1.Secret) (registry.Registry, error) {
		return &mockContainerRegistry{}, nil
	})

	tcs := []struct {
		scenario string
		policy   *v1alpha1.ImagePolicy
		patcher  patchClient
		log      string
		err      error
	}{
		{
			scenario: "ok, nothing to apply",
			policy: &v1alpha1.ImagePolicy{
				ObjectMeta: metav1.ObjectMeta{
					Name: "ip-test",
				},
				Spec: v1alpha1.ImagePolicySpec{
					Filter: v1alpha1.ImagePolicyFilter{
						GitHub: &v1.ObjectReference{
							Name:      "manifoldco",
							Namespace: "websites",
						},
					},
					ContainerRegistry: &v1alpha1.ContainerRegistry{
						Name: "mock",
						ImagePullSecrets: []v1.LocalObjectReference{
							{
								Name: "secret",
							},
						},
					},
				},
			},
			patcher: &mockPatchClient{
				GetFn: func(v interface{}, ns, name string) error {
					return nil
				},
				ApplyFn: func(runtime.Object, ...patcher.OptionFunc) ([]byte, error) {
					return nil, nil
				},
			},
		},
		{
			scenario: "when using a pinned version",
			policy: &v1alpha1.ImagePolicy{
				ObjectMeta: metav1.ObjectMeta{
					Name: "ip-test",
				},
				Spec: v1alpha1.ImagePolicySpec{
					Filter: v1alpha1.ImagePolicyFilter{
						Pinned: &v1alpha1.SemVerRelease{
							Version: "1.2.3",
						},
					},
					VersioningPolicy: v1.ObjectReference{
						Namespace: "vp",
					},
					ContainerRegistry: &v1alpha1.ContainerRegistry{
						Name: "mock",
						ImagePullSecrets: []v1.LocalObjectReference{
							{
								Name: "secret",
							},
						},
					},
				},
			},
			patcher: &mockPatchClient{
				GetFn: func(v interface{}, ns, name string) error {
					if ns == "vp" {
						vp := v.(*v1alpha1.VersioningPolicy)
						vp.Spec = v1alpha1.VersioningPolicySpec{
							SemVer: &v1alpha1.SemVerSource{
								Level: v1alpha1.SemVerLevelRelease,
							},
						}

						return nil
					}

					return nil
				},
				ApplyFn: func(runtime.Object, ...patcher.OptionFunc) ([]byte, error) {
					return nil, nil
				},
			},
		},
		{
			scenario: "when no filter is defined",
			policy: &v1alpha1.ImagePolicy{
				ObjectMeta: metav1.ObjectMeta{
					Name: "ip-test",
				},
				Spec: v1alpha1.ImagePolicySpec{
					Filter: v1alpha1.ImagePolicyFilter{},
					ContainerRegistry: &v1alpha1.ContainerRegistry{
						Name: "mock",
						ImagePullSecrets: []v1.LocalObjectReference{
							{
								Name: "secret",
							},
						},
					},
				},
			},
			patcher: &mockPatchClient{
				GetFn: func(v interface{}, ns, name string) error {
					return nil
				},
			},
			err: errors.New("image spec filter not defined"),
		},
		{
			scenario: "when github repo fails",
			policy: &v1alpha1.ImagePolicy{
				ObjectMeta: metav1.ObjectMeta{
					Name: "ip-test",
				},
				Spec: v1alpha1.ImagePolicySpec{
					Filter: v1alpha1.ImagePolicyFilter{
						GitHub: &v1.ObjectReference{
							Name:      "manifoldco",
							Namespace: "websites",
						},
					},
					ContainerRegistry: &v1alpha1.ContainerRegistry{
						Name: "mock",
						ImagePullSecrets: []v1.LocalObjectReference{
							{
								Name: "secret",
							},
						},
					},
				},
			},
			patcher: &mockPatchClient{
				GetFn: func(v interface{}, ns, name string) error {
					if ns == "websites" {
						return errors.New("github repo not found")
					}

					return nil
				},
			},
			log: "Could not retrieve GithubRepository for ip-test: github repo not found\n",
		},
		{
			scenario: "when container registry secrets are not present",
			policy: &v1alpha1.ImagePolicy{
				ObjectMeta: metav1.ObjectMeta{
					Name: "ip-test",
				},
				Spec: v1alpha1.ImagePolicySpec{
					Filter: v1alpha1.ImagePolicyFilter{
						GitHub: &v1.ObjectReference{
							Name:      "manifoldco",
							Namespace: "websites",
						},
					},
				},
			},
			patcher: &mockPatchClient{
				GetFn: func(v interface{}, ns, name string) error {
					return nil
				},
			},
			log: "Could not retrieve registry for ip-test: No ImagePullSecrets available\n",
		},
		{
			scenario: "when default container registry fails",
			policy: &v1alpha1.ImagePolicy{
				ObjectMeta: metav1.ObjectMeta{
					Name: "ip-test",
				},
				Spec: v1alpha1.ImagePolicySpec{
					Filter: v1alpha1.ImagePolicyFilter{
						GitHub: &v1.ObjectReference{
							Name:      "manifoldco",
							Namespace: "websites",
						},
					},
					ContainerRegistry: &v1alpha1.ContainerRegistry{
						ImagePullSecrets: []v1.LocalObjectReference{
							{
								Name: "secret",
							},
						},
					},
				},
			},
			patcher: &mockPatchClient{
				GetFn: func(v interface{}, ns, name string) error {
					return nil
				},
			},
			log: "Could not retrieve registry for ip-test: unknown docker registry\n",
		},
		{
			scenario: "when custom container registry fails",
			policy: &v1alpha1.ImagePolicy{
				ObjectMeta: metav1.ObjectMeta{
					Name: "ip-test",
				},
				Spec: v1alpha1.ImagePolicySpec{
					Filter: v1alpha1.ImagePolicyFilter{
						GitHub: &v1.ObjectReference{
							Name:      "manifoldco",
							Namespace: "websites",
						},
					},
					ContainerRegistry: &v1alpha1.ContainerRegistry{
						Name: "azure",
						ImagePullSecrets: []v1.LocalObjectReference{
							{
								Name: "secret",
							},
						},
					},
				},
			},
			patcher: &mockPatchClient{
				GetFn: func(v interface{}, ns, name string) error {
					return nil
				},
			},
			log: "Could not retrieve registry for ip-test: unknown azure registry\n",
		},
		{
			scenario: "when applying new policy fails",
			policy: &v1alpha1.ImagePolicy{
				ObjectMeta: metav1.ObjectMeta{
					Name: "ip-test",
				},
				Spec: v1alpha1.ImagePolicySpec{
					Filter: v1alpha1.ImagePolicyFilter{
						GitHub: &v1.ObjectReference{
							Name:      "manifoldco",
							Namespace: "websites",
						},
					},
					ContainerRegistry: &v1alpha1.ContainerRegistry{
						Name: "mock",
						ImagePullSecrets: []v1.LocalObjectReference{
							{
								Name: "secret",
							},
						},
					},
				},
			},
			patcher: &mockPatchClient{
				GetFn: func(v interface{}, ns, name string) error {
					return nil
				},
				ApplyFn: func(runtime.Object, ...patcher.OptionFunc) ([]byte, error) {
					return nil, errors.New("failed to patch")
				},
			},
			err: errors.New("failed to patch"),
			log: "Error syncing ImagePolicy ip-test (): failed to patch\n",
		},
	}

	for _, tc := range tcs {
		t.Run(tc.scenario, func(t *testing.T) {
			var buf bytes.Buffer

			c := &Controller{
				patcher: tc.patcher,
				logger:  log.New(&buf, "", 0),
			}

			err := c.syncPolicy(tc.policy)

			if !reflect.DeepEqual(tc.err, err) {
				t.Fatalf("expected error to eq %v got %v", tc.err, err)
			}

			log := buf.String()

			if tc.log != log {
				t.Fatalf("expected log message to eq %q got %q", tc.log, log)
			}

		})
	}
}

func TestFilterImages(t *testing.T) {

	repo := &v1alpha1.GitHubRepository{
		Spec: v1alpha1.GitHubRepositorySpec{
			Repo: "github.com//manifoldco/heighliner",
		},
		Status: v1alpha1.GitHubRepositoryStatus{
			Releases: []v1alpha1.GitHubRelease{
				{
					Name:  "heighliner",
					Tag:   "latest",
					Level: v1alpha1.SemVerLevelRelease,
				},
				{
					Name:  "heighliner",
					Tag:   "rc",
					Level: v1alpha1.SemVerLevelReleaseCandidate,
				},
				{
					Name:  "heighliner",
					Tag:   "pr",
					Level: v1alpha1.SemVerLevelPreview,
				},
			},
		},
	}

	registry := &mockRegistryClient{}

	tcs := []struct {
		level v1alpha1.SemVerLevel
		tag   string
	}{
		{v1alpha1.SemVerLevelRelease, "latest"},
		{v1alpha1.SemVerLevelReleaseCandidate, "rc"},
		{v1alpha1.SemVerLevelPreview, "pr"},
	}

	for _, tc := range tcs {
		t.Run(fmt.Sprintf("with %s version policy for %s", tc.level, tc.tag), func(t *testing.T) {
			ip := &v1alpha1.ImagePolicy{
				Spec: v1alpha1.ImagePolicySpec{
					Image: "manifoldco/heighliner",
					Filter: v1alpha1.ImagePolicyFilter{
						GitHub: &v1.ObjectReference{
							Name: "github.com/manifoldco/heighliner",
						},
					},
				},
			}
			vp := &v1alpha1.VersioningPolicy{
				Spec: v1alpha1.VersioningPolicySpec{
					SemVer: &v1alpha1.SemVerSource{
						Level: tc.level,
					},
				},
			}

			expectedReleases := []v1alpha1.Release{
				{
					SemVer: &v1alpha1.SemVerRelease{
						Name:    "heighliner",
						Version: tc.tag,
					},

					Image: "manifoldco/heighliner:" + tc.tag,
				},
			}

			actualReleases, err := filterImages(ip.Spec.Image, ip.Spec.Match, repo, registry, vp)
			if err != nil {
				t.Errorf("Error filtering images for %s", ip.Name)
			}

			if expectedReleases[0].Image != actualReleases[0].Image {
				t.Errorf("Expected release image for release to be %s instead got %s", expectedReleases[0].Image, actualReleases[0].Image)
			}
		})
	}
}

type mockRegistryClient struct{}

func (c *mockRegistryClient) TagFor(image string, tag string, matcher *v1alpha1.ImagePolicyMatch) (string, error) {
	return tag, nil
}

type mockPatchClient struct {
	GetFn   func(interface{}, string, string) error
	ApplyFn func(runtime.Object, ...patcher.OptionFunc) ([]byte, error)
}

func (p *mockPatchClient) Get(v interface{}, ns string, name string) error {
	return p.GetFn(v, ns, name)
}

func (p *mockPatchClient) Apply(obj runtime.Object, opts ...patcher.OptionFunc) ([]byte, error) {
	return p.ApplyFn(obj, opts...)
}

type mockContainerRegistry struct{}

func (r *mockContainerRegistry) TagFor(string, string, *v1alpha1.ImagePolicyMatch) (string, error) {
	return "v0.0.1", nil
}
