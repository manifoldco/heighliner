package networkpolicy

import (
	"github.com/manifoldco/heighliner/pkg/api/v1alpha1"
	"github.com/manifoldco/heighliner/pkg/k8sutils"

	"github.com/jelmersnoeck/kubekit"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func buildServiceForRelease(svc *v1alpha1.Microservice, np *v1alpha1.NetworkPolicy, release *v1alpha1.Release, versioned bool) (*corev1.Service, error) {
	if len(np.Spec.Ports) == 0 {
		return nil, nil
	}

	name := svc.Name
	if versioned {
		name = release.FullName(svc.Name)
	}

	labels := k8sutils.Labels(np.Labels, np.ObjectMeta)
	labels["hlnr.io/microservice.full_name"] = release.FullName(svc.Name)
	labels["hlnr.io/microservice.name"] = svc.Name
	labels["hlnr.io/microservice.release"] = release.Name()
	labels["hlnr.io/microservice.version"] = release.Version()

	selector := labels
	delete(selector, k8sutils.LabelServiceKey)

	annotations := k8sutils.Annotations(np.Annotations, v1alpha1.Version, np)

	sessionAffinity := corev1.ServiceAffinityNone
	if np.Spec.SessionAffinity != nil && np.Spec.SessionAffinity.ClientIP != nil {
		sessionAffinity = corev1.ServiceAffinityClientIP
	}

	return &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Service",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			// TODO(jelmer): we'll want a hashed name here based on timestamp
			// etc.
			Name:        name,
			Namespace:   svc.Namespace,
			Labels:      labels,
			Annotations: annotations,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(
					svc,
					v1alpha1.SchemeGroupVersion.WithKind(kubekit.TypeName(svc)),
				),
			},
		},
		Spec: corev1.ServiceSpec{
			Type:                  corev1.ServiceTypeNodePort,
			Ports:                 getServicePorts(np.Spec.Ports),
			Selector:              selector,
			SessionAffinity:       sessionAffinity,
			SessionAffinityConfig: np.Spec.SessionAffinity,
		},
	}, nil
}

func getServicePorts(networkPorts []v1alpha1.NetworkPort) []corev1.ServicePort {
	ports := make([]corev1.ServicePort, len(networkPorts))

	for i, port := range networkPorts {
		ports[i] = corev1.ServicePort{
			Protocol:   corev1.ProtocolTCP,
			Name:       port.Name,
			Port:       port.Port,
			TargetPort: intstr.FromInt(int(port.TargetPort)),
		}
	}

	return ports
}
