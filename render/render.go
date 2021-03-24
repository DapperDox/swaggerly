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

// Package render handles rendering markdown documentation.
package render

import (
	"bufio"
	"bytes"
	"html/template"
	"io"
	"math"
	"net/http"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"

	"github.com/spf13/viper"
	"github.com/unrolled/render"

	"github.com/kenjones-cisco/dapperdox/config"
	"github.com/kenjones-cisco/dapperdox/render/asset"
	"github.com/kenjones-cisco/dapperdox/spec"
)

var (
	// global instance of github.com/unrolled/render.Render.
	_render *render.Render

	counter int
)

// Register is alias for initializing new render.Render.
func Register() {
	log().Debug("initializing Render")

	_render = newRender()
}

// HTML is an alias to github.com/unrolled/render.Render.HTML.
func HTML(w io.Writer, status int, name string, binding interface{}, htmlOpt ...render.HTMLOptions) {
	_ = _render.HTML(w, status, name, binding, htmlOpt...)
}

// TemplateLookup is an alias to github.com/unrolled/render.TemplateLookup.
func TemplateLookup(t string) *template.Template {
	return _render.TemplateLookup(t)
}

func newRender() *render.Render {
	log().Trace("creating instance of render.Render")

	asset.CompileGFMMap()

	// XXX Order of directory importing is IMPORTANT XXX
	if viper.GetString(config.AssetsDir) != "" {
		asset.Compile(filepath.Join(viper.GetString(config.AssetsDir), "templates"), "assets/templates")
		asset.Compile(filepath.Join(viper.GetString(config.AssetsDir), "static"), "assets/static")
		asset.Compile(filepath.Join(viper.GetString(config.AssetsDir), "themes", viper.GetString(config.Theme)), "assets")
		compileSections(viper.GetString(config.AssetsDir))
	}

	// Import custom theme from custom directory (if defined)
	if viper.GetString(config.Theme) != "" {
		dir := filepath.Join(viper.GetString(config.DefaultAssetsDir), "themes")
		if viper.GetString(config.ThemeDir) != "" {
			dir = viper.GetString(config.ThemeDir)
		}

		asset.Compile(filepath.Join(dir, viper.GetString(config.Theme)), "assets")
	}

	if viper.GetString(config.Theme) != "default" {
		// The default theme underpins all others
		asset.Compile(filepath.Join(viper.GetString(config.DefaultAssetsDir), "themes", "default"), "assets")
	}

	compileSections(viper.GetString(config.DefaultAssetsDir))

	// Fallback to local templates directory
	asset.Compile(filepath.Join(viper.GetString(config.DefaultAssetsDir), "templates"), "assets/templates")
	// Fallback to local static directory
	asset.Compile(filepath.Join(viper.GetString(config.DefaultAssetsDir), "static"), "assets/static")

	return render.New(render.Options{
		Asset:      asset.Asset,
		AssetNames: asset.Names,
		Directory:  "assets/templates",
		Delims:     render.Delims{Left: "[:", Right: ":]"},
		Layout:     "layout",
		Funcs: []template.FuncMap{{
			"map":           Map,
			"ext":           Extend,
			"fnn":           FirstNotNil,
			"arr":           Arr,
			"lc":            strings.ToLower,
			"uc":            strings.ToUpper,
			"join":          strings.Join,
			"concat":        func(a, b string) string { return a + b },
			"counter_set":   func(a int) int { counter = a; return counter },
			"counter_add":   func(a int) int { counter += a; return counter },
			"mod":           func(a int, m int) int { return a % m },
			"sub":           func(a, b interface{}) int64 { return toInt64(a) - toInt64(b) },
			"safehtml":      func(s string) template.HTML { return template.HTML(s) },
			"haveTemplate":  TemplateLookup,
			"overlay":       func(n string, d ...interface{}) template.HTML { return overlayFunc(n, d) },
			"getAssetPaths": func(s string, d ...interface{}) []string { return getAssetPaths(s, d) },
		}},
	})
}

func compileSections(assetsDir string) {
	// specification specific guides
	for _, specification := range spec.APISuite {
		log().Debugf("- Specification assets for %q", specification.APIInfo.Title)
		compileSectionPart(specification.ID, assetsDir, "templates", "assets/templates/")
		compileSectionPart(specification.ID, assetsDir, "static", "assets/static/")
	}
}

func compileSectionPart(id, assetsDir, part, prefix string) {
	stem := filepath.Join(id, part)
	asset.Compile(filepath.Join(assetsDir, "sections", stem), filepath.Join(prefix, stem))
}

// htmlWriter implements an HTML Writer interface.
type htmlWriter struct {
	h *bufio.Writer
}

// Header provides empty implementation.
func (w htmlWriter) Header() http.Header { return http.Header{} }

// WriteHeader provides empty implementation.
func (w htmlWriter) WriteHeader(int) {}

// Write provides empty implementation.
func (w htmlWriter) Write(data []byte) (int, error) { return w.h.Write(data) }

// Flush provides empty implementation.
func (w htmlWriter) Flush() { _ = w.h.Flush() }

// XXX WHY ARRAY of DATA?
func overlayFunc(name string, data []interface{}) template.HTML { // TODO Will be specification specific
	if len(data) == 0 || data[0] == nil {
		log().Debug("Data nil")

		return ""
	}

	log().Tracef("Overlay: Looking for overlay %s", name)

	datamap, ok := data[0].(map[string]interface{})
	if !ok {
		log().Trace("Overlay: type convert of data[0] to map[string]interface{} failed. Not an expected type.")

		return ""
	}

	var b bytes.Buffer

	// Look for an overlay file in declaration order.... Highest priority is first.
	for _, op := range overlayPaths(name, datamap) {
		log().Tracef("Overlay: Does %q exist?", op)

		if TemplateLookup(op) != nil {
			log().Tracef("Applying overlay %q", op)

			writer := htmlWriter{h: bufio.NewWriter(&b)}

			// data is a single item array (though I've not figured out why yet!)
			_ = newRender().HTML(writer, http.StatusOK, op, data[0], render.HTMLOptions{Layout: ""})
			writer.Flush()

			break
		}
	}

	return template.HTML(b.String())
}

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

func getAssetPaths(_ string, data []interface{}) []string {
	datamap, _ := data[0].(map[string]interface{})

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

// Some path stem and asset name helper stuff, to allow the path generation code to
// create asset file paths (for author debug), or the imported assets they create (use by
// the overlay handler).
type overlayStems struct {
	specStem   string
	globalStem string
	asset      string
}

func getOverlayStems(overlayAsset string) *overlayStems {
	if overlayAsset != "" {
		return &overlayStems{
			asset: "/" + overlayAsset + "/overlay",
		}
	}

	return &overlayStems{
		specStem:   "assets/sections/",
		globalStem: "assets/templates/",
		asset:      ".md",
	}
}

func getMethodAssetPaths(overlayAsset string, paths *[]string, datamap map[string]interface{}) {
	method, _ := datamap["Method"].(spec.Method)
	apiID := method.APIGroup.ID

	a := getOverlayStems(overlayAsset)

	if specID, ok := datamap["ID"].(string); ok {
		*paths = append(*paths,
			a.specStem+specID+"/templates/reference/"+apiID+"/"+method.ID+a.asset,
			a.specStem+specID+"/templates/reference/"+apiID+"/"+method.Method+a.asset,
			a.specStem+specID+"/templates/reference/"+apiID+"/method"+a.asset,
			a.specStem+specID+"/templates/reference/"+method.ID+a.asset,
			a.specStem+specID+"/templates/reference/"+method.Method+a.asset,
			a.specStem+specID+"/templates/reference/method"+a.asset,
		)
	}

	*paths = append(*paths,
		a.globalStem+"reference/"+method.ID+a.asset,
		a.globalStem+"reference/"+method.Method+a.asset,
		a.globalStem+"reference/method"+a.asset,
	)
}

func getAPIAssetPaths(overlayAsset string, paths *[]string, datamap map[string]interface{}) {
	apiID := datamap["API"].(spec.APIGroup).ID

	a := getOverlayStems(overlayAsset)

	if specID, ok := datamap["ID"].(string); ok {
		*paths = append(*paths,
			a.specStem+specID+"/templates/reference/"+apiID+a.asset,
			a.specStem+specID+"/templates/reference/api"+a.asset,
		)
	}

	*paths = append(*paths, a.globalStem+"reference/api"+a.asset)
}

func getResourceAssetPaths(overlayAsset string, paths *[]string, datamap map[string]interface{}) {
	resID := datamap["Resource"].(*spec.Resource).ID
	a := getOverlayStems(overlayAsset)

	if specID, ok := datamap["ID"].(string); ok {
		*paths = append(*paths,
			a.specStem+specID+"/templates/resource/"+resID+a.asset,
			a.specStem+specID+"/templates/reference/resource"+a.asset,
		)
	}

	*paths = append(*paths, a.globalStem+"resource/resource"+a.asset)
}

func getSpecificationListPaths(overlayAsset string, paths *[]string, _ map[string]interface{}) {
	a := getOverlayStems(overlayAsset)
	*paths = append(*paths, a.globalStem+"reference/specification_list"+a.asset)
}

func getSpecificationSummaryPaths(overlayAsset string, paths *[]string, datamap map[string]interface{}) {
	a := getOverlayStems(overlayAsset)

	if specID, ok := datamap["ID"].(string); ok {
		*paths = append(*paths, a.specStem+specID+"/templates/reference/specification_summary"+a.asset)
	}

	*paths = append(*paths, a.globalStem+"reference/specification_summary"+a.asset)
}

// toInt64 converts integer types to 64-bit integers.
func toInt64(v interface{}) int64 {
	if str, ok := v.(string); ok {
		iv, err := strconv.ParseInt(str, 10, 64)
		if err != nil {
			return 0
		}

		return iv
	}

	val := reflect.Indirect(reflect.ValueOf(v))

	switch val.Kind() {
	case reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Int:
		return val.Int()
	case reflect.Uint8, reflect.Uint16, reflect.Uint32:
		return int64(val.Uint())
	case reflect.Uint, reflect.Uint64:
		tv := val.Uint()
		if tv <= math.MaxInt64 {
			return int64(tv)
		}

		// TODO: What is the sensible thing to do here?
		return math.MaxInt64
	case reflect.Float32, reflect.Float64:
		return int64(val.Float())
	case reflect.Bool:
		if val.Bool() {
			return 1
		}

		return 0
	default:
		return 0
	}
}
