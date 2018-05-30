package githubrepository

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

const authTokenKey = "GITHUB_AUTH_TOKEN"

// NewController returns a new GitHubRepository Controller.
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
		&GitHubRepositoryResource,
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				c.syncPolicy(obj)
			},
			UpdateFunc: func(old, new interface{}) {
				if ok, err := k8sutils.ShouldSync(old, new); ok && err == nil {
					c.syncPolicy(new)
				}
			},
			DeleteFunc: func(obj interface{}) {
				cp := obj.(*v1alpha1.GitHubRepository).DeepCopy()
				log.Printf("Deleting GitHubRepository %s", cp.Name)
				c.deleteHooks(obj)
			},
		},
	)

	go watcher.Run(ctx.Done())
}

func (c *Controller) deleteHooks(obj interface{}) error {
	ghp := obj.(*v1alpha1.GitHubRepository).DeepCopy()

	if ghp.Status.Webhook == nil {
		return nil
	}

	repo := ghp.Spec
	ctx := context.Background()

	authToken, err := getSecretAuthToken(c.patcher, ghp.Namespace, repo.ConfigSecret.Name)
	if err != nil {
		log.Printf("Could not get authToken for repository %s (%s): %s", ghp.Name, ghp.Namespace, err)
		return err
	}

	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: authToken},
	)
	tc := oauth2.NewClient(ctx, ts)
	client := github.NewClient(tc)

	if _, err := client.Repositories.DeleteHook(ctx, repo.Owner, repo.Repo, *ghp.Status.Webhook.ID); err != nil {
		log.Printf("Could not delete hook from GitHub for %s (%s): %s", ghp.Name, ghp.Namespace, err)
		return err
	}

	wcfg := webhookConfig{
		repo:      repo.Repo,
		owner:     repo.Owner,
		slug:      repo.Slug(),
		name:      ghp.Name,
		namespace: ghp.Namespace,
	}

	// delete the webhook from the callback server
	c.propagateHook(wcfg, ghp.Status.Webhook, true)

	return nil
}

func (c *Controller) syncPolicy(obj interface{}) error {
	ghp := obj.(*v1alpha1.GitHubRepository).DeepCopy()

	hook, err := c.ensureHooks(c.patcher, ghp, c.cfg)
	if err != nil {
		log.Printf("Could not ensure GitHub hooks for %s (%s): %s", ghp.Spec.Slug(), ghp.Namespace, err)
		return err
	}

	wh := ghp.Status.Webhook
	if wh != nil && wh.ID != nil && *wh.ID == *hook.ID && wh.Secret == hook.Secret {
		// no change needed
		return nil
	}

	// need to specify types again until we resolve the mapping issue
	ghp.TypeMeta = metav1.TypeMeta{
		Kind:       "GitHubRepository",
		APIVersion: "hlnr.io/v1alpha1",
	}

	ghp.Status.Webhook = &v1alpha1.GitHubHook{
		ID:     hook.ID,
		Secret: hook.Secret,
	}

	// update the status
	if _, err := c.patcher.Apply(ghp); err != nil {
		log.Printf("Error syncing GitHubRepository %s (%s): %s", ghp.Name, ghp.Namespace, err)
		return err
	}

	return nil
}

func (c *Controller) ensureHooks(cl getClient, ghp *v1alpha1.GitHubRepository, cfg Config) (*v1alpha1.GitHubHook, error) {
	authToken, err := getSecretAuthToken(cl, ghp.Namespace, ghp.Spec.ConfigSecret.Name)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()

	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: authToken},
	)
	tc := oauth2.NewClient(ctx, ts)
	client := github.NewClient(tc)

	repo := ghp.Spec
	wcfg := webhookConfig{
		name:        ghp.Name,
		namespace:   ghp.Namespace,
		payloadURL:  cfg.PayloadURL(repo.Owner, repo.Repo),
		insecureSSL: cfg.InsecureSSL,
		owner:       repo.Owner,
		repo:        repo.Repo,
		slug:        repo.Slug(),
	}

	ghHook := ghp.Status.Webhook
	if ghp.Status.Webhook != nil {
		// hooks are set, do an update
		hook := newGHHook(wcfg, ghHook.Secret)
		hook, rsp, err := client.Repositories.EditHook(ctx, repo.Owner, repo.Repo, *ghHook.ID, hook)
		if err != nil {
			if rsp != nil && rsp.StatusCode == http.StatusNotFound {
				ghHook, err = c.createWebhook(ctx, client.Repositories, wcfg)
				if err != nil {
					return nil, err
				}
			} else {
				return nil, err
			}
		} else {
			ghHook = &v1alpha1.GitHubHook{
				ID:     hook.ID,
				Secret: ghHook.Secret,
			}

			c.propagateHook(wcfg, ghHook, false)
		}
	} else {
		// hooks are not set, create them
		ghHook, err = c.createWebhook(ctx, client.Repositories, wcfg)
		if err != nil {
			return nil, err
		}
	}

	return ghHook, nil
}

func (c *Controller) createWebhook(ctx context.Context, cl webhookClient, cfg webhookConfig) (*v1alpha1.GitHubHook, error) {
	secret := k8sutils.RandomString(32)

	hook := newGHHook(cfg, secret)
	hook, _, err := cl.CreateHook(ctx, cfg.owner, cfg.repo, hook)
	if err != nil {
		return nil, err
	}

	ghHook := &v1alpha1.GitHubHook{
		ID:     hook.ID,
		Secret: secret,
	}

	// send it to the callbackserver
	c.propagateHook(cfg, ghHook, false)

	return ghHook, nil
}

func newGHHook(cfg webhookConfig, secret string) *github.Hook {
	return &github.Hook{
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

func (c *Controller) propagateHook(cfg webhookConfig, hook *v1alpha1.GitHubHook, delete bool) {
	cbh := callbackHook{
		crdName:      cfg.name,
		crdNamespace: cfg.namespace,
		repo:         cfg.slug,
		hook:         hook,
		delete:       delete,
	}

	c.hooksChan <- cbh
}
