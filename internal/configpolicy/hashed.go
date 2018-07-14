package configpolicy

import (
	"crypto/md5"
	"fmt"
	"io"
	"sort"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
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
			byteValue, err := valueFromEnvVar(p, namespace, env)
			if err != nil {
				return nil, err
			}
			value = string(byteValue)
		}

		io.WriteString(h, fmt.Sprintf("%s:%s;", env.Name, value))
	}

	return h.Sum(nil), nil
}

func getEnvFromSourceHash(p objectGetter, namespace string, envs []corev1.EnvFromSource) ([]byte, error) {
	h := md5.New()

	// the array of references is already sorted, no need to rearrange these
	for _, env := range envs {
		values, err := valuesFromSource(p, namespace, env)
		if err != nil {
			return nil, err
		}

		// the data for the source values is a map. Ranging over maps is random
		// so we need to sort it to make sure we always have the same end
		// result.
		sortedKeys := sort.StringSlice{}
		for k := range values {
			sortedKeys = append(sortedKeys, k)
		}
		sortedKeys.Sort()

		for _, mapKey := range sortedKeys {
			io.WriteString(h, fmt.Sprintf("%s:%s;", mapKey, string(values[mapKey])))
		}
	}

	return h.Sum(nil), nil
}

func valuesFromSource(p objectGetter, ns string, env corev1.EnvFromSource) (map[string][]byte, error) {
	data := map[string][]byte{}
	switch {
	case env.ConfigMapRef != nil:
		cmr := env.ConfigMapRef
		config := &corev1.ConfigMap{
			TypeMeta: metav1.TypeMeta{
				Kind:       "ConfigMap",
				APIVersion: "v1",
			},
		}

		if err := p.Get(config, ns, cmr.Name); err != nil {
			return nil, err
		}

		for k, v := range config.Data {
			data[fmt.Sprintf("%s%s", env.Prefix, k)] = []byte(v)
		}
	case env.SecretRef != nil:
		sr := env.SecretRef

		secret := &corev1.Secret{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Secret",
				APIVersion: "v1",
			},
		}

		if err := p.Get(secret, ns, sr.Name); err != nil {
			return nil, err
		}

		data = secret.Data
	}

	return data, nil
}

func valueFromEnvVar(p objectGetter, ns string, env corev1.EnvVar) ([]byte, error) {
	if env.ValueFrom == nil {
		return nil, nil
	}
	vf := env.ValueFrom

	var data []byte
	switch {
	case vf.ConfigMapKeyRef != nil:
		ckr := vf.ConfigMapKeyRef
		config := &corev1.ConfigMap{
			TypeMeta: metav1.TypeMeta{
				Kind:       "ConfigMap",
				APIVersion: "v1",
			},
		}

		if err := p.Get(config, ns, ckr.Name); err != nil {
			if errors.IsNotFound(err) && ckr.Optional != nil && *ckr.Optional {
				return nil, nil
			}

			return nil, err
		}

		rawData, ok := config.Data[ckr.Key]
		if !ok && isRequired(ckr.Optional) {
			return nil, errors.NewNotFound(
				schema.GroupResource{Group: "v1", Resource: "ConfigMap"},
				fmt.Sprintf("%s: %s", ckr.Name, ckr.Key),
			)
		}

		// TODO(jelmer): in 1.10 there's `BinaryData`
		data = []byte(rawData)
	case vf.SecretKeyRef != nil:
		skr := vf.SecretKeyRef
		secret := &corev1.Secret{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Secret",
				APIVersion: "v1",
			},
		}

		if err := p.Get(secret, ns, skr.Name); err != nil {
			if errors.IsNotFound(err) && skr.Optional != nil && *skr.Optional {
				return nil, nil
			}

			return nil, err
		}

		rawData, ok := secret.Data[skr.Key]
		if !ok && isRequired(skr.Optional) {
			return nil, errors.NewNotFound(
				schema.GroupResource{Group: "v1", Resource: "Secret"},
				fmt.Sprintf("%s: %s", skr.Name, skr.Key),
			)
		}

		data = rawData
	}

	return data, nil
}

func isRequired(optional *bool) bool {
	return optional == nil || (optional != nil && !*optional)
}
