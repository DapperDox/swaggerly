package discover

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"time"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kenjones-cisco/dapperdox/discover/models"
)

const (
	defaultVal = "default"
)

func copyFile(src, dst string) {
	// Read all content of src to data
	data, _ := ioutil.ReadFile(src)
	// Write data to dst
	_ = ioutil.WriteFile(dst, data, 0644)
}

func copyKubeToken() {
	_ = os.MkdirAll("/var/run/secrets/kubernetes.io/serviceaccount", 0755)

	copyFile("testdata/token", "/var/run/secrets/kubernetes.io/serviceaccount/token")
}

func copyKubeCert() {
	_ = os.MkdirAll("/var/run/secrets/kubernetes.io/serviceaccount", 0755)

	copyFile("testdata/ca.crt", "/var/run/secrets/kubernetes.io/serviceaccount/ca.crt")
}

func genServerAPI(path string) *httptest.Server {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		spec, err := ioutil.ReadFile(path)
		if err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
		} else {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)

			_, _ = fmt.Fprintln(w, string(spec))
		}
	}))

	return srv
}

var testServices = []*models.Service{
	{
		Hostname: "abc.default.svc.cluster.local",
		Ports:    []*models.Port{{Name: "http", Port: 80, Protocol: models.ProtocolHTTP}},
	},
	{
		Hostname: "bob.default.svc.cluster.local",
		Address:  "10.0.1.1",
		Ports:    []*models.Port{{Name: "https", Port: 443, Protocol: models.ProtocolHTTPS}},
	},
	{
		Hostname: "cat.default.svc.cluster.local",
		Ports:    []*models.Port{{Name: "http", Port: 80, Protocol: models.ProtocolHTTP}},
	},
	{
		Hostname: "dog.default.svc.cluster.local",
		Address:  "10.0.1.2",
		Ports:    []*models.Port{{Name: "http", Port: 80, Protocol: models.ProtocolHTTP}, {Name: "grpc", Port: 90, Protocol: models.ProtocolGRPC}},
	},
	{
		Hostname: "foo.default.svc.cluster.local",
		Ports:    []*models.Port{{Name: "https", Port: 443, Protocol: models.ProtocolHTTPS}},
	},
}

var testIgnoredServices = []*models.Service{
	{
		Hostname: "apigw",
		Ports:    []*models.Port{{Name: "http", Port: 80, Protocol: models.ProtocolHTTP}},
	},
	{
		Hostname: "auth-local",
		Ports:    []*models.Port{{Name: "http", Port: 80, Protocol: models.ProtocolHTTP}},
	},
	{
		Hostname: "discovery",
		Ports:    []*models.Port{{Name: "http", Port: 80, Protocol: models.ProtocolHTTP}},
	},
	{
		Hostname: "docs",
		Ports:    []*models.Port{{Name: "http", Port: 80, Protocol: models.ProtocolHTTP}},
	},
	{
		Hostname: "redis",
		Ports:    []*models.Port{{Name: "http", Port: 80, Protocol: models.ProtocolHTTP}},
	},
	{
		Hostname: "ui",
		Ports:    []*models.Port{{Name: "http", Port: 80, Protocol: models.ProtocolHTTP}},
	},
}

var (
	sm                = models.NewServiceMap(testServices...)
	testServiceMap    = &sm
	sm2               = models.NewServiceMap()
	emptyServiceMap   = &sm2
	sm3               = models.NewServiceMap(testIgnoredServices...)
	ignoredServiceMap = &sm3
)

var testDeployments = []*models.Deployment{
	{
		Name:              "abc-svc",
		Namespace:         "mcmp-rtp",
		CreationTimestamp: v1.NewTime(time.Now()),
		Version:           "1",
	},
	{
		Name:              "bob-svc",
		Namespace:         "mcmp-rtp",
		CreationTimestamp: v1.NewTime(time.Now()),
		Version:           "13",
	},
	{
		Name:              "cat-svc",
		Namespace:         "mcmp-rtp",
		CreationTimestamp: v1.NewTime(time.Now()),
		Version:           "7",
	},
	{
		Name:              "dog-svc",
		Namespace:         "mcmp-rtp",
		CreationTimestamp: v1.NewTime(time.Now()),
		Version:           "4",
	},
	{
		Name:              "foo-svc",
		Namespace:         "mcmp-rtp",
		CreationTimestamp: v1.NewTime(time.Now()),
		Version:           "9",
	},
}

var (
	dm                 = models.NewDeploymentMap()
	emptyDeploymentMap = &dm
)

type fakeController struct {
	wantErr bool
	wantNil bool
}

func (c *fakeController) AppendServiceHandler(f func(*models.Service, models.Event)) error {
	return nil
}

func (c *fakeController) AppendDeploymentHandler(f func(*models.Deployment, models.Event)) error {
	return nil
}

func (c *fakeController) Run(stop <-chan struct{}) {
	<-stop
}
