package githubrepository

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/google/go-github/github"
	"github.com/gorilla/mux"
	"github.com/jelmersnoeck/kubekit/patcher"
	"github.com/manifoldco/heighliner/pkg/api/v1alpha1"
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

	fmt.Println(string(payload))
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
