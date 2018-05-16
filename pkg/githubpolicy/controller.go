package githubpolicy

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

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

// Controller will take care of syncing the internal status of the GitHub Policy
// object with the available releases on GitHub.
type Controller struct {
	rc        *rest.RESTClient
	cs        kubernetes.Interface
	patcher   *patcher.Patcher
	namespace string
	domain    string
	insecure  bool
}

type getClient interface {
	Get(interface{}, string, string) error
}

type webhookClient interface {
	CreateHook(context.Context, string, string, *github.Hook) (*github.Hook, *github.Response, error)
}

type ghConfig struct {
	insecure bool
	domain   string
}

func (c ghConfig) URL() string {
	scheme := "https://"
	if c.insecure {
		scheme = "http://"
	}

	return fmt.Sprintf("%s%s/payload", scheme, c.domain)
}

// NewController returns a new GitHubPolicy Controller.
func NewController(cfg *rest.Config, cs kubernetes.Interface, namespace, domain string, insecure bool) (*Controller, error) {
	rc, err := kubekit.RESTClient(cfg, &v1alpha1.SchemeGroupVersion, v1alpha1.AddToScheme)
	if err != nil {
		return nil, err
	}

	return &Controller{
		cs:        cs,
		rc:        rc,
		patcher:   patcher.New("hlnr-github-policy", cmdutil.NewFactory(nil)),
		namespace: namespace,
		domain:    domain,
		insecure:  insecure,
	}, nil
}

// Run runs the Controller in the background and sets up watchers to take action
// when the desired state is altered.
func (c *Controller) Run() error {
	log.Printf("Starting controller...")
	ctx, cancel := context.WithCancel(context.Background())

	go c.run(ctx)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	<-quit
	log.Printf("Shutdown requested...")
	cancel()

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
				c.syncPolicy(new)
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

	cfg := ghConfig{
		insecure: c.insecure,
		domain:   c.domain,
	}

	for _, repo := range ghp.Spec.Repositories {
		if err := ensureHooks(c.patcher, repo, ghp, cfg); err != nil {
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

func ensureHooks(cl getClient, repo v1alpha1.GitHubRepository, ghp *v1alpha1.GitHubPolicy, cfg ghConfig) error {
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
				"url":          cfg.URL(),
				"content_type": "json",
				"insecure_ssl": cfg.insecure,
			},
		}

		hook, rsp, err := client.Repositories.EditHook(ctx, repo.Owner, repo.Name, ghHook.ID, hook)
		if err != nil {
			if rsp.StatusCode == http.StatusNotFound {
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

func createWebhook(ctx context.Context, cl webhookClient, owner, name string, cfg ghConfig) (v1alpha1.GitHubHook, error) {
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
			"url":          cfg.URL(),
			"content_type": "json",
			"insecure_ssl": cfg.insecure,
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
