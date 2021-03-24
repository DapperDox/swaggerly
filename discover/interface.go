package discover

// DiscoveryManager Default Object for discoverer manager.
type DiscoveryManager interface {
	// Shutdown safely stops Discovery process
	Shutdown()
	// Run starts the discovery process
	Run()
}
