package githubpolicy

import (
	"testing"

	"k8s.io/api/core/v1"
)

func TestGetSecretAuthToken(t *testing.T) {
	cl := &dummyClient{}

	t.Run("without valid key", func(t *testing.T) {
		cl.getFunc = func(obj interface{}, ns, name string) error {
			secret := obj.(*v1.Secret)
			secret.StringData = map[string]string{
				"WRONG_KEY": "",
			}
			return nil
		}

		_, err := getSecretAuthToken(cl, "test", "test-secret")
		if err == nil {
			t.Errorf("Expected an error, got none")
		}
	})

	t.Run("with valid key", func(t *testing.T) {
		cl.getFunc = func(obj interface{}, ns, name string) error {
			secret := obj.(*v1.Secret)
			secret.StringData = map[string]string{
				"GITHUB_AUTH_TOKEN": "dXB0b3duZnVuYw==",
			}
			return nil
		}

		token, err := getSecretAuthToken(cl, "test", "test-secret")
		if err != nil {
			t.Errorf("Expected no error, got '%s'", err)
		}

		if expected := "uptownfunc"; string(token) != expected {
			t.Errorf("Expected token to equal '%s', got '%s'", expected, string(token))
		}
	})
}

type dummyClient struct {
	getFunc func(interface{}, string, string) error
}

func (c *dummyClient) Get(obj interface{}, ns string, name string) error {
	return c.getFunc(obj, ns, name)
}
