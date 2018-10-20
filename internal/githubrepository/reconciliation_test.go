package githubrepository

import (
	"context"
	"errors"
	"net/http"
	"reflect"
	"testing"
	"time"

	"github.com/google/go-github/github"
	"github.com/manifoldco/heighliner/apis/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestReconciliateRepository(t *testing.T) {
	now := time.Now()

	testCases := []struct {
		desc     string
		ghp      *v1alpha1.GitHubRepository
		period   time.Duration
		client   reconcilationClient
		releases []v1alpha1.GitHubRelease
		err      error
	}{
		{
			desc: "when reconciliation has not last updated yet",
			ghp: &v1alpha1.GitHubRepository{
				Status: v1alpha1.GitHubRepositoryStatus{
					Reconciliation: v1alpha1.GitHubReconciliation{},
				},
			},
			period: 10 * time.Minute,
			err:    nil,
		},
		{
			desc: "when update isn't due yet",
			ghp: &v1alpha1.GitHubRepository{
				Status: v1alpha1.GitHubRepositoryStatus{
					Reconciliation: v1alpha1.GitHubReconciliation{
						LastUpdate: metaTime(now.Add(-5 * time.Minute)),
					},
				},
			},
			period: 10 * time.Minute,
			err:    nil,
		},
		{
			desc: "when the latest release is the same",
			ghp: &v1alpha1.GitHubRepository{
				Spec: v1alpha1.GitHubRepositorySpec{
					Owner: "manifoldco",
					Repo:  "heighliner",
				},
				Status: v1alpha1.GitHubRepositoryStatus{
					Reconciliation: v1alpha1.GitHubReconciliation{
						LastUpdate: metaTime(now.Add(-15 * time.Minute)),
					},
					Releases: []v1alpha1.GitHubRelease{
						{Tag: "v1.0.0"},
					},
				},
			},
			client: &mockReconciliationClient{
				GetLatestReleaseFn: func(ctx context.Context, owner, repo string) (
					*github.RepositoryRelease, *github.Response, error) {
					tag := "v1.0.0"
					return &github.RepositoryRelease{TagName: &tag}, nil, nil
				},
				ListPullRequestsFn: func(ctx context.Context, owner, repo string,
					opt *github.PullRequestListOptions) ([]*github.PullRequest, *github.Response, error) {
					return nil, nil, nil
				},
			},
			period: 10 * time.Minute,
			releases: []v1alpha1.GitHubRelease{
				{Tag: "v1.0.0"},
			},
		},
		{
			desc: "when the latest release is not the same",
			ghp: &v1alpha1.GitHubRepository{
				Spec: v1alpha1.GitHubRepositorySpec{
					Owner: "manifoldco",
					Repo:  "heighliner",
				},
				Status: v1alpha1.GitHubRepositoryStatus{
					Reconciliation: v1alpha1.GitHubReconciliation{
						LastUpdate: metaTime(now.Add(-15 * time.Minute)),
					},
					Releases: []v1alpha1.GitHubRelease{
						{Tag: "v1.0.0"},
						{Tag: "pr-sha1"},
					},
				},
			},
			client: &mockReconciliationClient{
				GetLatestReleaseFn: func(ctx context.Context, owner, repo string) (
					*github.RepositoryRelease, *github.Response, error) {
					tag := "v2.0.0"
					return &github.RepositoryRelease{TagName: &tag}, nil, nil
				},
				ListReleasesFn: func(ctx context.Context, owner, repo string,
					opt *github.ListOptions) ([]*github.RepositoryRelease, *github.Response, error) {
					first := "v1.0.0"
					second := "v2.0.0"
					draft := false
					ts := github.Timestamp{Time: now}

					releases := []*github.RepositoryRelease{
						{
							TagName:     &first,
							Draft:       &draft,
							PublishedAt: &ts,
						},
						{
							TagName:     &second,
							Draft:       &draft,
							PublishedAt: &ts,
						},
					}
					resp := &github.Response{
						NextPage: 0,
					}
					return releases, resp, nil
				},
				ListPullRequestsFn: func(ctx context.Context, owner, repo string,
					opt *github.PullRequestListOptions) ([]*github.PullRequest, *github.Response, error) {

					state := "open"
					ref := "123"
					sha := "456"
					ts := github.Timestamp{Time: now}

					prs := []*github.PullRequest{
						{
							State: &state,
							Head: &github.PullRequestBranch{
								Ref: &ref,
								SHA: &sha,
								Repo: &github.Repository{
									UpdatedAt: &ts,
								},
							},
						},
					}

					return prs, nil, nil
				},
			},
			period: 10 * time.Minute,
			releases: []v1alpha1.GitHubRelease{
				{
					Name:        "v1.0.0",
					Tag:         "v1.0.0",
					ReleaseTime: metav1.NewTime(now),
					Level:       v1alpha1.SemVerLevelRelease,
				},
				{
					Name:        "v2.0.0",
					Tag:         "v2.0.0",
					ReleaseTime: metav1.NewTime(now),
					Level:       v1alpha1.SemVerLevelRelease,
				},
				{
					Name:        "123",
					Tag:         "456",
					ReleaseTime: metav1.NewTime(now),
					Level:       v1alpha1.SemVerLevelPreview,
				},
			},
		},
		{
			desc: "when latest release fails",
			ghp: &v1alpha1.GitHubRepository{
				Spec: v1alpha1.GitHubRepositorySpec{
					Owner: "manifoldco",
					Repo:  "heighliner",
				},
				Status: v1alpha1.GitHubRepositoryStatus{
					Reconciliation: v1alpha1.GitHubReconciliation{
						LastUpdate: metaTime(now.Add(-15 * time.Minute)),
					},
					Releases: []v1alpha1.GitHubRelease{
						{Tag: "v1.0.0"},
					},
				},
			},
			client: &mockReconciliationClient{
				GetLatestReleaseFn: func(ctx context.Context, owner, repo string) (
					*github.RepositoryRelease, *github.Response, error) {
					resp := &github.Response{
						Response: &http.Response{
							StatusCode: http.StatusInternalServerError,
						},
					}
					return nil, resp, errors.New("failed to get releases")
				},
			},
			period: 10 * time.Minute,
			releases: []v1alpha1.GitHubRelease{
				{Tag: "v1.0.0"},
			},
			err: errors.New("failed to get releases"),
		},
		{
			desc: "when list release fails",
			ghp: &v1alpha1.GitHubRepository{
				Spec: v1alpha1.GitHubRepositorySpec{
					Owner: "manifoldco",
					Repo:  "heighliner",
				},
				Status: v1alpha1.GitHubRepositoryStatus{
					Reconciliation: v1alpha1.GitHubReconciliation{
						LastUpdate: metaTime(now.Add(-15 * time.Minute)),
					},
					Releases: []v1alpha1.GitHubRelease{
						{Tag: "v1.0.0"},
					},
				},
			},
			client: &mockReconciliationClient{
				GetLatestReleaseFn: func(ctx context.Context, owner, repo string) (
					*github.RepositoryRelease, *github.Response, error) {
					resp := &github.Response{
						Response: &http.Response{
							StatusCode: http.StatusNotFound,
						},
					}
					return nil, resp, errors.New("not found")
				},
				ListReleasesFn: func(ctx context.Context, owner, repo string,
					opt *github.ListOptions) ([]*github.RepositoryRelease, *github.Response, error) {
					return nil, nil, errors.New("failed to get releases")
				},
			},
			period: 10 * time.Minute,
			releases: []v1alpha1.GitHubRelease{
				{Tag: "v1.0.0"},
			},
			err: errors.New("failed to get releases"),
		},
		{
			desc: "when list prs fails",
			ghp: &v1alpha1.GitHubRepository{
				Spec: v1alpha1.GitHubRepositorySpec{
					Owner: "manifoldco",
					Repo:  "heighliner",
				},
				Status: v1alpha1.GitHubRepositoryStatus{
					Reconciliation: v1alpha1.GitHubReconciliation{
						LastUpdate: metaTime(now.Add(-15 * time.Minute)),
					},
					Releases: []v1alpha1.GitHubRelease{
						{Tag: "v1.0.0"},
					},
				},
			},
			client: &mockReconciliationClient{
				GetLatestReleaseFn: func(ctx context.Context, owner, repo string) (
					*github.RepositoryRelease, *github.Response, error) {
					resp := &github.Response{
						Response: &http.Response{
							StatusCode: http.StatusNotFound,
						},
					}
					return nil, resp, errors.New("not found")
				},
				ListReleasesFn: func(ctx context.Context, owner, repo string,
					opt *github.ListOptions) ([]*github.RepositoryRelease, *github.Response, error) {
					resp := &github.Response{
						NextPage: 0,
					}
					return nil, resp, nil
				},
				ListPullRequestsFn: func(ctx context.Context, owner, repo string,
					opt *github.PullRequestListOptions) ([]*github.PullRequest, *github.Response, error) {
					return nil, nil, errors.New("failed to get prs")
				},
			},
			period: 10 * time.Minute,
			releases: []v1alpha1.GitHubRelease{
				{Tag: "v1.0.0"},
			},
			err: errors.New("failed to get prs"),
		},
	}
	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {

			ctx := context.Background()

			err := reconciliateRepository(ctx, tC.client, tC.ghp, tC.period)
			switch {
			case tC.err != nil && err != nil && tC.err.Error() == err.Error(): //ok
			case tC.err != nil && err != nil && tC.err.Error() != err.Error():
				t.Fatalf("Expected error to eq %v got %v", tC.err, err)
			case tC.err != nil:
				t.Fatalf("Expected error %v, got none", tC.err)
			case err != nil:
				t.Fatalf("Expected no errors, got %v", err)
			}

			releases := tC.ghp.Status.Releases

			if !reflect.DeepEqual(releases, tC.releases) {
				t.Fatalf("Expected releases to eq %v, got %v", tC.releases, releases)
			}
		})
	}
}

type mockReconciliationClient struct {
	GetLatestReleaseFn func(ctx context.Context, owner, repo string) (*github.RepositoryRelease,
		*github.Response, error)
	ListReleasesFn func(ctx context.Context, owner, repo string, opt *github.ListOptions) (
		[]*github.RepositoryRelease, *github.Response, error)
	ListPullRequestsFn func(ctx context.Context, owner string, repo string,
		opt *github.PullRequestListOptions) ([]*github.PullRequest, *github.Response, error)
}

func (m *mockReconciliationClient) GetLatestRelease(ctx context.Context, owner, repo string) (
	*github.RepositoryRelease, *github.Response, error) {
	return m.GetLatestReleaseFn(ctx, owner, repo)
}

func (m *mockReconciliationClient) ListReleases(ctx context.Context, owner, repo string,
	opt *github.ListOptions) ([]*github.RepositoryRelease, *github.Response, error) {
	return m.ListReleasesFn(ctx, owner, repo, opt)
}

func (m *mockReconciliationClient) ListPullRequests(ctx context.Context, owner, repo string,
	opt *github.PullRequestListOptions) ([]*github.PullRequest, *github.Response, error) {
	return m.ListPullRequestsFn(ctx, owner, repo, opt)
}
