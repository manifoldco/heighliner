package meta

import (
	"regexp"
	"unicode/utf8"

	"github.com/jelmersnoeck/kubekit"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/manifoldco/heighliner/pkg/api/v1alpha1"
)

// Annotations returns a set of annotations annotated with the Heighliner
// defaults.
func Annotations(ann map[string]string, version string, resource runtime.Object) map[string]string {
	if ann == nil {
		ann = map[string]string{}
	}

	ann["hlnr.io/version"] = version
	ann["hlnr.io/component"] = kubekit.TypeName(resource)
	return ann
}

// Labels returns a new set of labels annotated with Heighliner specific
// defaults.
func Labels(labels map[string]string, m metav1.Object) map[string]string {
	if labels == nil {
		labels = map[string]string{}
	}

	labels[LabelServiceKey] = labelize(m.GetName())
	return labels
}

// MicroserviceLabels returns a new set of labels annotated with Heighliner specific
// defaults (as from Label), and release specific values.
func MicroserviceLabels(ms *v1alpha1.Microservice, r *v1alpha1.Release, parent metav1.Object) map[string]string {
	labels := Labels(parent.GetLabels(), parent)

	labels["hlnr.io/microservice.full_name"] = labelize(r.FullName(ms.Name))
	labels["hlnr.io/microservice.name"] = labelize(ms.Name)
	labels["hlnr.io/microservice.release"] = labelize(r.Name())
	labels["hlnr.io/microservice.version"] = labelize(r.Version())

	return labels
}

// trim a string to at most len runes from a utf8 byte sequence.
func trim(s string, l int) string {
	var ns string
	var i int
	for _, c := range s {
		i++
		ns += string(c)
		if i >= l {
			break
		}
	}

	return ns
}

// elide trims a string to l chars, with '...0' in the end, included in l.
func elide(s string, l int) string {
	if l <= 4 {
		return trim(s, l)
	}

	n := utf8.RuneCount([]byte(s))
	if n <= l {
		return s
	}

	return trim(s, l-4) + "...0"
}

var unlabelChars = regexp.MustCompile(`([^a-zA-Z0-9._-])`)
var unlabelStart = regexp.MustCompile(`^([^a-zA-Z0-9])`)
var unlabelEnd = regexp.MustCompile(`([^a-zA-Z0-9])$`)

// labelize coerces a string into a format valid for k8s label values, as
// outlined here: https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/#syntax-and-character-set
//
// Our rules:
//   - If the value starts or ends with a non-alphanumeric value, 0 is
//     prepended/appended.
//   - characters that are not within [a-zA-Z0-9._-] are replaced with _.
//   - values greater than 63 characters are elided, with a trailing 0.
//
// Values passed through labelize for normalization should be for
// informational/debugging purposes only. If you rely on the label for
// something, make sure you normalize it yourself.
//
// Note that Name fields are up to 253 chars long, and so should be passed
// through labelize as well.
func labelize(s string) string {
	s = unlabelStart.ReplaceAllString(s, "0${1}")
	s = unlabelEnd.ReplaceAllString(s, "${1}0")
	s = unlabelChars.ReplaceAllString(s, "_")
	return elide(s, 63)
}
