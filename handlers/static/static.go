package static

import (
	"mime"
	"net/http"
	"path/filepath"
	"strings"

	//"github.com/dapperdox/dapperdox/assets"
	"github.com/dapperdox/dapperdox/logger"
	"github.com/dapperdox/dapperdox/render"
	"github.com/dapperdox/dapperdox/render/asset"
	"github.com/gorilla/pat"
)

// Register creates routes for each static resource
func Register(r *pat.Router) {
	logger.Debugln(nil, "registering not found handler in static package")

	r.NotFoundHandler = http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		render.HTML(w, http.StatusNotFound, "error", render.DefaultVars(req, nil, map[string]interface{}{"error": "Page not found", "code": 404}))
	})

	logger.Debugln(nil, "registering static content handlers for static package")

	var allow bool

	for _, file := range asset.AssetNames() {
		mime.AddExtensionType(".woff", "font")
		mimeType := mime.TypeByExtension(filepath.Ext(file))

		switch {
		case strings.HasPrefix(mimeType, "image"),
			strings.HasPrefix(mimeType, "text/css"),
			strings.HasPrefix(mimeType, "font"),
			strings.HasSuffix(mimeType, "javascript"):
			allow = true
		default:
			allow = false
		}

		if allow {
			// Drop assets/static prefix
			path := strings.TrimPrefix(file, "assets/static")

			logger.Debugf(nil, "registering handler for static asset: %s", path)

			r.Path(path).Methods("GET").HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				if b, err := asset.Asset("assets/static" + path); err == nil {
					w.Header().Set("Content-Type", mimeType)
					w.Header().Set("Cache-control", "public, max-age=259200")
					w.WriteHeader(200)
					w.Write(b)
					return
				}
				// This should never happen!
				logger.Errorf(nil, "it happened ¯\\_(ツ)_/¯", path)
				r.NotFoundHandler.ServeHTTP(w, req)
			})
		}
	}
}
