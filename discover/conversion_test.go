package discover

import (
	"testing"
	"time"

	"k8s.io/api/apps/v1beta1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kenjones-cisco/dapperdox/discover/models"
)

var (
	domainSuffix = "company.com"

	protocols = []struct {
		name  string
		proto v1.Protocol
		out   models.Protocol
	}{
		{"", v1.ProtocolTCP, models.ProtocolTCP},
		{"http", v1.ProtocolTCP, models.ProtocolHTTP},
		{"http-test", v1.ProtocolTCP, models.ProtocolHTTP},
		{"http", v1.ProtocolUDP, models.ProtocolUDP},
		{"httptest", v1.ProtocolTCP, models.ProtocolTCP},
		{"https", v1.ProtocolTCP, models.ProtocolHTTPS},
		{"https-test", v1.ProtocolTCP, models.ProtocolHTTPS},
		{"http2", v1.ProtocolTCP, models.ProtocolHTTP2},
		{"http2-test", v1.ProtocolTCP, models.ProtocolHTTP2},
		{"grpc", v1.ProtocolTCP, models.ProtocolGRPC},
		{"grpc-test", v1.ProtocolTCP, models.ProtocolGRPC},
		{"mongo", v1.ProtocolTCP, models.ProtocolMongo},
		{"mongo-test", v1.ProtocolTCP, models.ProtocolMongo},
		{"redis", v1.ProtocolTCP, models.ProtocolRedis},
		{"redis-test", v1.ProtocolTCP, models.ProtocolRedis},
	}
)

func Test_convertProtocol(t *testing.T) {
	for _, tt := range protocols {
		out := convertProtocol(tt.name, tt.proto)
		if out != tt.out {
			t.Errorf("convertProtocol(%q, %q) => %q, want %q", tt.name, tt.proto, out, tt.out)
		}
	}
}

func TestServiceConversion(t *testing.T) {
	serviceName := "service1"
	namespace := defaultVal

	localSvc := v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      serviceName,
			Namespace: namespace,
			Annotations: map[string]string{
				"other/annotation": "test",
			},
		},
		Spec: v1.ServiceSpec{
			ClusterIP: "10.0.0.1",
			Ports: []v1.ServicePort{
				{
					Name:     "http",
					Port:     8080,
					Protocol: v1.ProtocolTCP,
				},
				{
					Name:     "https",
					Protocol: v1.ProtocolTCP,
					Port:     443,
				},
			},
		},
	}

	service := convertService(&localSvc, domainSuffix)
	if service == nil {
		t.Errorf("could not convert service")
	}

	if service != nil && len(service.Ports) != len(localSvc.Spec.Ports) {
		t.Errorf("incorrect number of ports => %v, want %v",
			len(service.Ports), len(localSvc.Spec.Ports))
	}

	if service.External() {
		t.Error("service should not be external")
	}

	if service.Hostname != serviceHostname(serviceName, namespace, domainSuffix) {
		t.Errorf("service hostname incorrect => %q, want %q",
			service.Hostname, serviceHostname(serviceName, namespace, domainSuffix))
	}

	if service.Address != localSvc.Spec.ClusterIP {
		t.Errorf("service IP incorrect => %q, want %q", service.Address, localSvc.Spec.ClusterIP)
	}
}

func TestExternalServiceConversion(t *testing.T) {
	serviceName := "service1"
	namespace := defaultVal

	extSvc := v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      serviceName,
			Namespace: namespace,
		},
		Spec: v1.ServiceSpec{
			Ports: []v1.ServicePort{
				{
					Name:     "http",
					Port:     80,
					Protocol: v1.ProtocolTCP,
				},
			},
			Type:         v1.ServiceTypeExternalName,
			ExternalName: "google.com",
		},
	}

	service := convertService(&extSvc, domainSuffix)
	if service == nil {
		t.Errorf("could not convert external service")
	}

	if service != nil && len(service.Ports) != len(extSvc.Spec.Ports) {
		t.Errorf("incorrect number of ports => %v, want %v",
			len(service.Ports), len(extSvc.Spec.Ports))
	}

	if service.ExternalName != extSvc.Spec.ExternalName || !service.External() {
		t.Error("service should be external")
	}

	if service.Hostname != serviceHostname(serviceName, namespace, domainSuffix) {
		t.Errorf("service hostname incorrect => %q, want %q",
			service.Hostname, extSvc.Spec.ExternalName)
	}
}

func Test_convertDeployment(t *testing.T) {
	ttime := metav1.NewTime(time.Now())

	type args struct {
		dpl *v1beta1.Deployment
	}

	tests := []struct {
		name string
		args args
		want *models.Deployment
	}{
		{
			name: "success - populated k8s Deployment converts to Deployment",
			args: args{
				dpl: &v1beta1.Deployment{},
			},
			want: &models.Deployment{},
		},
		{
			name: "success - empty k8s Deployment converts to empty Deployment",
			args: args{
				dpl: &v1beta1.Deployment{
					ObjectMeta: metav1.ObjectMeta{
						Name:              "abc-svc",
						Namespace:         "mcmp-rtp",
						CreationTimestamp: ttime,
						Annotations: map[string]string{
							revKeyRef: "3",
						},
					},
				},
			},
			want: &models.Deployment{
				Name:              "abc-svc",
				Namespace:         "mcmp-rtp",
				CreationTimestamp: ttime,
				Version:           "3",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := convertDeployment(tt.args.dpl)
			if *got != *tt.want {
				t.Errorf("convertDeployment() returned %v, expected %v", *got, *tt.want)
			}
		})
	}
}
