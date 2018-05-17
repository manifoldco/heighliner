package githubpolicy

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/google/go-github/github"
	"github.com/manifoldco/heighliner/pkg/api/v1alpha1"
	"github.com/manifoldco/heighliner/pkg/k8sutils"
	"golang.org/x/oauth2"

	"github.com/jelmersnoeck/kubekit"
	"github.com/jelmersnoeck/kubekit/patcher"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	cmdutil "k8s.io/kubernetes/pkg/kubectl/cmd/util"
)

func init() {
	// let's not hit rate limits
	kubekit.ResyncPeriod = 30 * time.Second
}

// Controller will take care of syncing the internal status of the GitHub Policy
// object with the available releases on GitHub.
type Controller struct {
	rc        *rest.RESTClient
	cs        kubernetes.Interface
	patcher   patchClient
	namespace string
	cfg       Config

	// we'll be sharing data between several goroutines - the controller and
	// callback server. This channel is to share information between the two.
	hooksChan chan callbackHook
}

type webhookClient interface {
	CreateHook(context.Context, string, string, *github.Hook) (*github.Hook, *github.Response, error)
}

// NewController returns a new GitHubPolicy Controller.
func NewController(rcfg *rest.Config, cs kubernetes.Interface, namespace string, cfg Config) (*Controller, error) {
	rc, err := kubekit.RESTClient(rcfg, &v1alpha1.SchemeGroupVersion, v1alpha1.AddToScheme)
	if err != nil {
		return nil, err
	}

	return &Controller{
		cs:        cs,
		rc:        rc,
		patcher:   patcher.New("hlnr-github-policy", cmdutil.NewFactory(nil)),
		namespace: namespace,
		cfg:       cfg,
		hooksChan: make(chan callbackHook),
	}, nil
}

// Run runs the Controller in the background and sets up watchers to take action
// when the desired state is altered.
func (c *Controller) Run() error {
	ctx, cancel := context.WithCancel(context.Background())

	log.Printf("Starting WebHooks server...")
	srv := &callbackServer{
		patcher:   c.patcher,
		hooksChan: c.hooksChan,
	}
	go srv.start(c.cfg.CallbackPort)

	log.Printf("Starting controller...")

	go c.run(ctx)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	<-quit
	log.Printf("Shutdown requested...")
	cancel()

	log.Printf("Shutting down WebHooks server...")
	if err := srv.stop(ctx); err != nil {
		log.Printf("Error shutting down WebHooks server: %s", err)
	}

	<-ctx.Done()
	log.Printf("Shutting down...")

	return nil
}

func (c *Controller) run(ctx context.Context) {
	watcher := kubekit.NewWatcher(
		c.rc,
		c.namespace,
		&GitHubPolicyResource,
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				c.syncPolicy(obj)
			},
			UpdateFunc: func(old, new interface{}) {
				if ok, err := k8sutils.SpecChanges(old, new); ok && err == nil {
					c.syncPolicy(new)
				}
			},
			DeleteFunc: func(obj interface{}) {
				cp := obj.(*v1alpha1.GitHubPolicy).DeepCopy()
				log.Printf("Deleting GitHubPolicy %s", cp.Name)
			},
		},
	)

	go watcher.Run(ctx.Done())
}

func (c *Controller) syncPolicy(obj interface{}) error {
	ghp := obj.(*v1alpha1.GitHubPolicy).DeepCopy()

	for _, repo := range ghp.Spec.Repositories {
		if err := ensureHooks(c.patcher, repo, ghp, c.cfg); err != nil {
			log.Printf("Could not ensure GitHub hooks for %s (%s): %s", repo.Slug(), ghp.Namespace, err)
			continue
		}
	}

	// remove the hooks that are not needed anymore
	if err := cleanStatus(ghp); err != nil {
		log.Printf("Error cleaning up status for %s (%s): %s", ghp.Name, ghp.Namespace, err)
		return err
	}

	// need to specify types again until we resolve the mapping issue
	ghp.TypeMeta = metav1.TypeMeta{
		Kind:       "GitHubPolicy",
		APIVersion: "hlnr.io/v1alpha1",
	}

	// update the status
	if _, err := c.patcher.Apply(ghp); err != nil {
		log.Printf("Error syncing GitHubPolicy %s (%s): %s", ghp.Name, ghp.Namespace, err)
		return err
	}

	return nil
}

func cleanStatus(ghp *v1alpha1.GitHubPolicy) error {
	for slug := range ghp.Status.Hooks {
		found := false
		for _, repo := range ghp.Spec.Repositories {
			if slug == repo.Slug() {
				found = true
				break
			}
		}

		if found {
			continue
		}

		// TODO(jelmer): ideally we'd try and deregister the hook as well, but
		// we need to match this up with the "old" version of the repository.
		delete(ghp.Status.Hooks, slug)
	}

	for slug := range ghp.Status.Releases {
		found := false
		for _, repo := range ghp.Spec.Repositories {
			if slug == repo.Slug() {
				found = true
				break
			}
		}

		if found {
			continue
		}

		// TODO(jelmer): ideally we'd try and deregister the hook as well, but
		// we need to match this up with the "old" version of the repository.
		delete(ghp.Status.Releases, slug)
	}

	return nil
}

func ensureHooks(cl getClient, repo v1alpha1.GitHubRepository, ghp *v1alpha1.GitHubPolicy, cfg Config) error {
	authToken, err := getSecretAuthToken(cl, ghp.Namespace, repo.ConfigSecret.Name)
	if err != nil {
		return err
	}

	ctx := context.Background()

	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: authToken},
	)
	tc := oauth2.NewClient(ctx, ts)
	client := github.NewClient(tc)

	ghHook, ok := ghp.Status.Hooks[repo.Slug()]
	if ok {
		// hooks are set, do an update
		hook := &github.Hook{
			Name:   k8sutils.PtrString("web"),
			Active: k8sutils.PtrBool(true),
			Events: []string{
				"pull_request",
				"release",
			},
			Config: map[string]interface{}{
				"secret":       ghHook.Secret,
				"url":          cfg.PayloadURL(repo.Owner, repo.Name),
				"content_type": "json",
				"insecure_ssl": cfg.InsecureSSL,
			},
		}

		hook, rsp, err := client.Repositories.EditHook(ctx, repo.Owner, repo.Name, ghHook.ID, hook)
		if err != nil {
			if rsp != nil && rsp.StatusCode == http.StatusNotFound {
				ghHook, err = createWebhook(ctx, client.Repositories, repo.Owner, repo.Name, cfg)
				if err != nil {
					return err
				}
			} else {
				return err
			}
		} else {
			ghHook = v1alpha1.GitHubHook{
				ID:     *hook.ID,
				Secret: ghHook.Secret,
			}
		}
	} else {
		// hooks are not set, create them
		ghHook, err = createWebhook(ctx, client.Repositories, repo.Owner, repo.Name, cfg)
		if err != nil {
			return err
		}
	}

	if ghp.Status.Hooks == nil {
		ghp.Status.Hooks = map[string]v1alpha1.GitHubHook{}
	}

	ghp.Status.Hooks[repo.Slug()] = ghHook
	return nil
}

func createWebhook(ctx context.Context, cl webhookClient, owner, name string, cfg Config) (v1alpha1.GitHubHook, error) {
	secret := k8sutils.RandomString(32)

	hook := &github.Hook{
		Name:   k8sutils.PtrString("web"),
		Active: k8sutils.PtrBool(true),
		Events: []string{
			"pull_request",
			"release",
		},
		Config: map[string]interface{}{
			"secret":       secret,
			"url":          cfg.PayloadURL(owner, name),
			"content_type": "json",
			"insecure_ssl": cfg.InsecureSSL,
		},
	}

	hook, _, err := cl.CreateHook(ctx, owner, name, hook)
	if err != nil {
		return v1alpha1.GitHubHook{}, err
	}

	return v1alpha1.GitHubHook{
		ID:     *hook.ID,
		Secret: secret,
	}, nil
}

func getSecretAuthToken(cl getClient, namespace, name string) (string, error) {
	configSecret := &v1.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: "v1",
		},
	}

	if err := cl.Get(configSecret, namespace, name); err != nil {
		return "", err
	}

	data := configSecret.Data
	secret, ok := data["GITHUB_AUTH_TOKEN"]
	if !ok {
		return "", fmt.Errorf("GITHUB_AUTH_TOKEN not found in '%s'", name)
	}

	return string(secret), nil
}

func (c *Controller) storeHook(crd *v1alpha1.GitHubPolicy, repositorySlug string, hook v1alpha1.GitHubHook) {
	cbh := callbackHook{
		crdName:      crd.Name,
		crdNamespace: crd.Namespace,
		repo:         repositorySlug,
		hook:         hook,
	}

	c.hooksChan <- cbh
}

func (c *Controller) deleteHook(crd *v1alpha1.GitHubPolicy, repositorySlug string, hook v1alpha1.GitHubHook) {
	cbh := callbackHook{
		crdName:      crd.Name,
		crdNamespace: crd.Namespace,
		repo:         repositorySlug,
		hook:         hook,
		delete:       true,
	}

	c.hooksChan <- cbh
}
