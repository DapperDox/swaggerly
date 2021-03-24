package discover

import (
	"fmt"
	"strings"

	"k8s.io/api/apps/v1beta1"
	v1 "k8s.io/api/core/v1"

	"github.com/kenjones-cisco/dapperdox/discover/models"
)

const (
	revKeyRef = "deployment.kubernetes.io/revision"
)

func convertPort(port *v1.ServicePort) *models.Port {
	return &models.Port{
		Name:     port.Name,
		Port:     int(port.Port),
		Protocol: convertProtocol(port.Name, port.Protocol),
	}
}

func convertService(svc *v1.Service, domainSuffix string) *models.Service {
	addr, external := "", ""
	if svc.Spec.ClusterIP != "" && svc.Spec.ClusterIP != v1.ClusterIPNone {
		addr = svc.Spec.ClusterIP
	}

	if svc.Spec.Type == v1.ServiceTypeExternalName && svc.Spec.ExternalName != "" {
		external = svc.Spec.ExternalName
	}

	ports := make([]*models.Port, 0, len(svc.Spec.Ports))
	for i := range svc.Spec.Ports {
		ports = append(ports, convertPort(&svc.Spec.Ports[i]))
	}

	loadBalancingDisabled := addr == "" && external == "" // headless services should not be load balanced

	return &models.Service{
		Hostname:              serviceHostname(svc.Name, svc.Namespace, domainSuffix),
		Ports:                 ports,
		Address:               addr,
		ExternalName:          external,
		LoadBalancingDisabled: loadBalancingDisabled,
	}
}

func convertDeployment(dpl *v1beta1.Deployment) *models.Deployment {
	return &models.Deployment{
		Name:              dpl.Name,
		Namespace:         dpl.Namespace,
		CreationTimestamp: dpl.CreationTimestamp,
		Version:           dpl.Annotations[revKeyRef],
	}
}

// serviceHostname produces FQDN for a k8s service.
func serviceHostname(name, namespace, domainSuffix string) string {
	return fmt.Sprintf("%s.%s.svc.%s", name, namespace, domainSuffix)
}

// convertProtocol from k8s protocol and port name.
func convertProtocol(name string, proto v1.Protocol) models.Protocol {
	out := models.ProtocolTCP

	switch proto {
	case v1.ProtocolUDP:
		out = models.ProtocolUDP
	case v1.ProtocolTCP:
		prefix := name

		i := strings.Index(name, "-")
		if i >= 0 {
			prefix = name[:i]
		}

		protocol := models.ConvertCaseInsensitiveStringToProtocol(prefix)

		if protocol != models.ProtocolUDP && protocol != models.ProtocolUnsupported {
			out = protocol
		}
	}

	return out
}
