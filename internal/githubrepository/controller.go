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
	"github.com/jelmersnoeck/kubekit"
	"github.com/jelmersnoeck/kubekit/patcher"
	"github.com/manifoldco/heighliner/apis/heighliner/v1alpha1"
	"github.com/manifoldco/heighliner/internal/k8sutils"
	"github.com/manifoldco/heighliner/internal/networkpolicy"
	"golang.org/x/oauth2"
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

type deploymentClient interface {
	CreateDeployment(context.Context, string, string, *github.DeploymentRequest) (*github.Deployment, *github.Response, error)
	CreateDeploymentStatus(context.Context, string, string, int64, *github.DeploymentStatusRequest) (*github.DeploymentStatus, *github.Response, error)
	ListDeploymentStatuses(context.Context, string, string, int64, *github.ListOptions) ([]*github.DeploymentStatus, *github.Response, error)
}

const authTokenKey = "GITHUB_AUTH_TOKEN"

// NewController returns a new GitHubRepository Controller.
func NewController(rcfg *rest.Config, cs kubernetes.Interface, namespace string, cfg Config) (*Controller, error) {
	// Let's not hit rate limits.
	// This is done here instead of an init function so we don't override the
	// global settings for other controllers.
	kubekit.ResyncPeriod = 30 * time.Second

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

	log.Printf("Watching for GitHub changes every %s", c.cfg.ReconciliationPeriod)

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
	repoWatcher := kubekit.NewWatcher(
		c.rc,
		c.namespace,
		&GitHubRepositoryResource,
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				c.syncPolicy(obj)
			},
			UpdateFunc: func(old, new interface{}) {
				c.syncPolicy(new)
			},
			DeleteFunc: func(obj interface{}) {
				cp := obj.(*v1alpha1.GitHubRepository).DeepCopy()
				log.Printf("Deleting GitHubRepository %s", cp.Name)
				c.deleteHooks(obj)
			},
		},
	)

	go repoWatcher.Run(ctx.Done())

	npWatcher := kubekit.NewWatcher(
		c.rc,
		c.namespace,
		&networkpolicy.NetworkPolicyResource,
		cache.ResourceEventHandlerFuncs{
			AddFunc:    func(obj interface{}) { c.syncDeployment(obj, false) },
			UpdateFunc: func(old, new interface{}) { c.syncDeployment(new, false) },
			DeleteFunc: func(obj interface{}) { c.syncDeployment(obj, true) },
		},
	)

	go npWatcher.Run(ctx.Done())

}

func (c *Controller) deleteHooks(obj interface{}) error {
	ghp := obj.(*v1alpha1.GitHubRepository).DeepCopy()

	if ghp.Status.Webhook == nil {
		return nil
	}

	repo := ghp.Spec
	ctx := context.Background()

	client, err := getGitHubClient(ctx, c.patcher, ghp.Namespace, repo.ConfigSecret.Name)
	if err != nil {
		log.Printf("Could not get GitHub client: %s", err)
		return err
	}

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

	ctx := context.Background()

	ghClient, err := getGitHubClient(ctx, c.patcher, ghp.Namespace, ghp.Spec.ConfigSecret.Name)
	if err != nil {
		log.Printf("Could not create GitHub cleint for %s (%s): %s", ghp.Spec.Slug(), ghp.Namespace, err)
		return err
	}

	hook, err := c.ensureHooks(c.patcher, ghp, c.cfg)
	if err != nil {
		log.Printf("Could not ensure GitHub hooks for %s (%s): %s", ghp.Spec.Slug(), ghp.Namespace, err)
		return err
	}

	rc := githubReconciliationClient{Client: ghClient}
	err = reconciliateRepository(ctx, &rc, ghp, c.cfg.ReconciliationPeriod)
	if err != nil {
		log.Printf("Could sync GitHub repo for %s (%s): %s", ghp.Spec.Slug(), ghp.Namespace, err)
		return err
	}

	ghp.Status.Webhook = &v1alpha1.GitHubHook{
		ID:     hook.ID,
		Secret: hook.Secret,
	}

	// need to specify types again until we resolve the mapping issue
	ghp.TypeMeta = metav1.TypeMeta{
		Kind:       "GitHubRepository",
		APIVersion: "hlnr.io/v1alpha1",
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

func (c *Controller) syncDeployment(obj interface{}, deleted bool) {
	np := obj.(*v1alpha1.NetworkPolicy)

	// Find the microservice referenced by this network policy
	msvcName := np.Name
	if np.Spec.Microservice == nil {
		msvcName = np.Spec.Microservice.Name
	}

	msvc := v1alpha1.Microservice{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Microservice",
			APIVersion: "hlnr.io/v1alpha1",
		},
	}
	if err := c.patcher.Get(&msvc, np.Namespace, msvcName); err != nil {
		log.Print("Error fetching Microservice:", err)
		return
	}

	// Find the ImagePolicy for the microservice
	ip := v1alpha1.ImagePolicy{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ImagePolicy",
			APIVersion: "hlnr.io/v1alpha1",
		},
	}
	if err := c.patcher.Get(&ip, msvc.Namespace, msvc.Spec.ImagePolicy.Name); err != nil {
		log.Print("Error fetching ImagePolicy:", err)
		return
	}

	if ip.Spec.Filter.GitHub == nil { // ignore image policies that aren't for github
		return
	}

	// Find the relevant GitHubRepository
	ghr := v1alpha1.GitHubRepository{
		TypeMeta: metav1.TypeMeta{
			Kind:       "GitHubRepository",
			APIVersion: "hlnr.io/v1alpha1",
		},
	}

	ghNamespace := ip.Spec.Filter.GitHub.Namespace
	if ghNamespace == "" {
		ghNamespace = msvc.Namespace
	}

	if err := c.patcher.Get(&ghr, ghNamespace, ip.Spec.Filter.GitHub.Name); err != nil {
		log.Print("Error fetching GitHubRepository:", err)
		return
	}

	changed, newReleases := reconcileDeployments(np.Status.Domains, deleted, ghr.Status.Releases)
	if len(changed) == 0 {
		return
	}

	// Fix the network policy reference. reconciliation doesn't care about it.
	npr := v1.ObjectReference{
		Name:      np.Name,
		Namespace: np.Namespace,
	}
	for i := range newReleases {
		if newReleases[i].Deployment != nil {
			newReleases[i].Deployment.NetworkPolicy = npr
		}
	}
	ghr.Status.Releases = newReleases

	ctx := context.Background()
	ghClient, err := getGitHubClient(ctx, c.patcher, ghr.Namespace, ghr.Spec.ConfigSecret.Name)
	if err != nil {
		log.Printf("Could not fetch client: %s", err)
		return
	}

	// Create deployment / status in github
	for _, idx := range changed {
		id, err := createGitHubDeployment(ctx, ghClient.Repositories, &ghr, newReleases[idx])
		if id != nil {
			newReleases[idx].Deployment.ID = id
		}

		if err != nil {
			log.Print("Error creating GitHub deployment:", err)
			continue // try the rest of the changes
		}
	}

	// persist state back to k8s
	if _, err := c.patcher.Apply(&ghr); err != nil {
		log.Printf("Error syncing GitHubRepository %s (%s): %s", ghr.Name, ghr.Namespace, err)
	}
}

func createGitHubDeployment(ctx context.Context, cl deploymentClient, repo *v1alpha1.GitHubRepository, release v1alpha1.GitHubRelease) (*int64, error) {
	id := release.Deployment.ID
	if id == nil {
		dpl := &github.DeploymentRequest{
			AutoMerge:            k8sutils.PtrBool(false),
			Description:          k8sutils.PtrString("Heighliner Deployment"),
			Ref:                  k8sutils.PtrString(release.Tag),
			TransientEnvironment: k8sutils.PtrBool(release.Name != repo.Spec.Repo),
			Environment:          release.Deployment.URL,
			RequiredContexts:     &[]string{},
		}

		deploy, _, err := cl.CreateDeployment(ctx, repo.Spec.Owner, repo.Spec.Repo, dpl)
		if err != nil {
			return nil, err
		}

		id = deploy.ID
	}

	status := &github.DeploymentStatusRequest{
		AutoInactive:   k8sutils.PtrBool(false), // we control ageing these off
		State:          &release.Deployment.State,
		EnvironmentURL: release.Deployment.URL,
	}

	// Check the last status to see if we need to create a new one.
	opt := &github.ListOptions{PerPage: 10}
	var prevStatus *github.DeploymentStatus
	for {
		statuses, resp, err := cl.ListDeploymentStatuses(ctx, repo.Spec.Owner, repo.Spec.Repo, *id, opt)
		if err != nil {
			return nil, err
		}

		if len(statuses) > 0 {
			prevStatus = statuses[len(statuses)-1]
		}

		if resp.NextPage == 0 {
			break
		}

		opt.Page = resp.NextPage
	}

	// XXX unfortunately we can't check the environment url.
	if prevStatus != nil && prevStatus.GetState() == status.GetState() {
		return id, nil
	}

	_, _, err := cl.CreateDeploymentStatus(ctx, repo.Spec.Owner, repo.Spec.Repo, *id, status)
	return id, err
}

// reconcileDeployments reconciles the list of provided domains and their
// deleted state with the releases. It ignores releases the domains to not
// reference.
//
// XXX because this only looks at a single networkpolicy's domains, if we delete
// a networkpolicy, and error while reconciling, we'll miss the deletion until
// we add a fill reconciliation on the github repository itself.
func reconcileDeployments(domains []v1alpha1.Domain, deleted bool, releases []v1alpha1.GitHubRelease) ([]int, []v1alpha1.GitHubRelease) {
	changed := make([]int, 0, len(releases))
	newReleases := make([]v1alpha1.GitHubRelease, 0, len(releases))

	for i, r := range releases {
		for _, d := range domains {
			if d.SemVer.Name != r.Name || d.SemVer.Version != r.Tag {
				continue
			}

			if r.Deployment == nil && !deleted {
				changed = append(changed, i)
				r.Deployment = &v1alpha1.Deployment{
					State: "success",
					URL:   &d.URL,
				}
			}

			if r.Deployment != nil && deleted && r.Deployment.State != "inactive" {
				r.Deployment.URL = nil
				r.Deployment.State = "inactive"
				changed = append(changed, i)
			}

			break
		}

		newReleases = append(newReleases, r)
	}

	return changed, newReleases
}

func getGitHubClient(ctx context.Context, cl getClient, namespace, name string) (*github.Client, error) {
	authToken, err := getSecretAuthToken(cl, namespace, name)
	if err != nil {
		log.Printf("Could not get authToken for repository %s (%s): %s", name, namespace, err)
		return nil, err
	}

	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: authToken},
	)
	tc := oauth2.NewClient(ctx, ts)
	return github.NewClient(tc), nil
}
