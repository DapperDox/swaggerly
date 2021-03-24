// Copyright 2017 Istio Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package discover

import (
	"os"
	"testing"
	"time"

	v1 "k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/kenjones-cisco/dapperdox/discover/models"
)

const (
	resync = 1 * time.Second
)

func Test_NewClient(t *testing.T) {
	if _, err := newClient(); err == nil {
		t.Errorf("newClient() error = %v, wantErr %v", err, true)
	}

	envVars := []string{"KUBERNETES_SERVICE_HOST", "KUBERNETES_SERVICE_PORT"}
	existingEnv := make(map[string]string)

	for _, k := range envVars {
		existingEnv[k] = os.Getenv(k)
	}

	defer func() {
		for k, v := range existingEnv {
			_ = os.Setenv(k, v)
		}
	}()

	_ = os.Setenv("KUBERNETES_SERVICE_HOST", "localhost")
	_ = os.Setenv("KUBERNETES_SERVICE_PORT", "8080")

	copyKubeToken()
	copyKubeCert()

	d, err := newClient()
	if err != nil {
		t.Errorf("newClient() error = %v, wantErr %v", err, true)
	}

	if d == nil {
		t.Errorf("newClient() = %v, want not nil", d)
	}
}

func setupServices(t *testing.T, catalog *catalog, namespace string) {
	t.Helper()

	createService(t, catalog, "svc1", namespace, nil, []int32{8080}, map[string]string{"app": "prod-app"})
	createService(t, catalog, "svc2", namespace, nil, []int32{8081}, map[string]string{"app": "staging-app"})
}

func TestController_Services(t *testing.T) {
	stop := make(chan struct{})
	defer close(stop)

	namespace := "nsA"
	catalog := makeFakeKubeAPIController(namespace)

	go catalog.Run(stop)

	setupServices(t, catalog, namespace)
}

func makeFakeKubeAPIController(namespace string) *catalog {
	if namespace == "" {
		namespace = defaultVal
	}

	clientSet := fake.NewSimpleClientset()
	ctlg := newCatalog(clientSet, catalogOptions{
		WatchedNamespace: namespace,
		ResyncPeriod:     resync,
		DomainSuffix:     domainSuffix,
	})
	catalog, _ := ctlg.(*catalog)

	_ = catalog.AppendServiceHandler(updateServicesHandler)
	_ = catalog.AppendDeploymentHandler(updateDeploymentHandler)

	return catalog
}

func updateServicesHandler(*models.Service, models.Event)      {}
func updateDeploymentHandler(*models.Deployment, models.Event) {}

func createService(t *testing.T, catalog *catalog, name, namespace string, annotations map[string]string, ports []int32, selector map[string]string) {
	t.Helper()

	svcPorts := []v1.ServicePort{}

	for _, p := range ports {
		svcPorts = append(svcPorts, v1.ServicePort{
			Name:     "test-port",
			Port:     p,
			Protocol: "http",
		})
	}

	service := &v1.Service{
		ObjectMeta: meta_v1.ObjectMeta{
			Name:        name,
			Namespace:   namespace,
			Annotations: annotations,
		},
		Spec: v1.ServiceSpec{
			ClusterIP: "10.0.0.1",
			Ports:     svcPorts,
			Selector:  selector,
			Type:      v1.ServiceTypeClusterIP,
		},
	}

	if err := catalog.services.informer.GetStore().Add(service); err != nil {
		t.Errorf("Cannot create service %s in namespace %s (error: %v)", name, namespace, err)
	}
}
