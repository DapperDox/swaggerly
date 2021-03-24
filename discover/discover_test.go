package discover

import (
	"os"
	"sync"
	"testing"
	"time"

	"github.com/kenjones-cisco/dapperdox/discover/models"
)

func TestNewDiscoverer(t *testing.T) {
	if _, err := NewDiscoverer(); err == nil {
		t.Errorf("NewDiscoverer() error = %v, wantErr %v", err, true)
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

	d, err := NewDiscoverer()
	if err != nil {
		t.Errorf("NewDiscoverer() error = %v, wantErr %v", err, true)
	}

	if d == nil {
		t.Errorf("NewDiscoverer() = %v, want not nil", d)
	}
}

func TestDiscoverer_run_nil_service(t *testing.T) {
	d := &Discoverer{data: &state{services: emptyServiceMap}, services: nil, stop: make(chan struct{})}
	go d.Run()

	var once sync.Once

	once.Do(func() {
		time.Sleep(100 * time.Millisecond)
		d.Shutdown()
	})
}

func TestDiscoverer_run_fake_service(t *testing.T) {
	d := &Discoverer{data: &state{services: emptyServiceMap}, services: &fakeController{}, stop: make(chan struct{})}
	go d.Run()

	var once sync.Once

	once.Do(func() {
		time.Sleep(100 * time.Millisecond)
		d.Shutdown()
	})
}

func TestDiscoverer_updateServices(t *testing.T) {
	// handle the initial run and run where data does not change
	d := &Discoverer{data: &state{services: emptyServiceMap}, services: &fakeController{}, stop: make(chan struct{})}

	type args struct {
		s *models.Service
		e models.Event
	}

	tests := []struct {
		name string
		args args
	}{
		{
			name: "initial run",
			args: args{s: testServices[0], e: models.EventAdd},
		},
		{
			name: "next run with no data change",
			args: args{s: testServices[0], e: models.EventUpdate},
		},
		{
			name: "next run with delete data",
			args: args{s: testServices[0], e: models.EventDelete},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d.updateServices(tt.args.s, tt.args.e)
		})
	}

	// trigger failure within discover()
	d = &Discoverer{data: &state{services: emptyServiceMap}, services: &fakeController{}, stop: make(chan struct{})}
	d.updateServices(testServices[0], models.EventAdd)

	// trigger failure from services call
	d = &Discoverer{data: &state{services: emptyServiceMap}, services: &fakeController{wantErr: true, wantNil: false}, stop: make(chan struct{})}
	d.updateServices(testServices[0], models.EventAdd)

	// trigger no data from services call
	d = &Discoverer{data: &state{services: emptyServiceMap}, services: &fakeController{wantErr: false, wantNil: true}, stop: make(chan struct{})}
	d.updateServices(testServices[0], models.EventAdd)
}

func TestDiscoverer_updateDeployments(t *testing.T) {
	// handle the initial run and run where data does not change
	d := &Discoverer{data: &state{services: emptyServiceMap, deployments: emptyDeploymentMap}, services: &fakeController{}, stop: make(chan struct{})}

	type args struct {
		dpl *models.Deployment
		e   models.Event
	}

	tests := []struct {
		name string
		args args
	}{
		{
			name: "initial run",
			args: args{dpl: testDeployments[0], e: models.EventAdd},
		},
		{
			name: "next run with no data change",
			args: args{dpl: testDeployments[0], e: models.EventUpdate},
		},
		{
			name: "next run with delete data",
			args: args{dpl: testDeployments[0], e: models.EventDelete},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d.updateDeployments(tt.args.dpl, tt.args.e)
		})
	}

	// trigger failure within discover()
	d = &Discoverer{data: &state{services: emptyServiceMap, deployments: emptyDeploymentMap}, services: &fakeController{}, stop: make(chan struct{})}
	d.updateDeployments(testDeployments[0], models.EventAdd)

	// trigger failure from services call
	d = &Discoverer{data: &state{services: emptyServiceMap, deployments: emptyDeploymentMap}, services: &fakeController{wantErr: true, wantNil: false}, stop: make(chan struct{})}
	d.updateDeployments(testDeployments[0], models.EventAdd)

	// trigger no data from services call
	d = &Discoverer{data: &state{services: emptyServiceMap, deployments: emptyDeploymentMap}, services: &fakeController{wantErr: false, wantNil: true}, stop: make(chan struct{})}
	d.updateDeployments(testDeployments[0], models.EventAdd)
}
