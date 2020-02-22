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
// Package timeout implements a timeoutHandler

// Mostly borrowed from core net/http.

package timeout

import (
	"errors"
	"net/http"
	"sync"
	"time"

	"github.com/DapperDox/dapperdox/logger"
)

// Handler returns a Handler that runs h with the given time limit.
//
// The new Handler calls h.ServeHTTP to handle each request, but if a
// call runs for longer than its time limit, the handler responds with
// a 503 Service Unavailable error and the given message in its body.
// (If msg is empty, a suitable default message will be sent.)
// After such a timeout, writes by h to its ResponseWriter will return
// ErrHandlerTimeout.
func Handler(h http.Handler, dt time.Duration, fh http.Handler) http.Handler {
	f := func() <-chan time.Time {
		return time.After(dt)
	}
	return &handler{h, f, fh}
}

// ErrHandlerTimeout is returned on ResponseWriter Write calls
// in handlers which have timed out.
var ErrHandlerTimeout = errors.New("http: Handler timeout")

type handler struct {
	handler     http.Handler
	timeout     func() <-chan time.Time // returns channel producing a timeout
	failHandler http.Handler
}

func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	done := make(chan bool, 1)
	tw := &writer{w: w}
	go func() {
		h.handler.ServeHTTP(tw, r)
		done <- true
	}()
	select {
	case <-done:
		return
	case <-h.timeout():
		tw.mu.Lock()
		defer tw.mu.Unlock()
		logger.Traceln(r, "request timed out")
		if !tw.wroteHeader {
			logger.Traceln(r, "headers not written, calling failure handler")
			h.failHandler.ServeHTTP(w, r)
		}
		tw.timedOut = true
	}
}

type writer struct {
	w http.ResponseWriter

	mu          sync.Mutex
	timedOut    bool
	wroteHeader bool
}

func (tw *writer) Header() http.Header {
	return tw.w.Header()
}

func (tw *writer) Write(p []byte) (int, error) {
	tw.mu.Lock()
	defer tw.mu.Unlock()
	tw.wroteHeader = true // implicitly at least
	if tw.timedOut {
		return 0, ErrHandlerTimeout
	}
	return tw.w.Write(p)
}

func (tw *writer) WriteHeader(code int) {
	tw.mu.Lock()
	defer tw.mu.Unlock()
	if tw.timedOut || tw.wroteHeader {
		return
	}
	tw.wroteHeader = true
	tw.w.WriteHeader(code)
}
