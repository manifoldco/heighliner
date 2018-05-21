package githubpolicy

import (
	"context"
	"errors"
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

const authTokenKey = "GITHUB_AUTH_TOKEN"

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
				c.deleteHooks(obj)
			},
		},
	)

	go watcher.Run(ctx.Done())
}

func (c *Controller) deleteHooks(obj interface{}) error {
	ghp := obj.(*v1alpha1.GitHubPolicy).DeepCopy()

	for _, hook := range ghp.Status.Hooks {
		ctx := context.Background()

		repo, err := hookRepository(hook, ghp.Spec.Repositories)
		if err != nil {
			log.Printf("Could not get hook for repository %s (%s): %s", ghp.Name, ghp.Namespace, err)
			continue
		}

		authToken, err := getSecretAuthToken(c.patcher, ghp.Namespace, repo.ConfigSecret.Name)
		if err != nil {
			log.Printf("Could not get authToken for repository %s (%s): %s", ghp.Name, ghp.Namespace, err)
			continue
		}

		ts := oauth2.StaticTokenSource(
			&oauth2.Token{AccessToken: authToken},
		)
		tc := oauth2.NewClient(ctx, ts)
		client := github.NewClient(tc)

		if _, err := client.Repositories.DeleteHook(ctx, hook.Owner, hook.Repo, hook.ID); err != nil {
			log.Printf("Could not delete hook from GitHub for %s (%s): %s", ghp.Name, ghp.Namespace, err)
			continue
		}

		wcfg := webhookConfig{
			repo:      repo.Name,
			owner:     repo.Owner,
			slug:      repo.Slug(),
			name:      ghp.Name,
			namespace: ghp.Namespace,
		}

		// delete the webhook from the callback server
		c.propagateHook(wcfg, hook, true)
	}

	return nil
}

func hookRepository(hook v1alpha1.GitHubHook, repos []v1alpha1.GitHubRepository) (v1alpha1.GitHubRepository, error) {
	for _, repo := range repos {
		if repo.Name == hook.Repo && repo.Owner == hook.Owner {
			return repo, nil
		}
	}

	return v1alpha1.GitHubRepository{}, errors.New("GitHubRepository not found")
}

func (c *Controller) syncPolicy(obj interface{}) error {
	ghp := obj.(*v1alpha1.GitHubPolicy).DeepCopy()

	for _, repo := range ghp.Spec.Repositories {
		if err := c.ensureHooks(c.patcher, repo, ghp, c.cfg); err != nil {
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

func (c *Controller) ensureHooks(cl getClient, repo v1alpha1.GitHubRepository, ghp *v1alpha1.GitHubPolicy, cfg Config) error {
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

	wcfg := webhookConfig{
		name:        ghp.Name,
		namespace:   ghp.Namespace,
		payloadURL:  cfg.PayloadURL(repo.Owner, repo.Name),
		insecureSSL: cfg.InsecureSSL,
		owner:       repo.Owner,
		repo:        repo.Name,
		slug:        repo.Slug(),
	}

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
				"url":          wcfg.payloadURL,
				"content_type": "json",
				"insecure_ssl": wcfg.insecureSSL,
			},
		}

		hook, rsp, err := client.Repositories.EditHook(ctx, repo.Owner, repo.Name, ghHook.ID, hook)
		if err != nil {
			if rsp != nil && rsp.StatusCode == http.StatusNotFound {
				ghHook, err = c.createWebhook(ctx, client.Repositories, wcfg)
				if err != nil {
					return err
				}
			} else {
				return err
			}
		} else {
			ghHook = v1alpha1.GitHubHook{
				Owner:  repo.Owner,
				Repo:   repo.Name,
				ID:     *hook.ID,
				Secret: ghHook.Secret,
			}

			c.propagateHook(wcfg, ghHook, false)
		}
	} else {
		// hooks are not set, create them
		ghHook, err = c.createWebhook(ctx, client.Repositories, wcfg)
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

func (c *Controller) createWebhook(ctx context.Context, cl webhookClient, cfg webhookConfig) (v1alpha1.GitHubHook, error) {
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
			"url":          cfg.payloadURL,
			"content_type": "json",
			"insecure_ssl": cfg.insecureSSL,
		},
	}

	hook, _, err := cl.CreateHook(ctx, cfg.owner, cfg.repo, hook)
	if err != nil {
		return v1alpha1.GitHubHook{}, err
	}

	ghHook := v1alpha1.GitHubHook{
		Repo:   cfg.repo,
		Owner:  cfg.owner,
		ID:     *hook.ID,
		Secret: secret,
	}

	// send it to the callbackserver
	c.propagateHook(cfg, ghHook, false)

	return ghHook, nil
}

type webhookConfig struct {
	// gh information
	repo        string
	owner       string
	slug        string
	payloadURL  string
	insecureSSL bool

	// crd information
	name      string
	namespace string
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
	secret, ok := data[authTokenKey]
	if !ok {
		return "", fmt.Errorf("%s not found in '%s'", authTokenKey, name)
	}

	return string(secret), nil
}

func (c *Controller) propagateHook(cfg webhookConfig, hook v1alpha1.GitHubHook, delete bool) {
	cbh := callbackHook{
		crdName:      cfg.name,
		crdNamespace: cfg.namespace,
		repo:         cfg.slug,
		hook:         hook,
		delete:       delete,
	}

	c.hooksChan <- cbh
}
