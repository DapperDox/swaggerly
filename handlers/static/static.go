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
package static

import (
	"mime"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/DapperDox/dapperdox/logger"
	"github.com/DapperDox/dapperdox/render"
	"github.com/DapperDox/dapperdox/render/asset"
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
		mimeType := mime.TypeByExtension(filepath.Ext(file))

		if mimeType == "" {
			continue
		}

		logger.Debugf(nil, "Got MIME type: %s", mimeType)

		switch {
		case strings.HasPrefix(mimeType, "image"),
			strings.HasPrefix(mimeType, "text/css"),
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
