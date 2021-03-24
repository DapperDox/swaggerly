package discover

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/justinas/nosurf"
	"github.com/spf13/viper"

	"github.com/kenjones-cisco/dapperdox/config"
	"github.com/kenjones-cisco/dapperdox/handlers/guides"
	"github.com/kenjones-cisco/dapperdox/handlers/home"
	"github.com/kenjones-cisco/dapperdox/handlers/proxy"
	"github.com/kenjones-cisco/dapperdox/handlers/reference"
	"github.com/kenjones-cisco/dapperdox/handlers/static"
	"github.com/kenjones-cisco/dapperdox/handlers/timeout"
	"github.com/kenjones-cisco/dapperdox/render"
	"github.com/kenjones-cisco/dapperdox/spec"
	"github.com/kenjones-cisco/dapperdox/version"
)

// NewDiscoverRouterChain creates a dynamic httpHandler based on discovered API specs.
func (d *Discoverer) NewDiscoverRouterChain() {
	router := mux.NewRouter()
	router.Use(
		handlers.RecoveryHandler(handlers.RecoveryLogger(log()), handlers.PrintRecoveryStack(true)),
		withLogger,
		timeoutHandler,
		withCsrf,
		injectHeaders,
		handlers.CORS(handlers.AllowedOrigins(viper.GetStringSlice(config.AllowOrigin))),
	)

	// custom route register leveraging discovered API specs
	// stored in Discoverer instance
	d.DiscoverRegisterRoutes()

	// TODO(): add custom spec loader
	if err := spec.LoadSpecifications(); err != nil {
		log().Fatalf("Load specification error: %s", err)
	}

	render.Register()

	reference.Register(router)
	guides.Register(router)
	static.Register(router)
	home.Register(router)
	proxy.Register(router)

	d.router = router
}

func withLogger(h http.Handler) http.Handler {
	return handlers.CombinedLoggingHandler(os.Stdout, h)
}

func withCsrf(h http.Handler) http.Handler {
	csrfHandler := nosurf.New(h)
	csrfHandler.SetFailureHandler(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		rsn := nosurf.Reason(req).Error()
		log().Warnf("failed csrf validation: %s", rsn)
		render.HTML(w, http.StatusBadRequest, "error", map[string]interface{}{"error": rsn})
	}))

	return csrfHandler
}

func timeoutHandler(h http.Handler) http.Handler {
	return timeout.Handler(h, 1*time.Second, http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		log().Warn("request timed out")
		render.HTML(w, http.StatusRequestTimeout, "error", map[string]interface{}{"error": "Request timed out"})
	}))
}

// Handle additional headers such as strict transport security for TLS, and
// giving the Server name.
func injectHeaders(h http.Handler) http.Handler {
	tlsEnabled := viper.GetString(config.TLSCert) != "" && viper.GetString(config.TLSKey) != ""

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Server", fmt.Sprintf("%s %s", version.ProductName, version.Version))

		if tlsEnabled {
			w.Header().Add("Strict-Transport-Security", "max-age=63072000; includeSubDomains")
		}

		h.ServeHTTP(w, r)
	})
}
