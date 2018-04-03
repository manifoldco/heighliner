package configpolicy

import (
	"crypto/md5"
	"io"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type objectGetter interface {
	Get(interface{}, string, string) error
}

// TODO(jelmer): stronger encryption? The value will be stored on the CRD status
// so it could be seen by others.
func getEnvVarHash(p objectGetter, namespace string, envs []corev1.EnvVar) ([]byte, error) {
	h := md5.New()

	// envs are an array of vars. This means looping over them is always in the
	// same order. We don't need to do any sort of special sorting on this.
	for _, env := range envs {
		value := env.Value
		if env.ValueFrom != nil {
			byteValue, err := valueFromSource(p, namespace, env)
			if err != nil {
				return nil, err
			}
			value = string(byteValue)
		}

		io.WriteString(h, value)
	}

	return h.Sum(nil), nil
}

func getEnvFromSourceHash(p objectGetter, namespace string, envs []corev1.EnvFromSource) ([]byte, error) {
	return nil, nil
}

func valueFromSource(p objectGetter, ns string, env corev1.EnvVar) ([]byte, error) {
	if env.ValueFrom == nil {
		return nil, nil
	}
	vf := env.ValueFrom

	var data []byte
	if vf.ConfigMapKeyRef != nil {
		ckr := vf.ConfigMapKeyRef
		config := &corev1.ConfigMap{
			TypeMeta: metav1.TypeMeta{
				Kind:       "ConfigMap",
				APIVersion: "v1",
			},
		}

		// TODO(jelmer): handle the case where the secret doesn't exist and it's
		// defined as optional.
		if err := p.Get(config, ns, ckr.Name); err != nil {
			return nil, err
		}

		// TODO(jelmer): in 1.10 there's `BinaryData`
		data = []byte(config.Data[ckr.Key])
	} else if vf.SecretKeyRef != nil {
		skr := vf.SecretKeyRef
		secret := &corev1.Secret{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Secret",
				APIVersion: "v1",
			},
		}

		// TODO(jelmer): handle the case where the secret doesn't exist and it's
		// defined as optional.
		if err := p.Get(secret, ns, skr.Name); err != nil {
			return nil, err
		}

		data = secret.Data[skr.Key]
	}

	return data, nil
}
