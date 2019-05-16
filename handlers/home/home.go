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
package home

import (
	"net/http"

	"github.com/frinka/dapperdox/config"
	"github.com/frinka/dapperdox/logger"
	"github.com/frinka/dapperdox/render"
	"github.com/frinka/dapperdox/spec"
	"github.com/gorilla/pat"
)

// ----------------------------------------------------------------------------------------
// Register creates routes for each home handler
func Register(r *pat.Router) {
	logger.Debugln(nil, "registering handlers for home page")

	count := 0
	// Homepages for each loaded specification
	var specification *spec.APISpecification // Ends up being populated with the last spec processed

	for _, specification = range spec.APISuite {

		logger.Tracef(nil, "Build homepage route for specification '%s'", specification.ID)

		r.Path("/" + specification.ID + "/reference").Methods("GET").HandlerFunc(specificationSummaryHandler(specification))

		// If missingh trailing slash, redirect to add it
		r.Path("/" + specification.ID).Methods("GET").HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			http.Redirect(w, req, "/"+specification.ID+"/", 302)
		})

		count++
	}

	cfg, _ := config.Get()

	if count == 1 && cfg.ForceSpecList == false {
		// If there is only one specification loaded, then hotwire '/' to redirect to the
		// specification summary page unless DapperDox is configured to show the specification list page.
		r.Path("/").Methods("GET").HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			http.Redirect(w, req, "/"+specification.ID+"/reference", 302)
		})
	} else {
		r.Path("/").Methods("GET").HandlerFunc(specificationListHandler)
	}
}

// ----------------------------------------------------------------------------------------
// Handler is a http.Handler for the specification list page
func specificationListHandler(w http.ResponseWriter, req *http.Request) {
	logger.Tracef(nil, "Render HTML for top level index page")

	render.HTML(w, http.StatusOK, "specification_list", render.DefaultVars(req, nil, render.Vars{"Title": "Specifications list", "SpecificationList": true}))
}

// ----------------------------------------------------------------------------------------
func specificationSummaryHandler(specification *spec.APISpecification) func(w http.ResponseWriter, req *http.Request) {

	// The default "theme" level reference index page.
	tmpl := "specification_summary"

	customTmpl := specification.ID + "/specification_summary"

	logger.Tracef(nil, "+ Test for template '%s'", customTmpl)

	if render.TemplateLookup(customTmpl) != nil {
		tmpl = customTmpl
	}
	return func(w http.ResponseWriter, req *http.Request) {
		render.HTML(w, http.StatusOK, tmpl, render.DefaultVars(req, specification, render.Vars{"Title": "Specification summary", "SpecificationSummary": true}))
	}
}

// ----------------------------------------------------------------------------------------
// end
