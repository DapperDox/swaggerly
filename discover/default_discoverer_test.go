package discover

import (
	"testing"
)

func Test_defaultdiscoverer_Run(t *testing.T) {
	d := NewDefaultDiscoverer()

	d.Run()
	d.Shutdown()
}
