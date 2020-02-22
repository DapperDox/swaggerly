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
package logger

import (
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"time"
)

// Level is a log level
type Level int

const (
	// Error is the error level
	Error = Level(iota)
	// Warn is the warn level
	Warn
	// Info is the info level
	Info
	// Debug is the debug level
	Debug
	// Trace is the trace level
	Trace
)

var (
	// Logf is the function called for *f functions
	Logf = log.Printf
	// Logln is the function called for *ln functions
	Logln = log.Println
)

type responseCapture struct {
	http.ResponseWriter
	statusCode int
}

func (r *responseCapture) WriteHeader(status int) {
	r.statusCode = status
	r.ResponseWriter.WriteHeader(status)
}

// Handler wraps a http.Handler and logs the status code and total response time
func Handler(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		rc := &responseCapture{w, 0}

		s := time.Now()
		Tracef(req, "request started: %v", s)

		h.ServeHTTP(rc, req)

		e := time.Now()
		Tracef(req, "request completed: %v", e)

		d := e.Sub(s)
		Infof(req, "%s %s (%d, %v)", req.Method, req.URL.Path, rc.statusCode, d)
	})
}

// LevelString is a level to string lookup map
var LevelString = map[Level]string{
	Error: "error",
	Warn:  "warn",
	Info:  "info",
	Debug: "debug",
	Trace: "trace",
}

// LevelFromString returns a log level from a string
func LevelFromString(level string) (Level, error) {
	for i, j := range LevelString {
		if j == level {
			return i, nil
		}
	}
	return Info, fmt.Errorf("invalid log level, expected error|warn|info|debug|trace, got '%s'", level)
}

// DefaultLevel is the default log level
var DefaultLevel = func() Level {
	if v := strings.ToLower(os.Getenv("LOGLEVEL")); len(v) > 0 {
		for i, j := range LevelString {
			if j == v {
				return i
			}
		}
		log.Fatalf("invalid log level in environment variable LOGLEVEL, expected error|warn|info|debug|trace, got '%s'", v)
	}
	return Info
}()

// Levelf implements log.Printf but includes X-Request-Id and requires a log level
func Levelf(req *http.Request, level Level, format string, args ...interface{}) {
	if level > getRequestLevel(req) {
		return
	}

	if req != nil {
		args = append([]interface{}{getRequestID(req), LevelString[level]}, args...)
		format = "[%s] [%s] " + format
	}

	Logf(format, args...)
}

// Levelln implements log.Printf but includes X-Request-Id and requires a log level
func Levelln(req *http.Request, level Level, message ...interface{}) {
	if level > getRequestLevel(req) {
		return
	}

	if req != nil {
		message = append([]interface{}{fmt.Sprintf("[%s] [%s]", getRequestID(req), LevelString[level])}, message...)
	}

	Logln(message...)
}

// Printf implements log.Printf but includes X-Request-Id
func Printf(req *http.Request, format string, args ...interface{}) {
	Levelf(req, Info, format, args...)
}

// Println implements log.Println but includes X-Request-Id
func Println(req *http.Request, message ...interface{}) {
	Levelln(req, Info, message...)
}

// Errorf logs a message at Error level
func Errorf(req *http.Request, format string, args ...interface{}) {
	Levelf(req, Error, format, args...)
}

// Errorln logs a message at Error level
func Errorln(req *http.Request, message ...interface{}) {
	Levelln(req, Error, message...)
}

// Warnln logs a message at Warn level
func Warnln(req *http.Request, message ...interface{}) {
	Levelln(req, Warn, message...)
}

// Warnf logs a message at Warn level
func Warnf(req *http.Request, format string, args ...interface{}) {
	Levelf(req, Warn, format, args...)
}

// Infof logs a message at Info level
func Infof(req *http.Request, format string, args ...interface{}) {
	Levelf(req, Info, format, args...)
}

// Infoln logs a message at Info level
func Infoln(req *http.Request, message ...interface{}) {
	Levelln(req, Info, message...)
}

// Debugf logs a message at Debug level
func Debugf(req *http.Request, format string, args ...interface{}) {
	Levelf(req, Debug, format, args...)
}

// Debugln logs a message at Debug level
func Debugln(req *http.Request, message ...interface{}) {
	Levelln(req, Debug, message...)
}

// Tracef logs a message at Trace level
func Tracef(req *http.Request, format string, args ...interface{}) {
	Levelf(req, Trace, format, args...)
}

// Traceln logs a message at Trace level
func Traceln(req *http.Request, message ...interface{}) {
	Levelln(req, Trace, message...)
}

func getRequestLevel(req *http.Request) Level {
	// TODO: configurable per-request?
	return DefaultLevel
}

func getRequestID(req *http.Request) (requestID string) {
	if requestID = req.Header.Get("X-Request-Id"); len(requestID) == 0 {
		requestID = randSeq(20)
		req.Header.Set("X-Request-Id", requestID)
	}
	return
}

var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func randSeq(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}
