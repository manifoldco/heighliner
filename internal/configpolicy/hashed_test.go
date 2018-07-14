package configpolicy

import (
	"crypto/md5"
	"fmt"
	"testing"

	"github.com/manifoldco/heighliner/internal/k8sutils"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func TestHashed_GetEnvVarHash(t *testing.T) {
	client := &getter{}
	client.flush()

	t.Run("with a simple set of values", func(t *testing.T) {
		defer client.flush()

		data := []corev1.EnvVar{
			{
				Name:  "MY_NEW_SECRET",
				Value: "SUPERSECRET_VALUE",
			},
			{
				Name:  "MY_SECRET",
				Value: "SUPERSECRET",
			},
		}

		hash, err := getEnvVarHash(client, "", data)
		if err != nil {
			t.Fatalf("Expected no error, got %s", err)
		}

		// concatenation of secrets requested above
		expected := fmt.Sprintf("%x", md5.Sum([]byte("MY_NEW_SECRET:SUPERSECRET_VALUE;MY_SECRET:SUPERSECRET;")))
		if hashString := fmt.Sprintf("%x", hash); hashString != expected {
			t.Errorf("Expected hash to equal '%s', got '%s'", expected, hashString)
		}
	})

	t.Run("with a set of secret references", func(t *testing.T) {
		defer client.flush()
		client.registerSecret("my-secret", "SUPER_KEY", "super-value")
		client.registerSecret("my-secret", "SECRET_KEY", "super-value")
		client.registerSecret("my-second-secret", "SECRET_KEY", "secret-value")
		expected := fmt.Sprintf("%x", md5.Sum([]byte("MY_NEW_SECRET:super-value;MY_SECRET:secret-value;")))

		data := []corev1.EnvVar{
			{
				Name: "MY_NEW_SECRET",
				ValueFrom: &corev1.EnvVarSource{
					SecretKeyRef: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: "my-secret",
						},
						Key: "SUPER_KEY",
					},
				},
			},
			{
				Name: "MY_SECRET",
				ValueFrom: &corev1.EnvVarSource{
					SecretKeyRef: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: "my-second-secret",
						},
						Key: "SECRET_KEY",
					},
				},
			},
		}

		hash, err := getEnvVarHash(client, "", data)
		if err != nil {
			t.Fatalf("Expected no error, got %s", err)
		}

		if hashString := fmt.Sprintf("%x", hash); hashString != expected {
			t.Errorf("Expected hash to equal '%s', got '%s'", expected, hashString)
		}
	})

	t.Run("with a mix of secret values and secret references", func(t *testing.T) {
		defer client.flush()
		client.registerSecret("my-secret", "SUPER_KEY", "super-value")
		client.registerSecret("my-secret", "SECRET_KEY", "super-value")
		client.registerSecret("my-second-secret", "SECRET_KEY", "secret-value")

		data := []corev1.EnvVar{
			{
				Name: "MY_NEW_SECRET",
				ValueFrom: &corev1.EnvVarSource{
					SecretKeyRef: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: "my-secret",
						},
						Key: "SUPER_KEY",
					},
				},
			},
			{
				Name:  "NEW_SECRET",
				Value: "SUPERSECRET_VALUE",
			},
			{
				Name: "MY_SECOND_SECRET",
				ValueFrom: &corev1.EnvVarSource{
					SecretKeyRef: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: "my-second-secret",
						},
						Key: "SECRET_KEY",
					},
				},
			},
			{
				Name:  "MY_SECRET",
				Value: "SUPERSECRET",
			},
		}

		hash, err := getEnvVarHash(client, "", data)
		if err != nil {
			t.Fatalf("Expected no error, got %s", err)
		}

		// concatenation of secrets requested above
		expected := fmt.Sprintf(
			"%x",
			md5.Sum([]byte("MY_NEW_SECRET:super-value;NEW_SECRET:SUPERSECRET_VALUE;MY_SECOND_SECRET:secret-value;MY_SECRET:SUPERSECRET;")),
		)
		if hashString := fmt.Sprintf("%x", hash); hashString != expected {
			t.Errorf("Expected hash to equal '%s', got '%s'", expected, hashString)
		}
	})

	t.Run("with a set of config maps and secrets", func(t *testing.T) {
		defer client.flush()
		client.registerConfig("global-config", "ENV", "prod")
		client.registerConfig("global-secret", "url", "my-url")
		client.registerSecret("my-secret", "SECRET_KEY", "super-value")

		data := []corev1.EnvVar{
			{
				Name: "ENV",
				ValueFrom: &corev1.EnvVarSource{
					ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: "global-config",
						},
						Key: "ENV",
					},
				},
			},
			{
				Name:  "MY_NEW_SECRET",
				Value: "SUPERSECRET_VALUE",
			},
			{
				Name: "MY_SECRET",
				ValueFrom: &corev1.EnvVarSource{
					SecretKeyRef: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: "my-secret",
						},
						Key: "SECRET_KEY",
					},
				},
			},
			{
				Name:  "MY_VALUE_SECRET",
				Value: "SUPERSECRET",
			},
		}

		hash, err := getEnvVarHash(client, "", data)
		if err != nil {
			t.Fatalf("Expected no error, got %s", err)
		}

		// concatenation of secrets requested above
		expected := fmt.Sprintf("%x", md5.Sum([]byte("ENV:prod;MY_NEW_SECRET:SUPERSECRET_VALUE;MY_SECRET:super-value;MY_VALUE_SECRET:SUPERSECRET;")))
		if hashString := fmt.Sprintf("%x", hash); hashString != expected {
			t.Errorf("Expected hash to equal '%s', got '%s'", expected, hashString)
		}
	})

	t.Run("with non existing secrets and configs", func(t *testing.T) {
		t.Run("without optional params", func(t *testing.T) {
			t.Run("config", func(t *testing.T) {
				defer client.flush()

				data := []corev1.EnvVar{
					{
						Name: "ENV",
						ValueFrom: &corev1.EnvVarSource{
							ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
								LocalObjectReference: corev1.LocalObjectReference{
									Name: "global-config",
								},
								Key: "ENV",
							},
						},
					},
				}
				_, err := getEnvVarHash(client, "", data)
				if !errors.IsNotFound(err) {
					t.Fatalf("Expected not found error, got %s", err)
				}
			})

			t.Run("secret", func(t *testing.T) {
				defer client.flush()

				data := []corev1.EnvVar{
					{
						Name: "API_KEY",
						ValueFrom: &corev1.EnvVarSource{
							SecretKeyRef: &corev1.SecretKeySelector{
								LocalObjectReference: corev1.LocalObjectReference{
									Name: "test-secrets",
								},
								Key: "API_KEY",
							},
						},
					},
				}

				_, err := getEnvVarHash(client, "", data)
				if !errors.IsNotFound(err) {
					t.Fatalf("Expected not found error, got %s", err)
				}
			})

			t.Run("no optional secret", func(t *testing.T) {
				defer client.flush()

				data := []corev1.EnvVar{
					{
						Name: "API_KEY",
						ValueFrom: &corev1.EnvVarSource{
							SecretKeyRef: &corev1.SecretKeySelector{
								Optional: k8sutils.PtrBool(false),
								LocalObjectReference: corev1.LocalObjectReference{
									Name: "test-secrets",
								},
								Key: "API_KEY",
							},
						},
					},
				}

				_, err := getEnvVarHash(client, "", data)
				if !errors.IsNotFound(err) {
					t.Fatalf("Expected not found error, got %s", err)
				}
			})

			t.Run("no optional config", func(t *testing.T) {
				defer client.flush()

				data := []corev1.EnvVar{
					{
						Name: "ENV",
						ValueFrom: &corev1.EnvVarSource{
							ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
								Optional: k8sutils.PtrBool(false),
								LocalObjectReference: corev1.LocalObjectReference{
									Name: "global-config",
								},
								Key: "ENV",
							},
						},
					},
				}
				_, err := getEnvVarHash(client, "", data)
				if !errors.IsNotFound(err) {
					t.Fatalf("Expected not found error, got %s", err)
				}
			})

		})

		t.Run("with optional params", func(t *testing.T) {
			fmt.Println(client.secretsData)
			t.Run("config", func(t *testing.T) {
				defer client.flush()

				data := []corev1.EnvVar{
					{
						Name: "ENV",
						ValueFrom: &corev1.EnvVarSource{
							ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
								Optional: k8sutils.PtrBool(true),
								LocalObjectReference: corev1.LocalObjectReference{
									Name: "global-config",
								},
								Key: "ENV",
							},
						},
					},
				}

				_, err := getEnvVarHash(client, "", data)
				if err != nil {
					t.Fatalf("Expected no error, got %s", err)
				}
			})

			t.Run("secret", func(t *testing.T) {
				defer client.flush()

				data := []corev1.EnvVar{
					{
						Name: "API_KEY",
						ValueFrom: &corev1.EnvVarSource{
							SecretKeyRef: &corev1.SecretKeySelector{
								Optional: k8sutils.PtrBool(true),
								LocalObjectReference: corev1.LocalObjectReference{
									Name: "test-secrets",
								},
								Key: "API_KEY",
							},
						},
					},
				}

				_, err := getEnvVarHash(client, "", data)
				if err != nil {
					t.Fatalf("Expected no error, got %s", err)
				}
			})
		})
	})

	t.Run("with non existing keys", func(t *testing.T) {
		client.registerConfig("global-config", "URL", "my-url")
		client.registerSecret("test-secrets", "TOKEN", "secret-token")
		defer client.flush()

		t.Run("without optional params", func(t *testing.T) {
			t.Run("config", func(t *testing.T) {
				data := []corev1.EnvVar{
					{
						Name: "ENV",
						ValueFrom: &corev1.EnvVarSource{
							ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
								LocalObjectReference: corev1.LocalObjectReference{
									Name: "global-config",
								},
								Key: "ENV",
							},
						},
					},
				}

				_, err := getEnvVarHash(client, "", data)
				if !errors.IsNotFound(err) {
					t.Fatalf("Expected not found error, got %s", err)
				}
			})

			t.Run("secret", func(t *testing.T) {
				data := []corev1.EnvVar{
					{
						Name: "API_KEY",
						ValueFrom: &corev1.EnvVarSource{
							SecretKeyRef: &corev1.SecretKeySelector{
								LocalObjectReference: corev1.LocalObjectReference{
									Name: "test-secrets",
								},
								Key: "API_KEY",
							},
						},
					},
				}

				_, err := getEnvVarHash(client, "", data)
				if !errors.IsNotFound(err) {
					t.Fatalf("Expected not found error, got %s", err)
				}
			})
		})

		t.Run("with optional params", func(t *testing.T) {
			t.Run("config", func(t *testing.T) {
				data := []corev1.EnvVar{
					{
						Name: "ENV",
						ValueFrom: &corev1.EnvVarSource{
							ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
								Optional: k8sutils.PtrBool(true),
								LocalObjectReference: corev1.LocalObjectReference{
									Name: "global-config",
								},
								Key: "ENV",
							},
						},
					},
				}

				_, err := getEnvVarHash(client, "", data)
				if err != nil {
					t.Fatalf("Expected no error, got %s", err)
				}
			})

			t.Run("secret", func(t *testing.T) {
				data := []corev1.EnvVar{
					{
						Name: "API_KEY",
						ValueFrom: &corev1.EnvVarSource{
							SecretKeyRef: &corev1.SecretKeySelector{
								Optional: k8sutils.PtrBool(true),
								LocalObjectReference: corev1.LocalObjectReference{
									Name: "test-secrets",
								},
								Key: "API_KEY",
							},
						},
					},
				}

				_, err := getEnvVarHash(client, "", data)
				if err != nil {
					t.Fatalf("Expected no error, got %s", err)
				}
			})
		})
	})
}

func TestHashed_EnvFromSource(t *testing.T) {
	client := &getter{}
	client.flush()

	t.Run("with a configmap reference", func(t *testing.T) {
		defer client.flush()

		client.registerConfig("global-config", "MY_KEY", "key-value")
		client.registerConfig("global-config", "ENV", "test")
		client.registerConfig("global-config", "BASE_URL", "http://test.co")
		expected := fmt.Sprintf("%x", md5.Sum([]byte("TEST_BASE_URL:http://test.co;TEST_ENV:test;TEST_MY_KEY:key-value;")))

		data := []corev1.EnvFromSource{
			{
				Prefix: "TEST_",
				ConfigMapRef: &corev1.ConfigMapEnvSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: "global-config",
					},
				},
			},
		}

		hash, err := getEnvFromSourceHash(client, "", data)
		if err != nil {
			t.Fatalf("Expected no error, got %s", err)
		}

		if hashString := fmt.Sprintf("%x", hash); hashString != expected {
			t.Errorf("Expected hash to equal '%s', got '%s'", expected, hashString)
		}
	})

	t.Run("with a secret reference", func(t *testing.T) {
		defer client.flush()

		client.registerSecret("global-secret", "TEST_KEY", "secret-value")
		client.registerSecret("global-secret", "API_KEY", "key-value")
		expected := fmt.Sprintf("%x", md5.Sum([]byte("API_KEY:key-value;TEST_KEY:secret-value;")))

		data := []corev1.EnvFromSource{
			{
				Prefix: "TEST_",
				SecretRef: &corev1.SecretEnvSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: "global-secret",
					},
				},
			},
		}

		hash, err := getEnvFromSourceHash(client, "", data)
		if err != nil {
			t.Fatalf("Expected no error, got %s", err)
		}

		if hashString := fmt.Sprintf("%x", hash); hashString != expected {
			t.Errorf("Expected hash to equal '%s', got '%s'", expected, hashString)
		}
	})

	t.Run("with a mix of secrets and references", func(t *testing.T) {
		defer client.flush()

		client.registerSecret("global-secret", "TEST_KEY", "secret-value")
		client.registerSecret("global-secret", "API_KEY", "key-value")
		client.registerSecret("secret", "ENCRYPTION", "kms:foo/bar")
		client.registerConfig("global-config", "MY_KEY", "key-value")
		client.registerConfig("global-config", "ENV", "test")
		client.registerConfig("global-config", "BASE_URL", "http://test.co")
		expected := fmt.Sprintf("%x", md5.Sum([]byte("API_KEY:key-value;TEST_KEY:secret-value;TEST_BASE_URL:http://test.co;TEST_ENV:test;TEST_MY_KEY:key-value;ENCRYPTION:kms:foo/bar;")))

		data := []corev1.EnvFromSource{
			{
				SecretRef: &corev1.SecretEnvSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: "global-secret",
					},
				},
			},
			{
				Prefix: "TEST_",
				ConfigMapRef: &corev1.ConfigMapEnvSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: "global-config",
					},
				},
			},
			{
				SecretRef: &corev1.SecretEnvSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: "secret",
					},
				},
			},
		}

		hash, err := getEnvFromSourceHash(client, "", data)
		if err != nil {
			t.Fatalf("Expected no error, got %s", err)
		}

		if hashString := fmt.Sprintf("%x", hash); hashString != expected {
			t.Errorf("Expected hash to equal '%s', got '%s'", expected, hashString)
		}
	})
}

type getter struct {
	secretsData map[string]map[string]string
	configData  map[string]map[string]string
}

func (g *getter) Get(obj interface{}, ns, name string) error {
	if secret, ok := obj.(*corev1.Secret); ok {
		return g.getSecret(secret, ns, name)
	} else if config, ok := obj.(*corev1.ConfigMap); ok {
		return g.getConfigMap(config, ns, name)
	}

	return errors.NewServiceUnavailable("Type not supported")
}

func (g *getter) getSecret(secret *corev1.Secret, ns, name string) error {
	data, ok := g.secretsData[name]
	if !ok {
		return errors.NewNotFound(schema.GroupResource{Group: "v1", Resource: "Secret"}, name)
	}

	binaryData := map[string][]byte{}
	for k, v := range data {
		binaryData[k] = []byte(v)
	}

	secret.StringData = data
	secret.Data = binaryData
	return nil
}

func (g *getter) getConfigMap(config *corev1.ConfigMap, ns, name string) error {
	data, ok := g.configData[name]
	if !ok {
		return errors.NewNotFound(schema.GroupResource{Group: "v1", Resource: "ConfigMap"}, name)
	}

	config.Data = data
	return nil
}

func (g *getter) registerSecret(secret, key, value string) {
	if _, ok := g.secretsData[secret]; !ok {
		g.secretsData[secret] = map[string]string{}
	}

	g.secretsData[secret][key] = value
}

func (g *getter) registerConfig(config, key, value string) {
	if _, ok := g.configData[config]; !ok {
		g.configData[config] = map[string]string{}
	}

	g.configData[config][key] = value
}

func (g *getter) flush() {
	g.secretsData = map[string]map[string]string{}
	g.configData = map[string]map[string]string{}
}
