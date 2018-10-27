package networkpolicy

import (
	"fmt"
	"reflect"
	"testing"

	"k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/jelmersnoeck/kubekit/patcher"
	"github.com/manifoldco/heighliner/apis/heighliner/v1alpha1"
	"github.com/manifoldco/heighliner/internal/tester"
)

func TestGroupReleases(t *testing.T) {
	t.Run("with a set of semver releases", func(t *testing.T) {
		t.Run("with different PR applications", func(t *testing.T) {
			releases := []v1alpha1.Release{
				{
					Image: "hlnr.io/test:1.2.3-pr.456-pr+201804281301",
					SemVer: &v1alpha1.SemVerRelease{
						Name:    "456-pr",
						Version: "0.1.0",
					},
					Level: v1alpha1.SemVerLevelPreview,
				},
				{
					Image: "hlnr.io/test:1.2.3-pr.456-pr+201804281308",
					SemVer: &v1alpha1.SemVerRelease{
						Name:    "456-pr",
						Version: "0.1.1",
					},
					Level: v1alpha1.SemVerLevelPreview,
				},
				{
					Image: "hlnr.io/test:1.2.3-pr.457-pr+201804281307",
					SemVer: &v1alpha1.SemVerRelease{
						Name:    "457-pr",
						Version: "0.1.0",
					},
					Level: v1alpha1.SemVerLevelPreview,
				},
			}

			results := groupReleases("test-deploy", releases)
			expectedLength := 2
			if len(results) != expectedLength {
				t.Errorf("Expected length to be %d, got %d", expectedLength, len(results))
			}
		})
	})
}

func TestController_syncNetworking(t *testing.T) {
	pc := new(tester.PatchClient)
	ki := fake.NewSimpleClientset()
	kc := ki.Core()
	ctrl := &Controller{
		cs:      kc,
		patcher: pc,
	}

	t.Run("without linked microservice", func(t *testing.T) {
		defer pc.Flush()

		myErr := fmt.Errorf("my error")
		pc.GetFunc = func(obj interface{}, namespace, name string) error {
			if _, ok := obj.(*v1alpha1.Microservice); !ok {
				t.Errorf("Expected object to be Microservice, got %T", obj)
			}

			return myErr
		}

		np := &v1alpha1.NetworkPolicy{}
		if err := ctrl.syncNetworking(np); err != myErr {
			t.Errorf("Expected '%s', got '%s'", myErr, err)
		}
	})

	t.Run("with linked microservice", func(t *testing.T) {
		t.Run("without any releases", func(t *testing.T) {
			defer pc.Flush()

			np := &v1alpha1.NetworkPolicy{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "unit-tests",
					Namespace: "testing",
				},
				Spec: v1alpha1.NetworkPolicySpec{
					Microservice: &v1.LocalObjectReference{
						Name: "unit-test",
					},
				},
			}

			pc.GetFunc = func(obj interface{}, namespace, name string) error {
				switch obj := obj.(type) {
				case *v1alpha1.Microservice:
					if name != "unit-test" {
						t.Fatalf("Expected Microservice name to be `unit-test`, got `%s`", name)
					}

					if namespace != "testing" {
						t.Fatalf("Expected Microservice namespace to be `testing`, got `%s`", namespace)
					}

					obj.Status.Releases = []v1alpha1.Release{}
					return nil
				}

				return fmt.Errorf("Object %T not supported", obj)
			}

			var npApplied bool
			pc.ApplyFunc = func(obj runtime.Object, opts ...patcher.OptionFunc) ([]byte, error) {
				switch obj.(type) {
				case *v1alpha1.NetworkPolicy:
					npApplied = true
					return nil, nil
				}

				return nil, fmt.Errorf("Object %T not supported", obj)
			}

			if err := ctrl.syncNetworking(np); err != nil {
				t.Errorf("Expected no error, got '%s'", err)
			}

			if npApplied {
				t.Errorf("Didn't expect NetworkPolicy to receive an update")
			}
		})

		t.Run("with a release", func(t *testing.T) {
			defer pc.Flush()

			np := &v1alpha1.NetworkPolicy{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "unit-tests",
					Namespace: "testing",
				},
				Spec: v1alpha1.NetworkPolicySpec{
					Microservice: &v1.LocalObjectReference{
						Name: "unit-test",
					},
					Ports: []v1alpha1.NetworkPort{
						{
							Name:       "headless",
							TargetPort: 8080,
							Port:       80,
						},
					},
					ExternalDNS: []v1alpha1.ExternalDNS{
						{
							Domain: "{{.StreamName}}.reviews.hlnr.io",
							Port:   "headless",
						},
					},
				},
			}

			pc.GetFunc = func(obj interface{}, namespace, name string) error {
				switch obj := obj.(type) {
				case *v1alpha1.Microservice:
					if name != "unit-test" {
						t.Fatalf("Expected Microservice name to be `unit-test`, got `%s`", name)
					}

					if namespace != "testing" {
						t.Fatalf("Expected Microservice namespace to be `testing`, got `%s`", namespace)
					}

					obj.ObjectMeta = metav1.ObjectMeta{
						Name:      name,
						Namespace: namespace,
					}
					obj.Status.Releases = []v1alpha1.Release{
						{
							Level: v1alpha1.SemVerLevelRelease,
							SemVer: &v1alpha1.SemVerRelease{
								Name:    "unit-test",
								Version: "1.2.3",
							},
						},
					}
					return nil
				case *v1.Service:
					return errors.NewNotFound(schema.GroupResource{}, name)
				}

				return fmt.Errorf("Object %T not supported", obj)
			}

			var ingApplied, npApplied bool
			pc.ApplyFunc = func(obj runtime.Object, opts ...patcher.OptionFunc) ([]byte, error) {
				switch obj := obj.(type) {
				case *v1alpha1.NetworkPolicy:
					npApplied = true

					expectedSemVer := v1alpha1.SemVerRelease{
						Name:    "unit-test",
						Version: "1.2.3",
					}
					if !reflect.DeepEqual(*obj.Status.Domains[0].SemVer, expectedSemVer) {
						t.Errorf("Expected semVer \n\n%#v\n\n to be \n\n%#v\n\n", *obj.Status.Domains[0].SemVer, expectedSemVer)
					}

					expectedStatus := v1alpha1.NetworkPolicyStatus{
						Domains: []v1alpha1.Domain{
							{
								URL:    "https://unit-test.reviews.hlnr.io",
								SemVer: obj.Status.Domains[0].SemVer,
							},
						},
					}

					if !reflect.DeepEqual(obj.Status, expectedStatus) {
						t.Errorf("Expected status \n\n%#v\n\n to be \n\n%#v\n\n", obj.Status, expectedStatus)
					}

					return nil, nil
				case *v1beta1.Ingress:
					expectedRules := v1beta1.HTTPIngressRuleValue{
						Paths: []v1beta1.HTTPIngressPath{
							{
								Path: "/",
								Backend: v1beta1.IngressBackend{
									ServiceName: "unit-test",
									ServicePort: intstr.FromString("headless"),
								},
							},
						},
					}

					if !reflect.DeepEqual(*obj.Spec.Rules[0].IngressRuleValue.HTTP, expectedRules) {
						t.Errorf("Expected IngressRules \n\n%#v\n\n to equal \n\n%#v\n\n", *obj.Spec.Rules[0].IngressRuleValue.HTTP, expectedRules)
					}

					expectedIngress := &v1beta1.Ingress{
						TypeMeta: metav1.TypeMeta{
							Kind:       "Ingress",
							APIVersion: "extensions/v1beta1",
						},
						ObjectMeta: metav1.ObjectMeta{
							Name:      "unit-test",
							Namespace: "testing",
							Labels: map[string]string{
								"hlnr.io/microservice.version":   "1.2.3",
								"hlnr.io/service":                "unit-tests",
								"hlnr.io/microservice.full_name": "unit-test-a5hpt4kc",
								"hlnr.io/microservice.name":      "unit-test",
								"hlnr.io/microservice.release":   "unit-test",
							},
							Annotations: map[string]string{
								"external-dns.alpha.kubernetes.io/ttl":      "300",
								"hlnr.io/version":                           "v1alpha1",
								"hlnr.io/component":                         "NetworkPolicy",
								"kubernetes.io/ingress.class":               "nginx",
								"external-dns.alpha.kubernetes.io/hostname": "unit-test.reviews.hlnr.io",
							},
							OwnerReferences: obj.OwnerReferences,
						},
						Spec: v1beta1.IngressSpec{
							TLS: []v1beta1.IngressTLS{
								{
									Hosts:      []string{"unit-test.reviews.hlnr.io"},
									SecretName: "heighliner-components"},
							},
							Rules: []v1beta1.IngressRule{
								{
									Host:             "unit-test.reviews.hlnr.io",
									IngressRuleValue: obj.Spec.Rules[0].IngressRuleValue, // this is a pointer, we've checked the value before, reassing the pointer
								},
							},
						},
					}

					if !reflect.DeepEqual(obj, expectedIngress) {
						t.Errorf("Expected Applied Ingress \n\n%#v\n\n to match ExpectedIngress\n\n%#v\n\n", obj.Spec, expectedIngress.Spec)
					}

					ingApplied = true
					return nil, nil
				}

				return nil, fmt.Errorf("Object %T not supported", obj)
			}

			if err := ctrl.syncNetworking(np); err != nil {
				t.Errorf("Expected no error, got '%s'", err)
			}

			// canary deployment test, we always deploy a specific service
			testSvc, err := kc.Services("testing").Get("unit-test-a5hpt4kc", metav1.GetOptions{})
			if err != nil {
				t.Errorf("Expected no error fetching the canary service, got %s", err)
			}
			expectedTestSvc := &v1.Service{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Service",
					APIVersion: "v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "unit-test-a5hpt4kc",
					Namespace: "testing",
					Labels: map[string]string{
						"hlnr.io/microservice.release":   "unit-test",
						"hlnr.io/microservice.version":   "1.2.3",
						"hlnr.io/service":                "unit-tests",
						"hlnr.io/microservice.full_name": "unit-test-a5hpt4kc",
						"hlnr.io/microservice.name":      "unit-test",
					},
					Annotations: map[string]string{
						"hlnr.io/component": "NetworkPolicy",
						"hlnr.io/version":   "v1alpha1",
					},
				},
				Spec: v1.ServiceSpec{
					Ports: []v1.ServicePort{
						{
							Name:       "headless",
							Protocol:   v1.ProtocolTCP,
							Port:       80,
							TargetPort: intstr.FromInt(8080),
						},
					},
					Selector: map[string]string{
						"hlnr.io/microservice.full_name": "unit-test-a5hpt4kc",
						"hlnr.io/microservice.name":      "unit-test",
						"hlnr.io/microservice.release":   "unit-test",
						"hlnr.io/microservice.version":   "1.2.3",
					},
					Type:            v1.ServiceTypeNodePort,
					SessionAffinity: v1.ServiceAffinityNone,
				},
			}
			if !reflect.DeepEqual(testSvc, expectedTestSvc) {
				t.Errorf("Expected testSvc to equal expectedTestSvc")
			}

			// service that is linked to the ingress, marking the active deploy
			ingressSvc, err := kc.Services("testing").Get("unit-test", metav1.GetOptions{})
			if err != nil {
				t.Errorf("Expected no error fetching the production service, got %s", err)
			}

			expectedIngressSvc := testSvc
			expectedIngressSvc.Name = "unit-test"
			if !reflect.DeepEqual(ingressSvc, expectedIngressSvc) {
				t.Errorf("Expected ingressSvc to equal expectedIngressSvc")
			}

			if !ingApplied {
				t.Errorf("Expected an Ingress to be created for the NetworkPolicy")
			}

			if !npApplied {
				t.Fatalf("Expected the NetworkPolicy to have received an updated")
			}
		})
	})
}
