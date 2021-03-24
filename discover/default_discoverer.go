package discover

type defaultDiscoverer struct{}

// NewDefaultDiscoverer Default Object for client manager.
func NewDefaultDiscoverer() DiscoveryManager {
	return &defaultDiscoverer{}
}

func (d *defaultDiscoverer) Shutdown() {
	log().Info("Discoverer not implemented")
}

// Run starts the discovery process.
func (d *defaultDiscoverer) Run() {
	log().Info("Discoverer not implemented")
}
