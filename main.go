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
	"log"
	"net"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/frinka/dapperdox/spec"

	"github.com/frinka/dapperdox/config"
	"github.com/frinka/dapperdox/handlers/guides"
	"github.com/frinka/dapperdox/handlers/home"
	"github.com/frinka/dapperdox/handlers/reference"
	"github.com/frinka/dapperdox/handlers/specs"
	"github.com/frinka/dapperdox/handlers/static"
	"github.com/frinka/dapperdox/handlers/timeout"
	"github.com/frinka/dapperdox/logger"
	"github.com/frinka/dapperdox/network"
	"github.com/frinka/dapperdox/proxy"
	"github.com/frinka/dapperdox/render"

	"github.com/gorilla/pat"
	"github.com/justinas/alice"
	"github.com/justinas/nosurf"
)

var VERSION string = "1.2.3"
var tlsEnabled bool

// ---------------------------------------------------------------------------
func main() {
	tlsEnabled = false
	log.Printf("DapperDox server version %s starting\n", VERSION)

	os.Setenv("GOFIGURE_ENV_ARRAY", "1") // Enable gofigure array parsing of env vars

	cfg, err := config.Get()
	if err != nil {
		log.Fatalf("error configuring app: %s", err)
	}

	// logging before this point must rely on setting LOGLEVEL env var
	if l, err := logger.LevelFromString(cfg.LogLevel); err == nil {
		logger.DefaultLevel = l
	} else {
		logger.Errorf(nil, "error setting log level: %s", err)
		os.Exit(1)
	}

	router := pat.New()
	chain := alice.New(logger.Handler /*, context.ClearHandler*/, timeoutHandler, withCsrf, injectHeaders).Then(router)

	logger.Infof(nil, "listening on %s", cfg.BindAddr)
	listener, err := net.Listen("tcp", cfg.BindAddr)
	if err != nil {
		logger.Errorf(nil, "%s", err)
		os.Exit(1)
	}

	var wg sync.WaitGroup
	var sg sync.WaitGroup
	sg.Add(1)

	go func() {
		logger.Traceln(nil, "Listen for and serve swagger spec requests for start up")
		wg.Add(1)
		sg.Done()
		http.Serve(listener, chain)
		logger.Traceln(nil, "Finished service swagger specs for start up")
		wg.Done()
	}()

	sg.Wait()

	// Register the spec routes (Listener and server must be up and running by now)
	specs.Register(router)
	spec.LoadStatusCodes()

	if cfg.SpecURL != "" {
		err = spec.LoadSpecification(cfg.SpecURL)
	} else {
		err = spec.LoadSpecifications(cfg.BindAddr, true)
	}

	if err != nil {
		logger.Errorf(nil, "Load specification error: %s", err)
		os.Exit(1)
	}

	render.Register()

	reference.Register(router)
	guides.Register(router)
	static.Register(router) // TODO - Static content should be capable of being CDN hosted

	home.Register(router)
	proxy.Register(router)

	listener.Close() // Stop serving specs
	wg.Wait()        // wait for go routine serving specs to terminate

	listener, err = network.GetListener(&tlsEnabled)
	if err != nil {
		logger.Errorf(nil, "Error listening on %s: %s", cfg.BindAddr, err)
		os.Exit(1)
	}

	http.Serve(listener, chain)
}

// ---------------------------------------------------------------------------
func withCsrf(h http.Handler) http.Handler {
	csrfHandler := nosurf.New(h)
	csrfHandler.SetFailureHandler(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		rsn := nosurf.Reason(req).Error()
		logger.Warnf(req, "failed csrf validation: %s", rsn)
		render.HTML(w, http.StatusBadRequest, "error", map[string]interface{}{"error": rsn})
	}))
	return csrfHandler
}

// ---------------------------------------------------------------------------
func timeoutHandler(h http.Handler) http.Handler {
	return timeout.Handler(h, 1*time.Second, http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		logger.Warnln(req, "request timed out")
		render.HTML(w, http.StatusRequestTimeout, "error", map[string]interface{}{"error": "Request timed out"})
	}))
}

// ---------------------------------------------------------------------------
// Handle additional headers such as strict transport security for TLS, and
// giving the Server name.
func injectHeaders(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Server", "DapperDox "+VERSION)

		if tlsEnabled {
			w.Header().Add("Strict-Transport-Security", "max-age=63072000; includeSubDomains")
		}

		h.ServeHTTP(w, r)
	})
}

// ---------------------------------------------------------------------------
