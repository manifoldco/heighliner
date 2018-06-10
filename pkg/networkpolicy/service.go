package networkpolicy

import (
	"github.com/manifoldco/heighliner/pkg/api/v1alpha1"
	"github.com/manifoldco/heighliner/pkg/k8sutils"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func buildServiceForRelease(srv *corev1.Service, svc *v1alpha1.Microservice, np *v1alpha1.NetworkPolicy, release *v1alpha1.Release) *corev1.Service {
	labels := k8sutils.Labels(np.Labels, np.ObjectMeta)
	labels["hlnr.io/microservice.full_name"] = release.FullName(svc.Name)
	labels["hlnr.io/microservice.name"] = svc.Name
	labels["hlnr.io/microservice.release"] = release.Name()
	labels["hlnr.io/microservice.version"] = release.Version()

	if srv.Labels == nil {
		srv.Labels = labels
	} else {
		// maintain any existing labels, adding our new ones
		for k, v := range labels {
			srv.Labels[k] = v
		}
	}

	annotations := k8sutils.Annotations(np.Annotations, v1alpha1.Version, np)
	if srv.Annotations == nil {
		srv.Annotations = annotations
	} else {
		// maintain any existing annotations, adding our new ones
		for k, v := range annotations {
			srv.Annotations[k] = v
		}
	}

	selector := make(map[string]string)
	for k, v := range labels {
		selector[k] = v
	}
	delete(selector, k8sutils.LabelServiceKey)

	sessionAffinity := corev1.ServiceAffinityNone
	if np.Spec.SessionAffinity != nil && np.Spec.SessionAffinity.ClientIP != nil {
		sessionAffinity = corev1.ServiceAffinityClientIP
	}

	srv.OwnerReferences = release.OwnerReferences
	srv.Spec.Type = corev1.ServiceTypeNodePort
	srv.Spec.Ports = getServicePorts(np.Spec.Ports)
	srv.Spec.Selector = selector
	srv.Spec.SessionAffinity = sessionAffinity
	srv.Spec.SessionAffinityConfig = np.Spec.SessionAffinity

	return srv
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
