/*
Copyright (C) 2016-2017 dapperdox.com

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with this program.  If not, see <http://www.gnu.org/licenses/>.

*/
package main

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	"github.com/kenjones-cisco/dapperdox/config"
	"github.com/kenjones-cisco/dapperdox/discover"
	"github.com/kenjones-cisco/dapperdox/handlers"
	log "github.com/kenjones-cisco/dapperdox/logger"
	"github.com/kenjones-cisco/dapperdox/network"
	"github.com/kenjones-cisco/dapperdox/version"
)

func main() {
	pflag.Usage = func() {
		_, _ = fmt.Fprint(os.Stderr, "Usage:\n")
		_, _ = fmt.Fprintf(os.Stderr, "  %s [OPTIONS]\n\n", version.ShortName)
		_, _ = fmt.Fprintf(os.Stderr, "%s\n\n", version.ProductName)
		_, _ = fmt.Fprintln(os.Stderr, pflag.CommandLine.FlagUsages())
	}
	// parse the CLI flags
	pflag.Parse()
	_ = viper.BindPFlags(pflag.CommandLine)

	if viper.GetBool(config.Version) {
		fmt.Print(version.GetVersionDisplay())
		os.Exit(0)
	}

	config.Init()

	log.SetLevel(viper.GetString(config.LogLevel))

	// start auto-discovery process
	d, err := discover.NewDiscoverer()
	if err != nil {
		log.Logger().Warnf("unable to create Discoverer: %v", err)
		// set no-op default discoverer
		d = discover.NewDefaultDiscoverer()
	}

	go d.Run()

	chain := handlers.NewRouterChain()

	var listener net.Listener

	if viper.GetString(config.TLSCert) != "" && viper.GetString(config.TLSKey) != "" {
		listener, err = network.NewSecuredListener()
	} else {
		listener, err = network.NewListener()
	}

	if err != nil {
		log.Logger().Fatalf("Error listening on %s: %s", viper.GetString(config.BindAddr), err)
	}

	if err = http.Serve(listener, chain); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Logger().Fatalf("%v", err)
	}
}
