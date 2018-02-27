package k8sutils

import (
	"flag"

	"k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// Clientset creates an in cluster configured clientset.
func Clientset() (*rest.Config, kubernetes.Interface, clientset.Interface, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, nil, nil, err
	}

	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, nil, nil, err
	}

	apiextClientset, err := clientset.NewForConfig(config)
	if err != nil {
		return nil, nil, nil, err
	}

	return config, client, apiextClientset, nil
}

// SchemeBuilder represents what the interface of a SchemeBuilder.
type SchemeBuilder func(*runtime.Scheme) error

// RESTClient configures a new REST Client to be able to understand all the
// schemes defined. This way users can query objects associated with this
// scheme.
func RESTClient(cfg *rest.Config, sgv *schema.GroupVersion, schemeBuilders ...SchemeBuilder) (*rest.RESTClient, error) {
	scheme := runtime.NewScheme()

	for _, builder := range schemeBuilders {
		if err := builder(scheme); err != nil {
			return nil, err
		}
	}

	config := *cfg
	config.GroupVersion = sgv
	config.APIPath = "/apis"
	config.ContentType = runtime.ContentTypeJSON
	config.NegotiatedSerializer = serializer.DirectCodecFactory{
		CodecFactory: serializer.NewCodecFactory(scheme),
	}

	return rest.RESTClientFor(&config)
}

func init() {
	flag.Parse()
}
