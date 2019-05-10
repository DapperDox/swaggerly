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
package proxy

import (
	"dapperdox/config"
	"dapperdox/logger"
	"github.com/gorilla/pat"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"
)

type responseCapture struct {
	http.ResponseWriter
	statusCode int
}

func (r *responseCapture) WriteHeader(status int) {
	r.statusCode = status
	r.ResponseWriter.WriteHeader(status)
}

// -----------------------------------------------------------------------------

func Register(r *pat.Router) {
	cfg, _ := config.Get() // Don't worry about error. If there was something wrong with the config, we'd know by now.

	logger.Tracef(nil, "Registering proxied paths:\n")

	for i := range cfg.ProxyPath {
		slice := strings.Split(cfg.ProxyPath[i], "=")
		switch len(slice) {
		case 2:
			register(r, slice[0], slice[1])
		default:
			panic("Invalid ProxyPath specified - does not contain an = delimited path=host/path pair")
		}
	}
	logger.Tracef(nil, "Registering proxied paths done.\n")
}

// -----------------------------------------------------------------------------

func register(r *pat.Router, routePattern string, target string) {

	u, _ := url.Parse(target)

	logger.Tracef(nil, "+ %s -> %s\n", routePattern, target)

	proxy := httputil.NewSingleHostReverseProxy(u)
	od := proxy.Director

	proxy.Director = func(r *http.Request) {
		od(r)
		r.Host = r.URL.Host // Rewrite Host

		scheme := "http://"
		if r.TLS != nil {
			scheme = "https://"
		}
		logger.Debugf(r, "Proxy request to: %s%s%s", scheme, r.Host, r.URL.Path)
	}

	r.PathPrefix(routePattern).HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rc := &responseCapture{w, 0}
		s := time.Now()
		logger.Tracef(r, "Proxy request started: %v", s)

		proxy.ServeHTTP(rc, r)

		e := time.Now()
		logger.Tracef(r, "Proxy request completed: %v", e)

		d := e.Sub(s)
		logger.Infof(r, "PROXY %s %s (%d, %v)", r.Method, r.URL.Path, rc.statusCode, d)
	})
}

// -----------------------------------------------------------------------------
