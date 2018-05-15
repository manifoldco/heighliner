package githubpolicy

import (
	"context"
	"encoding/base64"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/manifoldco/heighliner/pkg/api/v1alpha1"

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
}

type getClient interface {
	Get(interface{}, string, string) error
}

// NewController returns a new GitHubPolicy Controller.
func NewController(cfg *rest.Config, cs kubernetes.Interface, namespace, domain string) (*Controller, error) {
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

	for _, repo := range ghp.Spec.Repositories {
		if err := syncRepository(c.patcher, ghp.Namespace, repo); err != nil {
			log.Printf("Could not sync repository %s/%s: %s", repo.Owner, repo.Name, err)
		}
	}

	return nil
}

func syncRepository(cl getClient, ns string, repo v1alpha1.GitHubRepository) error {
	authToken, err := getSecretAuthToken(cl, ns, repo.ConfigSecret.Name)
	fmt.Println(string(authToken))
	return err
}

func getSecretAuthToken(cl getClient, namespace, name string) ([]byte, error) {
	configSecret := &v1.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: "v1",
		},
	}

	if err := cl.Get(configSecret, namespace, name); err != nil {
		return nil, err
	}

	data := configSecret.StringData
	secret, ok := data["GITHUB_AUTH_TOKEN"]
	if !ok {
		return nil, fmt.Errorf("GITHUB_AUTH_TOKEN not found in '%s'", name)
	}

	return base64.StdEncoding.DecodeString(secret)
}
