package githubrepository

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/google/go-github/github"
	"github.com/gorilla/mux"
	"github.com/jelmersnoeck/kubekit/patcher"
	"github.com/manifoldco/heighliner/pkg/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type (
	getClient interface {
		Get(interface{}, string, string) error
	}

	applyClient interface {
		Apply(runtime.Object, ...patcher.OptionFunc) ([]byte, error)
	}

	patchClient interface {
		getClient
		applyClient
	}
)

// callbackServer is the server that knows how to handle GitHub callbcaks and
// which will create status objects for new callbacks.
type callbackServer struct {
	sync.RWMutex

	// patcher is what we'll use to do interactions with the definitions linked
	// to the callback payloads.
	patcher patchClient

	// hooks is a list of hooks which know about the CRD id, repo and installed
	// hooks. This will be used to set up correct endpoints and validate
	// payloads as well as making sure we update the correct CRD.
	hooks []callbackHook

	// srv is the server we'll use to serve our contents with.
	srv *http.Server

	// hooksChan is where we'll receive new webhooks on which we should monitor
	// in our callback server.
	hooksChan chan callbackHook
}

type callbackHook struct {
	crdName      string
	crdNamespace string
	repo         string
	hook         *v1alpha1.GitHubHook

	// delete is used to remove the callbackHook from the callbackServer
	delete bool
}

func (s *callbackServer) start(address string) {
	hdlr := mux.NewRouter()
	hdlr.HandleFunc("/payload/{owner}/{name}", s.payloadHandler)

	s.srv = &http.Server{
		Handler:      hdlr,
		Addr:         address,
		WriteTimeout: 10 * time.Second,
		ReadTimeout:  10 * time.Second,
	}

	// store incoming hooks on the server object
	go s.storeHooks()

	log.Fatal(s.srv.ListenAndServe())
}

func (s *callbackServer) stop(ctx context.Context) error {
	return s.srv.Shutdown(ctx)
}

func (s *callbackServer) payloadHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	cbHook, ok := s.hookForRepo(vars["owner"], vars["name"])
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("404 Not Found"))
		return
	}

	payload, err := github.ValidatePayload(r, []byte(cbHook.hook.Secret))
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}

	var release *v1alpha1.GitHubRelease
	var active bool
	switch r.Header.Get("X-GitHub-Event") {
	case "ping":
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK!"))
		return
	case "pull_request":
		release, active, err = getPullRequestRelease(payload)
	case "release":
		release, active, err = getOfficialRelease(payload)
	}

	if err := s.storeRelease(&cbHook, release, active); err != nil {
		log.Printf("Could not store release: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK!"))
}

func (s *callbackServer) storeRelease(hook *callbackHook, release *v1alpha1.GitHubRelease, active bool) error {
	if release == nil {
		return nil
	}

	ghr := v1alpha1.GitHubRepository{
		TypeMeta: metav1.TypeMeta{
			Kind:       "GitHubRepository",
			APIVersion: "hlnr.io/v1alpha1",
		},
	}
	if err := s.patcher.Get(&ghr, hook.crdNamespace, hook.crdName); err != nil {
		log.Printf("Could not find GitHubRepository: %s", err)
		return err
	}

	found := false
	newRel := make([]v1alpha1.GitHubRelease, 0, len(ghr.Status.Releases))
	for _, r := range ghr.Status.Releases {
		if r.Name == release.Name {
			found = true
			if !active {
				continue
			}
			release.DeepCopyInto(&r)
		}

		newRel = append(newRel, r)
	}

	ghr.Status.Releases = newRel

	if !found && !active {
		ghr.Status.Releases = append(ghr.Status.Releases, *release)
	}

	if _, err := s.patcher.Apply(&ghr); err != nil {
		log.Printf("Could not update GitHubRepository: %s", err)
		return err
	}

	return nil
}

func (s *callbackServer) hookForRepo(owner, name string) (callbackHook, bool) {
	repo := v1alpha1.GitHubRepositorySpec{
		Repo:  name,
		Owner: owner,
	}
	slug := repo.Slug()
	s.RLock()
	defer s.RUnlock()

	for _, hook := range s.hooks {
		if hook.repo == slug {
			return hook, true
		}
	}

	return callbackHook{}, false
}

func (s *callbackServer) storeHooks() {
	log.Println("Starting hooks check")
	for hook := range s.hooksChan {
		if i, ok := s.isHookPresent(hook); ok {
			if hook.delete {
				log.Printf("Deleting callback for hook %s (%s): %s", hook.crdName, hook.crdNamespace, hook.repo)

				s.Lock()
				s.hooks = append(s.hooks[:i], s.hooks[i+1:]...)
				s.Unlock()
			}

			continue
		}

		log.Printf("Setting up callback for hook %s (%s): %s", hook.crdName, hook.crdNamespace, hook.repo)
		s.Lock()
		s.hooks = append(s.hooks, hook)
		s.Unlock()
	}
}

func (s *callbackServer) isHookPresent(cbh callbackHook) (int, bool) {
	s.RLock()
	defer s.RUnlock()

	for i, hook := range s.hooks {
		if hook.crdName == cbh.crdName && hook.crdNamespace == cbh.crdNamespace && hook.repo == cbh.repo {
			return i, true
		}
	}

	return 0, false
}

func getPullRequestRelease(payload []byte) (*v1alpha1.GitHubRelease, bool, error) {
	pre := &github.PullRequestEvent{}
	if err := json.Unmarshal(payload, pre); err != nil {
		return nil, false, err
	}

	return &v1alpha1.GitHubRelease{
		Name:       *pre.PullRequest.Head.Ref,
		Tag:        *pre.PullRequest.Head.SHA,
		Level:      v1alpha1.SemVerLevelPreview,
		ReleasedAt: releasedAtFromGitHubTimestamp(pre.PullRequest.Head.Repo.UpdatedAt),
	}, *pre.PullRequest.State != "closed", nil
}

func getOfficialRelease(payload []byte) (*v1alpha1.GitHubRelease, bool, error) {
	re := &github.ReleaseEvent{}
	if err := json.Unmarshal(payload, re); err != nil {
		return nil, false, err
	}

	if *re.Release.Draft {
		return nil, false, nil
	}

	lvl := v1alpha1.SemVerLevelRelease
	if re.Release.Prerelease != nil && *re.Release.Prerelease {
		lvl = v1alpha1.SemVerLevelReleaseCandidate
	}

	name := *re.Release.TagName
	if re.Release.Name != nil {
		name = *re.Release.Name
	}

	return &v1alpha1.GitHubRelease{
		Name:       name,
		Tag:        *re.Release.TagName,
		Level:      lvl,
		ReleasedAt: releasedAtFromGitHubTimestamp(re.Release.PublishedAt),
	}, true, nil // There's no webhook for release deletion
}

func releasedAtFromGitHubTimestamp(ts *github.Timestamp) metav1.Time {
	return metav1.NewTime(ts.Time)
}
