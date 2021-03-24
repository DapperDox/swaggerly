package discover

import (
	"net/http"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"

	"github.com/kenjones-cisco/dapperdox/config"
)

const (
	discoveryBasePath = "discover"
)

var specReplacer *strings.Replacer

// DiscoverRegisterRoutes adds newly discovered spec into dapperdox routes.
func (d *Discoverer) DiscoverRegisterRoutes() {
	log().Info("Registering route specifications")

	// Build a replacer to search/replace specification URLs
	if specReplacer == nil {
		var replacements []string

		// Configure the replacer with key=value pairs
		for k, v := range viper.GetStringMapString(config.SpecRewriteURL) {
			if v != "" {
				// Map between configured to=from URL pair
				replacements = append(replacements, k, v)
			} else {
				// Map between configured URL and site URL
				replacements = append(replacements, k, viper.GetString(config.SiteURL))
			}
		}

		specReplacer = strings.NewReplacer(replacements...)

		for path, data := range d.Specs() {
			log().Debugf("  - %s", path)

			path = filepath.ToSlash(path)

			// Strip base path and file extension
			route := strings.TrimPrefix(path, discoveryBasePath)

			log().Debugf("    = URL : %s", route)
			log().Debugf("    + File: %s", path)

			d.router.Path(route).Methods(http.MethodGet).HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				log().Debugf("Serve discovered spec %s", route)
				w.Header().Set("Content-Type", "application/json")
				w.Header().Set("Cache-control", "public, max-age=259200")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(specReplacer.Replace(string(data))))
			})
		}
	}
}
