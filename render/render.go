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
package render

import (
	"bufio"
	"bytes"
	"html/template"
	"net/http"
	"strings"

	"github.com/DapperDox/dapperdox/config"
	"github.com/DapperDox/dapperdox/logger"
	"github.com/DapperDox/dapperdox/navigation"
	"github.com/DapperDox/dapperdox/render/asset"
	"github.com/DapperDox/dapperdox/spec"
	"github.com/ian-kent/htmlform"
	"github.com/unrolled/render"
)

// Render is a global instance of github.com/unrolled/render.Render
var Render *render.Render

//var guides interface{}
type GuideType []*navigation.NavigationNode
type overlayPathList []string

var guides = map[string]GuideType{} // Guides are per specification-id, or 'top-level'

// Vars is a map of variables
type Vars map[string]interface{}

var counter int

// ----------------------------------------------------------------------------------------

func Register() {
	Render = New()
}

// ----------------------------------------------------------------------------------------
// New creates a new instance of github.com/unrolled/render.Render
func New() *render.Render {
	logger.Tracef(nil, "creating instance of render.Render")

	cfg, _ := config.Get()

	asset.CompileGFMMap()

	// XXX Order of directory importing is IMPORTANT XXX
	if len(cfg.AssetsDir) != 0 {
		asset.Compile(cfg.AssetsDir+"/templates", "assets/templates")
		asset.Compile(cfg.AssetsDir+"/static", "assets/static")
		asset.Compile(cfg.AssetsDir+"/themes/"+cfg.Theme, "assets")
		compileSections(cfg.AssetsDir)
	}

	// Import custom theme from custom directory (if defined)
	if len(cfg.Theme) != 0 {
		dir := cfg.DefaultAssetsDir + "/themes"
		if len(cfg.ThemeDir) != 0 {
			dir = cfg.ThemeDir
		}
		asset.Compile(dir+"/"+cfg.Theme, "assets")
	}

	if cfg.Theme != "default" {
		// The default theme underpins all others
		asset.Compile(cfg.DefaultAssetsDir+"/themes/default", "assets")
	}

	// Fallback to local templates directory
	asset.Compile(cfg.DefaultAssetsDir+"/templates", "assets/templates")
	// Fallback to local static directory
	asset.Compile(cfg.DefaultAssetsDir+"/static", "assets/static")

	return render.New(render.Options{
		Asset:      asset.Asset,
		AssetNames: asset.AssetNames,
		Directory:  "assets/templates",
		Delims:     render.Delims{Left: "[:", Right: ":]"},
		Layout:     "layout",
		Funcs: []template.FuncMap{{
			"map":           htmlform.Map,
			"ext":           htmlform.Extend,
			"fnn":           htmlform.FirstNotNil,
			"arr":           htmlform.Arr,
			"lc":            strings.ToLower,
			"uc":            strings.ToUpper,
			"join":          strings.Join,
			"concat":        func(a, b string) string { return a + b },
			"counter_set":   func(a int) int { counter = a; return counter },
			"counter_add":   func(a int) int { counter += a; return counter },
			"mod":           func(a int, m int) int { return a % m },
			"safehtml":      func(s string) template.HTML { return template.HTML(s) },
			"haveTemplate":  func(n string) *template.Template { return TemplateLookup(n) },
			"overlay":       func(n string, d ...interface{}) template.HTML { return overlay(n, d) },
			"getAssetPaths": func(s string, d ...interface{}) []string { return getAssetPaths(s, d) },
		}},
	})
}

// ----------------------------------------------------------------------------------------

func compileSections(assetsDir string) {
	// specification specific guides
	for _, specification := range spec.APISuite {
		logger.Debugf(nil, "- Specification assets for '%s'", specification.APIInfo.Title)
		compileSectionPart(assetsDir, specification, "templates", "assets/templates/")
		compileSectionPart(assetsDir, specification, "static", "assets/static/")
	}
}

// ----------------------------------------------------------------------------------------
func compileSectionPart(assetsDir string, spec *spec.APISpecification, part string, prefix string) {
	stem := spec.ID + "/" + part
	asset.Compile(assetsDir+"/sections/"+stem, prefix+stem)
}

// ----------------------------------------------------------------------------------------
type HTMLWriter struct {
	h *bufio.Writer
}

func (w HTMLWriter) Header() http.Header            { return http.Header{} }
func (w HTMLWriter) WriteHeader(int)                {}
func (w HTMLWriter) Write(data []byte) (int, error) { return w.h.Write(data) }
func (w HTMLWriter) Flush()                         { w.h.Flush() }

// XXX WHY ARRAY of DATA?
func overlay(name string, data []interface{}) template.HTML { // TODO Will be specification specific

	if data == nil || data[0] == nil {
		logger.Printf(nil, "Data nil\n")
		return ""
	}

	logger.Tracef(nil, "Overlay: Looking for overlay %s\n", name)

	var datamap map[string]interface{}
	var ok bool
	if datamap, ok = data[0].(map[string]interface{}); !ok {
		logger.Tracef(nil, "Overlay: type convert of data[0] to map[string]interface{} failed. Not an expected type.")
		return ""
	}

	overlayName := overlayPaths(name, datamap)

	var b bytes.Buffer
	var overlay string

	// Look for an overlay file in declaration order.... Highest priority is first.
	for _, overlay = range overlayName {
		logger.Tracef(nil, "Overlay: Does '%s' exist?\n", overlay)
		if TemplateLookup(overlay) != nil {
			break
		}
		overlay = ""
	}

	if overlay != "" {
		logger.Tracef(nil, "Applying overlay '%s'\n", overlay)
		writer := HTMLWriter{h: bufio.NewWriter(&b)}

		r := New()
		// data is a single item array (though I've not figured out why yet!)
		r.HTML(writer, http.StatusOK, overlay, data[0], render.HTMLOptions{Layout: ""})
		writer.Flush()
	}

	return template.HTML(b.String())
}

// ----------------------------------------------------------------------------------------

func overlayPaths(name string, datamap map[string]interface{}) []string {

	var overlayName []string

	// Use the passed in data structures to determine what type of "page" we are on:
	// 1. API summary page
	// 2. A method/operation page
	// 3. Resource
	// 4. Specification List page
	//
	if _, ok := datamap["API"].(spec.APIGroup); ok {
		if _, ok := datamap["Methods"].([]spec.Method); ok {
			getAPIAssetPaths(name, &overlayName, datamap)
		}
		if _, ok := datamap["Method"].(spec.Method); ok {
			getMethodAssetPaths(name, &overlayName, datamap)
		}
	}
	if _, ok := datamap["Resource"].(*spec.Resource); ok {
		getResourceAssetPaths(name, &overlayName, datamap)
	}
	if _, ok := datamap["SpecificationList"]; ok {
		getSpecificationListPaths(name, &overlayName, datamap)
	}
	if _, ok := datamap["SpecificationSummary"]; ok {
		getSpecificationSummaryPaths(name, &overlayName, datamap)
	}

	return overlayName
}

// ----------------------------------------------------------------------------------------
// HTML is an alias to github.com/unrolled/render.Render.HTML
func HTML(w http.ResponseWriter, status int, name string, binding interface{}, htmlOpt ...render.HTMLOptions) {
	Render.HTML(w, status, name, binding, htmlOpt...)
}

// ----------------------------------------------------------------------------------------
func TemplateLookup(t string) *template.Template {
	return Render.TemplateLookup(t)
}

// ----------------------------------------------------------------------------------------
// DefaultVars adds the default vars (config, specs, others....) to the data map
func DefaultVars(req *http.Request, apiSpec *spec.APISpecification, m Vars) map[string]interface{} {
	if m == nil {
		logger.Traceln(req, "creating new template data map")
		m = make(map[string]interface{})
	}

	cfg, _ := config.Get()
	m["Config"] = cfg
	m["APISuite"] = spec.APISuite

	// If we have a multiple specifications or are forcing a parent "root" page for the single specification
	// then set MultipleSpecs to true to enable navigation back to the root page.
	if cfg.ForceSpecList || len(spec.APISuite) > 1 {
		m["MultipleSpecs"] = true
	}

	if apiSpec == nil {
		m["NavigationGuides"] = guides[""] // Global guides
		m["SpecPath"] = ""

		return m
	}

	// Per specification defaults
	m["NavigationGuides"] = guides[apiSpec.ID]

	m["ID"] = apiSpec.ID
	m["SpecPath"] = "/" + apiSpec.ID
	m["APIs"] = apiSpec.APIs
	m["APIVersions"] = apiSpec.APIVersions
	m["Resources"] = apiSpec.ResourceList
	m["Info"] = apiSpec.APIInfo
	m["SpecURL"] = apiSpec.URL

	return m
}

// ----------------------------------------------------------------------------------------
func SetGuidesNavigation(apiSpec *spec.APISpecification, guidesnav *[]*navigation.NavigationNode) {
	id := ""
	if apiSpec != nil {
		id = apiSpec.ID
	}
	guides[id] = *guidesnav
}

// ----------------------------------------------------------------------------------------

func getAssetPaths(name string, data []interface{}) []string {
	datamap := data[0].(map[string]interface{})

	var paths []string

	if _, ok := datamap["API"]; ok {
		if _, ok := datamap["Methods"]; ok {
			// API-group summary page - Shows operations in a group
			getAPIAssetPaths("", &paths, datamap)
			return paths
		}
	}
	if _, ok := datamap["Method"]; ok {
		getMethodAssetPaths("", &paths, datamap) // Method page
		return paths
	}
	if _, ok := datamap["Resource"]; ok {
		getResourceAssetPaths("", &paths, datamap) // Resource page
		return paths
	}
	if _, ok := datamap["SpecificationList"]; ok {
		getSpecificationListPaths("", &paths, datamap) // Specification List page
		return paths
	}
	if _, ok := datamap["SpecificationSummary"]; ok {
		getSpecificationSummaryPaths("", &paths, datamap) // Specification List page
		return paths
	}

	return nil
}

// ----------------------------------------------------------------------------------------
// Some path stem and asset name helper stuff, to allow the path generation code to
// create asset file paths (for author debug), or the imported assets they create (use by
// the overlay handler).
type overlayStems struct {
	specStem   string
	globalStem string
	asset      string
}

func getOverlayStems(overlayAsset string) *overlayStems {
	a := &overlayStems{
		specStem:   "assets/sections/",
		globalStem: "assets/templates/",
		asset:      ".md",
	}
	if len(overlayAsset) > 0 {
		a.specStem = ""
		a.globalStem = ""
		a.asset = "/" + overlayAsset + "/overlay"
	}
	return a
}

// ----------------------------------------------------------------------------------------

func getMethodAssetPaths(overlayAsset string, paths *[]string, datamap map[string]interface{}) {

	method := datamap["Method"].(spec.Method)
	apiID := method.APIGroup.ID

	a := getOverlayStems(overlayAsset)

	if specID, ok := datamap["ID"].(string); ok {
		*paths = append(*paths, a.specStem+specID+"/templates/reference/"+apiID+"/"+method.ID+a.asset)
		*paths = append(*paths, a.specStem+specID+"/templates/reference/"+apiID+"/"+method.Method+a.asset)
		*paths = append(*paths, a.specStem+specID+"/templates/reference/"+apiID+"/method"+a.asset)

		*paths = append(*paths, a.specStem+specID+"/templates/reference/"+method.ID+a.asset)
		*paths = append(*paths, a.specStem+specID+"/templates/reference/"+method.Method+a.asset)
		*paths = append(*paths, a.specStem+specID+"/templates/reference/method"+a.asset)
	}

	*paths = append(*paths, a.globalStem+"reference/"+method.ID+a.asset)
	*paths = append(*paths, a.globalStem+"reference/"+method.Method+a.asset)
	*paths = append(*paths, a.globalStem+"reference/method"+a.asset)
}

// ----------------------------------------------------------------------------------------

func getAPIAssetPaths(overlayAsset string, paths *[]string, datamap map[string]interface{}) {

	apiID := datamap["API"].(spec.APIGroup).ID

	a := getOverlayStems(overlayAsset)

	if specID, ok := datamap["ID"].(string); ok {
		*paths = append(*paths, a.specStem+specID+"/templates/reference/"+apiID+a.asset)
		*paths = append(*paths, a.specStem+specID+"/templates/reference/api"+a.asset)
	}

	*paths = append(*paths, a.globalStem+"reference/api"+a.asset)
}

// ----------------------------------------------------------------------------------------

func getResourceAssetPaths(overlayAsset string, paths *[]string, datamap map[string]interface{}) {

	resID := datamap["Resource"].(*spec.Resource).ID
	a := getOverlayStems(overlayAsset)

	if specID, ok := datamap["ID"].(string); ok {
		*paths = append(*paths, a.specStem+specID+"/templates/resource/"+resID+a.asset)
		*paths = append(*paths, a.specStem+specID+"/templates/reference/resource"+a.asset)
	}

	*paths = append(*paths, a.globalStem+"resource/resource"+a.asset)
}

// ----------------------------------------------------------------------------------------

func getSpecificationListPaths(overlayAsset string, paths *[]string, datamap map[string]interface{}) {

	a := getOverlayStems(overlayAsset)
	*paths = append(*paths, a.globalStem+"reference/specification_list"+a.asset)
}

// ----------------------------------------------------------------------------------------

func getSpecificationSummaryPaths(overlayAsset string, paths *[]string, datamap map[string]interface{}) {

	a := getOverlayStems(overlayAsset)
	if specID, ok := datamap["ID"].(string); ok {
		*paths = append(*paths, a.specStem+specID+"/templates/reference/specification_summary"+a.asset)
	}
	*paths = append(*paths, a.globalStem+"reference/specification_summary"+a.asset)
}

// ----------------------------------------------------------------------------------------
// end
