package vsvc

import (
	"log"

	"github.com/jelmersnoeck/kubekit"
	"github.com/manifoldco/heighliner/pkg/api/v1alpha1"
	"github.com/manifoldco/heighliner/pkg/k8sutils"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// getService creates the Service Object for a VersionedMicroservice.
func getService(crd *v1alpha1.VersionedMicroservice) (*corev1.Service, error) {
	if crd.Spec.Network == nil {
		log.Printf("Skipping Service creation for %s", crd.Name)
	}

	labels := crd.Labels
	labels[k8sutils.LabelServiceKey] = crd.Name
	annotations := k8sutils.Annotations(crd.Annotations, v1alpha1.Version, crd)

	ports, err := getServicePorts(crd.Spec.Network.Ports)
	if err != nil {
		return nil, err
	}

	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:        crd.Name,
			Namespace:   crd.Namespace,
			Labels:      labels,
			Annotations: annotations,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(
					crd,
					v1alpha1.SchemeGroupVersion.WithKind(kubekit.TypeName(crd)),
				),
			},
		},
		Spec: corev1.ServiceSpec{
			Type:  corev1.ServiceTypeNodePort,
			Ports: ports,
			Selector: map[string]string{
				k8sutils.LabelServiceKey: crd.Name,
			},
			// TODO(jelmer): make this configurable
			SessionAffinity: corev1.ServiceAffinityNone,
		},
	}

	return svc, nil
}

func getServicePorts(networkPorts []v1alpha1.NetworkPort) ([]corev1.ServicePort, error) {
	ports := make([]corev1.ServicePort, len(networkPorts))

	for i, port := range networkPorts {
		// TODO(jelmer): add validation, should potentially be done through
		// an actua CRD validator.
		ports[i] = corev1.ServicePort{
			Protocol:   corev1.ProtocolTCP,
			Name:       port.Name,
			Port:       port.Port,
			TargetPort: intstr.FromInt(int(port.TargetPort)),
		}
	}

	return ports, nil
}
