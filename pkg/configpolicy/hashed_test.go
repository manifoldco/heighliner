package configpolicy

import (
	"crypto/md5"
	"errors"
	"fmt"
	"testing"

	corev1 "k8s.io/api/core/v1"
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
		expected := fmt.Sprintf("%x", md5.Sum([]byte("SUPERSECRET_VALUESUPERSECRET")))
		if hashString := fmt.Sprintf("%x", hash); hashString != expected {
			t.Errorf("Expected hash to equal '%s', got '%s'", expected, hashString)
		}
	})

	t.Run("with a set of secret references", func(t *testing.T) {
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

		// concatenation of secrets requested above
		expected := fmt.Sprintf("%x", md5.Sum([]byte("super-valuesecret-value")))
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
				Name:  "MY_NEW_SECRET",
				Value: "SUPERSECRET_VALUE",
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
		expected := fmt.Sprintf("%x", md5.Sum([]byte("super-valueSUPERSECRET_VALUEsecret-valueSUPERSECRET")))
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
				Name:  "MY_SECRET",
				Value: "SUPERSECRET",
			},
		}

		hash, err := getEnvVarHash(client, "", data)
		if err != nil {
			t.Fatalf("Expected no error, got %s", err)
		}

		// concatenation of secrets requested above
		expected := fmt.Sprintf("%x", md5.Sum([]byte("prodSUPERSECRET_VALUEsuper-valueSUPERSECRET")))
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

	return errors.New("Type not supported")
}

func (g *getter) getSecret(secret *corev1.Secret, ns, name string) error {
	data, ok := g.secretsData[name]
	if !ok {
		return errors.New("Not found")
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
		return errors.New("Not found")
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
