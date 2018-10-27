package githubrepository

import (
	"context"
	"reflect"
	"testing"

	"k8s.io/api/core/v1"

	"github.com/google/go-github/github"
	"github.com/manifoldco/heighliner/apis/heighliner/v1alpha1"
	"github.com/manifoldco/heighliner/internal/k8sutils"
)

func TestGetSecretAuthToken(t *testing.T) {
	cl := &dummyClient{}

	t.Run("without valid key", func(t *testing.T) {
		cl.getFunc = func(obj interface{}, ns, name string) error {
			secret := obj.(*v1.Secret)
			secret.StringData = map[string]string{
				"WRONG_KEY": "",
			}
			return nil
		}

		_, err := getSecretAuthToken(cl, "test", "test-secret")
		if err == nil {
			t.Errorf("Expected an error, got none")
		}
	})

	t.Run("with valid key", func(t *testing.T) {
		expected := "uptownfunc"
		cl.getFunc = func(obj interface{}, ns, name string) error {
			secret := obj.(*v1.Secret)
			secret.Data = map[string][]byte{
				"GITHUB_AUTH_TOKEN": []byte(expected),
			}
			return nil
		}

		token, err := getSecretAuthToken(cl, "test", "test-secret")
		if err != nil {
			t.Errorf("Expected no error, got '%s'", err)
		}

		if token != expected {
			t.Errorf("Expected token to equal '%s', got '%s'", expected, token)
		}
	})
}

func TestCreateDeployment(t *testing.T) {
	repo := &v1alpha1.GitHubRepository{
		Spec: v1alpha1.GitHubRepositorySpec{
			Owner: "manifoldco",
			Repo:  "heighliner",
		},
	}

	release := v1alpha1.GitHubRelease{
		Deployment: &v1alpha1.Deployment{
			URL:   k8sutils.PtrString("my-url"),
			State: "success",
		},
	}

	t.Run("with a successful request", func(t *testing.T) {
		cl := defaultDummyDeploymentClient(t)

		id, err := createGitHubDeployment(context.Background(), cl, repo, release)
		if err != nil {
			t.Errorf("Expected no error, got '%s'", err)
		}

		if *id != int64(1234) {
			t.Errorf("Expected id to equal '1234', got '%d'", id)
		}
	})

	t.Run("With existing matching status", func(t *testing.T) {
		cl := defaultDummyDeploymentClient(t)

		cl.sf = func(context.Context, string, string, int64, *github.DeploymentStatusRequest) (*github.DeploymentStatus, *github.Response, error) {
			t.Errorf("Status creation should not have been called.")
			return nil, nil, nil
		}

		cl.lf = func(context.Context, string, string, int64, *github.ListOptions) ([]*github.DeploymentStatus, *github.Response, error) {
			return []*github.DeploymentStatus{{State: k8sutils.PtrString("success")}}, &github.Response{}, nil
		}

		id, err := createGitHubDeployment(context.Background(), cl, repo, release)
		if err != nil {
			t.Errorf("Expected no error, got '%s'", err)
		}

		if *id != int64(1234) {
			t.Errorf("Expected id to equal '1234', got '%d'", id)
		}

	})
}

type dummyDeploymentClient struct {
	f  func(context.Context, string, string, *github.DeploymentRequest) (*github.Deployment, *github.Response, error)
	sf func(context.Context, string, string, int64, *github.DeploymentStatusRequest) (*github.DeploymentStatus, *github.Response, error)
	lf func(context.Context, string, string, int64, *github.ListOptions) ([]*github.DeploymentStatus, *github.Response, error)
}

func (c *dummyDeploymentClient) CreateDeployment(ctx context.Context, owner, repo string, request *github.DeploymentRequest) (*github.Deployment, *github.Response, error) {
	return c.f(ctx, owner, repo, request)
}

func (c *dummyDeploymentClient) CreateDeploymentStatus(ctx context.Context, owner, repo string, id int64, request *github.DeploymentStatusRequest) (*github.DeploymentStatus, *github.Response, error) {
	return c.sf(ctx, owner, repo, id, request)
}

func (c *dummyDeploymentClient) ListDeploymentStatuses(ctx context.Context, owner, repo string, id int64, opt *github.ListOptions) ([]*github.DeploymentStatus, *github.Response, error) {
	return c.lf(ctx, owner, repo, id, opt)
}

func defaultDummyDeploymentClient(t *testing.T) *dummyDeploymentClient {
	dplID := int64(1234)

	return &dummyDeploymentClient{
		f: func(ctx context.Context, owner, repo string, request *github.DeploymentRequest) (*github.Deployment, *github.Response, error) {
			dpl := &github.Deployment{
				ID: &dplID,
			}
			return dpl, nil, nil
		},

		sf: func(ctx context.Context, owner, repo string, id int64, request *github.DeploymentStatusRequest) (*github.DeploymentStatus, *github.Response, error) {
			if id != dplID {
				t.Errorf("Wrong ID supplied to deployment status")
			}

			status := &github.DeploymentStatus{}
			return status, nil, nil
		},

		lf: func(ctx context.Context, owner, repo string, id int64, opt *github.ListOptions) ([]*github.DeploymentStatus, *github.Response, error) {
			return nil, &github.Response{}, nil
		},
	}

}

type dummyClient struct {
	getFunc func(interface{}, string, string) error
}

func (c *dummyClient) Get(obj interface{}, ns string, name string) error {
	return c.getFunc(obj, ns, name)
}

func TestReconcileDeployments(t *testing.T) {
	fakeURL := "https://www.fake.com"

	tcs := []struct {
		name     string
		domains  []v1alpha1.Domain
		deleted  bool
		releases []v1alpha1.GitHubRelease
		out      []v1alpha1.GitHubRelease
		changed  []int
	}{
		{"No releases", []v1alpha1.Domain{{}}, false, nil, []v1alpha1.GitHubRelease{}, []int{}},
		{"No domains", nil, false, []v1alpha1.GitHubRelease{{}}, []v1alpha1.GitHubRelease{{}}, []int{}},

		{
			"New domain",
			[]v1alpha1.Domain{{URL: fakeURL, SemVer: &v1alpha1.SemVerRelease{Name: "foo", Version: "1"}}},
			false,
			[]v1alpha1.GitHubRelease{{Name: "foo", Tag: "1"}},
			[]v1alpha1.GitHubRelease{{Name: "foo", Tag: "1", Deployment: &v1alpha1.Deployment{State: "success", URL: &fakeURL}}},
			[]int{0},
		},

		{
			"Existing deploy",
			[]v1alpha1.Domain{{URL: fakeURL, SemVer: &v1alpha1.SemVerRelease{Name: "foo", Version: "1"}}},
			false,
			[]v1alpha1.GitHubRelease{{Name: "foo", Tag: "1", Deployment: &v1alpha1.Deployment{State: "success", URL: &fakeURL}}},
			[]v1alpha1.GitHubRelease{{Name: "foo", Tag: "1", Deployment: &v1alpha1.Deployment{State: "success", URL: &fakeURL}}},
			[]int{},
		},

		{
			"Removed domain",
			[]v1alpha1.Domain{{URL: fakeURL, SemVer: &v1alpha1.SemVerRelease{Name: "foo", Version: "1"}}},
			true,
			[]v1alpha1.GitHubRelease{{Name: "foo", Tag: "1", Deployment: &v1alpha1.Deployment{State: "success", URL: &fakeURL}}},
			[]v1alpha1.GitHubRelease{{Name: "foo", Tag: "1", Deployment: &v1alpha1.Deployment{State: "inactive"}}},
			[]int{0},
		},

		{
			"Unknown releases are kept the same",
			[]v1alpha1.Domain{{URL: fakeURL, SemVer: &v1alpha1.SemVerRelease{Name: "bar", Version: "1"}}},
			true,
			[]v1alpha1.GitHubRelease{{Name: "foo", Tag: "1", Deployment: &v1alpha1.Deployment{State: "success", URL: &fakeURL}}},
			[]v1alpha1.GitHubRelease{{Name: "foo", Tag: "1", Deployment: &v1alpha1.Deployment{State: "success", URL: &fakeURL}}},
			[]int{},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			changed, newReleases := reconcileDeployments(tc.domains, tc.deleted, tc.releases)

			if !reflect.DeepEqual(changed, tc.changed) {
				t.Error("bad result for changed. got:", changed, "wanted:", tc.changed)
			}

			if len(newReleases) != len(tc.out) {
				t.Error("wrong number of releases returned. got:", newReleases, "wanted:", tc.out)
			}

			if !reflect.DeepEqual(newReleases, tc.out) {
				t.Error("releases did not match! got:", newReleases, "expected:", tc.out)
			}
		})
	}
}
