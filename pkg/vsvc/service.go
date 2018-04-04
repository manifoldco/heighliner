package vsvc

import (
	"log"

	"github.com/manifoldco/heighliner/pkg/api/v1alpha1"
	"github.com/manifoldco/heighliner/pkg/k8sutils"

	"github.com/jelmersnoeck/kubekit"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// getService creates the Service Object for a VersionedMicroservice.
func getService(crd *v1alpha1.VersionedMicroservice) (runtime.Object, error) {
	network := crd.Spec.Network
	if network == nil {
		log.Printf("No network configured for %s, skipping service", crd.Name)
		return nil, nil
	}

	// if a DNS entry is specified but no ports are given, we should set up a
	// default port.
	if len(network.Ports) == 0 && network.DNS != nil {
		network.Ports = []v1alpha1.NetworkPort{
			{
				Name:       "headless",
				TargetPort: 8080,
				Port:       80,
			},
		}
	} else if len(network.Ports) == 0 {
		log.Printf("No ports or DNS configured for %s, skipping service", crd.Name)
		return nil, nil
	}

	labels := k8sutils.Labels(crd.Labels, crd.ObjectMeta)
	annotations := k8sutils.Annotations(crd.Annotations, v1alpha1.Version, crd)

	ports, err := getServicePorts(network.Ports)
	if err != nil {
		return nil, err
	}

	svc := &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Service",
			APIVersion: "v1",
		},
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
			SessionAffinity: network.SessionAffinity,
		},
	}

	return svc, nil
}

func getServicePorts(networkPorts []v1alpha1.NetworkPort) ([]corev1.ServicePort, error) {
	ports := make([]corev1.ServicePort, len(networkPorts))

	for i, port := range networkPorts {
		ports[i] = corev1.ServicePort{
			Protocol:   corev1.ProtocolTCP,
			Name:       port.Name,
			Port:       port.Port,
			TargetPort: intstr.FromInt(int(port.TargetPort)),
		}
	}

	return ports, nil
}
