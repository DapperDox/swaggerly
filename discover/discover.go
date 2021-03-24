package discover

import (
	"strings"

	"github.com/gorilla/mux"
	"github.com/spf13/viper"

	"github.com/kenjones-cisco/dapperdox/config"
	"github.com/kenjones-cisco/dapperdox/discover/models"
)

// Discoverer represents the state of the discovery mechanism.
type Discoverer struct {
	services watcher

	data   *state
	stop   chan struct{}
	router *mux.Router

	vpr *viper.Viper

	specs map[string][]byte
}

type state struct {
	services    *models.ServiceMap
	deployments *models.DeploymentMap
}

var ignoredServices []string

// NewDiscoverer configures a new instance of a Discoverer using Kubernetes client.
func NewDiscoverer() (DiscoveryManager, error) {
	log().Info("initializing new discoverer instance")

	client, err := newClient()
	if err != nil {
		return nil, err
	}

	// copy Viper configurations to avoid any loss of viper
	// due to high number of goroutines in Auto-Discover process
	vprcpy := viper.GetViper()

	ctlg := newCatalog(client, catalogOptions{
		WatchedNamespace: vprcpy.GetString(config.DiscoveryNamespace),
		ResyncPeriod:     vprcpy.GetDuration(config.DiscoveryInterval),
		DomainSuffix:     "cluster.local",
	})

	sm := models.NewServiceMap()
	dm := models.NewDeploymentMap()

	d := &Discoverer{
		services: ctlg,
		data: &state{
			services:    &sm,
			deployments: &dm,
		},
		stop:  make(chan struct{}),
		vpr:   vprcpy,
		specs: make(map[string][]byte),
	}

	// register handlers; ignore errors as it will always return nil
	_ = ctlg.AppendServiceHandler(d.updateServices)
	_ = ctlg.AppendDeploymentHandler(d.updateDeployments)

	return d, nil
}

// Shutdown safely stops Discovery process.
func (d *Discoverer) Shutdown() {
	close(d.stop)
	log().Info("shutting down discovery process")
}

// Run starts the discovery process.
func (d *Discoverer) Run() {
	d.discover()

	if d.services != nil {
		go d.services.Run(d.stop)
	}
}

// Specs returns discovered API specs.
func (d *Discoverer) Specs() map[string][]byte {
	return d.specs
}

func (d *Discoverer) discover() {
	// fetch API specs from services and process the necessary
	// API changes to meet documentation requirements
	//  - remove private APIs and Methods
	//  - set necessary extensions for dapperdox
	//  - rewrite spec details for Schema, Security Definitions, Security
	specs := d.fetchAPISpecs()
	if len(specs) == 0 {
		return
	}

	// update local cache with latest service specs
	d.specs = specs

	// TODO(): dynamically register new routes and load new specs

	log().Debug("successfully processed API changes")
}

func (d *Discoverer) updateServices(s *models.Service, e models.Event) {
	log().Debugf("(Discover Handler) Service: %v Event: %v", s, e)

	if inIgnoreList(s.Hostname) {
		log().Debugf("%v service is blacklisted", s.Hostname)

		return
	}

	switch e {
	case models.EventAdd, models.EventUpdate:
		d.data.services.Insert(s)
	case models.EventDelete:
		d.data.services.Delete(s)
	}

	d.discover()
}

func (d *Discoverer) updateDeployments(dpl *models.Deployment, e models.Event) {
	log().Debugf("(Discover Handler) Deployment: %v Event: %v", dpl, e)

	switch e {
	case models.EventAdd, models.EventUpdate:
		d.data.deployments.Insert(dpl)
	case models.EventDelete:
		d.data.deployments.Delete(dpl)
	}

	d.discover()
}

func inIgnoreList(name string) bool {
	if len(ignoredServices) == 0 {
		ignoredServices = viper.GetStringSlice(config.DiscoveryServiceIgnoreList)
	}

	for _, svc := range ignoredServices {
		if strings.Contains(name, svc) {
			return true
		}
	}

	return false
}
