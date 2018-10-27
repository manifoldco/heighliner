package githubrepository

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/google/go-github/github"
	"github.com/manifoldco/heighliner/apis/heighliner/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// reconciliateRepository checks whether the reconciliation period has changed and if a new sync is
// required. If so, it gets the repository releases and opened pull-requests and update
// .Status.Releases.
func reconciliateRepository(ctx context.Context, ghClient reconcilationClient,
	ghp *v1alpha1.GitHubRepository, period time.Duration) error {

	if ghp.Status.Reconciliation.LastUpdate == nil {
		ghp.Status.Reconciliation.LastUpdate = metaTime(time.Now())
	}

	last := ghp.Status.Reconciliation.LastUpdate.Time

	next := last.Add(period)
	now := time.Now()

	if now.Before(next) {
		return nil
	}

	// GitHub doesn't have a way to sort or filter releases. Instead of getting all releases
	// all the time, we check first if we already have the latest release. If so, there is no
	// need to get all releases.
	lastestRelease, resp, err := ghClient.GetLatestRelease(ctx, ghp.Spec.Owner, ghp.Spec.Repo)
	if err != nil && resp.StatusCode != http.StatusNotFound {
		return err
	}

	fetchAllReleases := true
	if lastestRelease != nil {
		for _, r := range ghp.Status.Releases {
			if lastestRelease.TagName != nil && *lastestRelease.TagName == r.Tag {
				fetchAllReleases = false
				break
			}
		}
	}

	var releases []v1alpha1.GitHubRelease

	// If we need to fetch all releases, we loop over all release pages and collect all
	// releases. We then override the current list of .Status.Releases with this new one.
	if fetchAllReleases {
		var allReleases []*github.RepositoryRelease

		opt := &github.ListOptions{}

		for {
			releases, resp, err := ghClient.ListReleases(ctx, ghp.Spec.Owner, ghp.Spec.Repo, opt)
			if err != nil {
				return err
			}

			allReleases = append(allReleases, releases...)
			if resp.NextPage == 0 {
				break
			}

			opt.Page = resp.NextPage
		}

		for _, release := range allReleases {
			r, active := convertRelease(release)
			if active {
				releases = append(releases, *r)
			}
		}
	} else {
		currentReleases := ghp.Status.Releases

		// Remove previews from the current list because we are fetching all PRs below.
		for _, r := range currentReleases {
			if r.Level != v1alpha1.SemVerLevelPreview {
				releases = append(releases, r)
			}
		}
	}

	opt := &github.PullRequestListOptions{
		State:     "open",
		Sort:      "updated",
		Direction: "desc",
	}

	// Get the updated PRs. This will only get the latest 30. It should be enough for
	// most use-cases.
	prs, _, err := ghClient.ListPullRequests(ctx, ghp.Spec.Owner, ghp.Spec.Repo, opt)
	if err != nil {
		return err
	}

	for _, p := range prs {
		pr, _ := convertPullRequest(p)
		releases = append(releases, *pr)
	}

	diffReleases(ghp.Status.Releases, releases)

	ghp.Status.Releases = releases

	ghp.Status.Reconciliation.LastUpdate = metaTime(time.Now())

	return nil
}

// diffReleases logs the number of releases added or removed.
func diffReleases(old, new []v1alpha1.GitHubRelease) {
	diff := make(map[string]bool)

	added := len(new)
	removed := 0

	for _, n := range new {
		diff[n.Tag] = true
	}

	for _, o := range old {
		_, ok := diff[o.Tag]
		if ok {
			added--
		} else {
			removed++
		}
	}

	if removed > 0 {
		log.Printf("Removed %d releases", removed)
	}

	if added > 0 {
		log.Printf("Added %d releases", added)
	}
}

// reconciliationClient is an inteface with a subset of functions the GitHub client must implement
// to allow reconciliation of releases and opened pull-requests.
type reconcilationClient interface {
	GetLatestRelease(ctx context.Context, owner, repo string) (*github.RepositoryRelease,
		*github.Response, error)
	ListReleases(ctx context.Context, owner, repo string, opt *github.ListOptions) (
		[]*github.RepositoryRelease, *github.Response, error)
	ListPullRequests(ctx context.Context, owner string, repo string,
		opt *github.PullRequestListOptions) ([]*github.PullRequest, *github.Response, error)
}

type githubReconciliationClient struct {
	*github.Client
}

func (gh *githubReconciliationClient) GetLatestRelease(ctx context.Context, owner, repo string) (
	*github.RepositoryRelease, *github.Response, error) {
	return gh.Client.Repositories.GetLatestRelease(ctx, owner, repo)
}

func (gh *githubReconciliationClient) ListReleases(ctx context.Context, owner, repo string,
	opt *github.ListOptions) ([]*github.RepositoryRelease, *github.Response, error) {
	return gh.Client.Repositories.ListReleases(ctx, owner, repo, opt)
}
func (gh *githubReconciliationClient) ListPullRequests(ctx context.Context, owner string,
	repo string, opt *github.PullRequestListOptions) ([]*github.PullRequest, *github.Response,
	error) {
	return gh.Client.PullRequests.List(ctx, owner, repo, opt)
}

func metaTime(t time.Time) *metav1.Time {
	mt := metav1.NewTime(t)
	return &mt
}
