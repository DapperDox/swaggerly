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

// Package spec provides API spec loading and parsing.
package spec

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/url"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/go-openapi/loads"
	"github.com/go-openapi/spec"
	"github.com/go-openapi/swag"
	"github.com/serenize/snaker"
	"github.com/spf13/viper"

	"github.com/kenjones-cisco/dapperdox/config"
	"github.com/kenjones-cisco/dapperdox/formatter"
)

const (
	arrayType = "array"

	excludeOpExt     = "x-excludeFromOperations"
	navMethodNameExt = "x-navigateMethodsByName"
	opNameExt        = "x-operationName"
	pathNameExt      = "x-pathName"
	sortMethodsByExt = "x-sortMethodsBy"
	groupByExt       = "x-groupby"
	versionExt       = "x-version"
	visibilityExt    = "x-visibility"
)

// all defined ResourceOrigin.
const (
	RequestBody ResourceOrigin = iota
	MethodResponse
)

var kababExclude = regexp.MustCompile(`[^\w\s]`) // Any non word or space character

var collectionTable = map[string]string{
	"csv":   "comma separated",
	"ssv":   "space separated",
	"tsv":   "tab separated",
	"pipes": "pipe separated",
	"multi": "multiple occurrences",
}

var sortTypes = map[string]bool{
	"path":       true,
	"method":     true,
	"operation":  true,
	"navigation": true,
	"summary":    true,
}

// APISuite holds multiple apis held by name.
var APISuite map[string]*APISpecification

// APISuiteGroups holds multiple apis sorted by groups.
var APISuiteGroups map[string][]*APISpecification

// APISpecification holds the content of a parsed api.
type APISpecification struct {
	ID      string
	APIs    APISet // APIs represents the parsed APIs
	APIInfo Info
	URL     string
	GroupBy string

	SecurityDefinitions map[string]SecurityScheme
	DefaultSecurity     map[string]Security
	ResourceList        map[string]map[string]*Resource // Version->ResourceName->Resource
	APIVersions         map[string]APISet               // Version->APISet
}

// APISet list of grouped APIs.
type APISet []APIGroup

// APIGroup parents all grouped API methods (Grouping controlled by tagging, if used, or by method path otherwise).
type APIGroup struct {
	ID                     string
	Name                   string
	URL                    *url.URL
	MethodNavigationByName bool
	MethodSortBy           []string
	Versions               map[string][]Method // All versions, keyed by version string.
	Methods                []Method            // The current version
	CurrentVersion         string              // The latest version in operation for the API
	Info                   *Info
	Consumes               []string
	Produces               []string
}

// Info holds display information about API.
type Info struct {
	Title       string
	Description string
}

// Version holds version to list of associated method.
type Version struct {
	Version string
	Methods []Method
}

// OAuth2Scheme is a specific security scheme.
type OAuth2Scheme struct {
	OAuth2Flow       string
	AuthorizationURL string
	TokenURL         string
	Scopes           map[string]string
}

// SecurityScheme holds the security scheme from a parsed api.
type SecurityScheme struct {
	IsAPIKey      bool
	IsBasic       bool
	IsOAuth2      bool
	Type          string
	Description   string
	ParamName     string
	ParamLocation string
	OAuth2Scheme
}

// Security holds the defined enabled security.
type Security struct {
	Scheme *SecurityScheme
	Scopes map[string]string
}

// Method represents an API method.
type Method struct {
	ID              string
	Name            string
	Description     string
	Method          string
	OperationName   string
	NavigationName  string
	Path            string
	Consumes        []string
	Produces        []string
	PathParams      []Parameter
	QueryParams     []Parameter
	HeaderParams    []Parameter
	BodyParam       *Parameter
	FormParams      []Parameter
	Responses       map[int]Response
	DefaultResponse *Response // A ptr to allow of easy checking of its existence in templates
	Resources       []*Resource
	Security        map[string]Security
	APIGroup        *APIGroup
	SortKey         string
}

// Parameter represents an API method parameter.
type Parameter struct {
	Type                        []string
	Enum                        []string
	Name                        string
	Description                 string
	In                          string
	CollectionFormat            string
	CollectionFormatDescription string
	Resource                    *Resource // For "in body" parameters
	Required                    bool
	IsArray                     bool // "in body" parameter is an array
}

// Response represents an API method response.
type Response struct {
	Description       string
	StatusDescription string
	Resource          *Resource
	Headers           []Header
	IsArray           bool
}

// ResourceOrigin defines different resource origin types.
type ResourceOrigin int

// Resource represents an API resource.
type Resource struct {
	ID                    string
	FQNS                  []string
	Title                 string
	Description           string
	Example               string
	Schema                string
	Type                  []string // Will contain two elements if an array or map [0]=array [1]=What type is in the array
	Properties            map[string]*Resource
	Required              bool
	ReadOnly              bool
	ExcludeFromOperations []string
	Methods               map[string]*Method
	Enum                  []string
	origin                ResourceOrigin
}

// Header represents an API parameter.
type Header struct {
	Name                        string
	Description                 string
	Type                        []string // Will contain two elements if an array [0]=array [1]=What type is in the array
	CollectionFormat            string
	CollectionFormatDescription string
	Default                     string
	Required                    bool
	Enum                        []string
}

// SortMethods implements sortable array of method.
type SortMethods []Method

func (a SortMethods) Len() int           { return len(a) }
func (a SortMethods) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a SortMethods) Less(i, j int) bool { return a[i].SortKey < a[j].SortKey }

// LoadSpecifications loads the provided api specifications.
func LoadSpecifications() error {
	loadStatusCodes()
	loadReplacer()

	if APISuite == nil {
		APISuite = make(map[string]*APISpecification)
		APISuiteGroups = make(map[string][]*APISpecification)
	}

	log().Infof("configured spec filenames: %v", viper.GetStringSlice(config.SpecFilename))

	for _, specLocation := range viper.GetStringSlice(config.SpecFilename) {
		log().Infof("specLocation: %s", specLocation)

		var (
			ok            bool
			specification *APISpecification
		)

		if specification, ok = APISuite[""]; !ok {
			specification = &APISpecification{}
		}

		if err := specification.load(specLocation); err != nil {
			return err
		}

		APISuite[specification.ID] = specification

		if _, exists := APISuiteGroups[specification.GroupBy]; !exists {
			APISuiteGroups[specification.GroupBy] = make([]*APISpecification, 0)
		}

		APISuiteGroups[specification.GroupBy] = append(APISuiteGroups[specification.GroupBy], specification)
	}

	return nil
}

func (c *APISpecification) load(specLocation string) error {
	document, err := loadSpec(normalizeSpecLocation(specLocation))
	if err != nil {
		return err
	}

	apispec := document.Spec()

	if isLocalSpecURL(specLocation) && !strings.HasPrefix(specLocation, "/") {
		specLocation = "/" + specLocation
	}

	c.URL = specLocation

	basePath := apispec.BasePath
	basePathLen := len(basePath)
	// Ignore basepath if it is a single '/'
	if basePathLen == 1 && basePath[0] == '/' {
		basePathLen = 0
	}

	scheme := "http"
	if apispec.Schemes != nil {
		scheme = apispec.Schemes[0]
	}

	host := apispec.Host
	if host == "" {
		host = viper.GetString(config.SpecDefaultHost)
	}

	u := &url.URL{
		Scheme: scheme,
		Host:   host,
	}

	c.APIInfo.Description = string(formatter.Markdown([]byte(apispec.Info.Description)))
	c.APIInfo.Title = apispec.Info.Title

	if c.APIInfo.Title == "" {
		log().Panicf("Error: Specification %s does not have a info.title member.", c.URL)
	}

	log().Tracef("Parse OpenAPI specification %q", c.APIInfo.Title)

	c.ID = titleToKebab(c.APIInfo.Title)

	c.getSecurityDefinitions(apispec)
	c.getDefaultSecurity(apispec)

	c.GroupBy = "default"
	if groupBy, ok := apispec.Extensions[groupByExt].(string); ok {
		c.GroupBy = groupBy
	}

	// Should methods in the navigation be presented by type (GET, POST) or name (string)?
	methodNavByName := false
	if byname, ok := apispec.Extensions[navMethodNameExt].(bool); ok {
		methodNavByName = byname
	}

	var methodSortBy []string

	if sortByList, ok := apispec.Extensions[sortMethodsByExt].([]interface{}); ok {
		for _, sortBy := range sortByList {
			keyname, _ := sortBy.(string)
			if _, ok := sortTypes[keyname]; !ok {
				log().Errorf("Error: Invalid x-sortBy value %s", keyname)
			} else {
				methodSortBy = append(methodSortBy, keyname)
			}
		}
	}

	// Use the top level TAGS to order the API resources/endpoints
	// If Tags: [] is not defined, or empty, then no filtering or ordering takes place,
	// and all API paths will be documented..
	for _, tag := range getTags(apispec) {
		log().Trace("  In tag loop...")
		// Tag matching may not be as expected if multiple paths have the same TAG (which is technically permitted)
		var (
			ok  bool
			api *APIGroup
		)

		groupingByTag := false
		if tag.Name != "" {
			groupingByTag = true
		}

		// Will only populate if Tagging used in spec. processMethod overrides if needed.
		name := tag.Description
		if name == "" {
			name = tag.Name
		}

		log().Tracef("    - %s", name)

		// If we're grouping by TAGs, then build the API at the tag level
		if groupingByTag {
			api = &APIGroup{
				ID:                     titleToKebab(name),
				Name:                   name,
				URL:                    u,
				Info:                   &c.APIInfo,
				MethodNavigationByName: methodNavByName,
				MethodSortBy:           methodSortBy,
				Consumes:               apispec.Consumes,
				Produces:               apispec.Produces,
			}
		}

		for path, pathItem := range document.Analyzer.AllPaths() {
			log().Trace("    In path loop...")

			if isPrivate(pathItem.Extensions) {
				log().Debugf("%s all operations private", basePath+path)

				continue
			}

			if basePathLen > 0 {
				path = basePath + path
			}

			// If not grouping by tag, then build the API at the path level
			if !groupingByTag {
				api = &APIGroup{
					ID:                     titleToKebab(name),
					Name:                   name,
					URL:                    u,
					Info:                   &c.APIInfo,
					MethodNavigationByName: methodNavByName,
					MethodSortBy:           methodSortBy,
					Consumes:               apispec.Consumes,
					Produces:               apispec.Produces,
				}
			}

			var ver string
			if ver, ok = pathItem.Extensions[versionExt].(string); !ok {
				ver = "latest"
			}

			api.CurrentVersion = ver

			pi := pathItem
			c.getMethods(tag, api, &api.Methods, &pi, path, ver) // Current version

			// If API was populated (will not be if tags do not match), add to set
			if !groupingByTag && len(api.Methods) > 0 {
				log().Tracef("    + Adding %s", name)

				sort.Sort(SortMethods(api.Methods))
				c.APIs = append(c.APIs, *api) // All APIs (versioned within)
			}
		}

		if groupingByTag && len(api.Methods) > 0 {
			log().Tracef("    + Adding %s", name)

			sort.Sort(SortMethods(api.Methods))
			c.APIs = append(c.APIs, *api) // All APIs (versioned within)
		}
	}

	// Build a API map, grouping by version
	for _, api := range c.APIs {
		for v := range api.Versions {
			if c.APIVersions == nil {
				c.APIVersions = make(map[string]APISet)
			}
			// Create copy of API and set Methods array to be correct for the version we are building
			napi := api
			napi.Methods = napi.Versions[v]
			napi.Versions = nil
			c.APIVersions[v] = append(c.APIVersions[v], napi) // Group APIs by version
		}
	}

	return nil
}

func (c *APISpecification) getMethods(tag spec.Tag, api *APIGroup, methods *[]Method, pi *spec.PathItem, path, version string) {
	c.getMethod(tag, api, methods, version, pi, pi.Get, path, "get")
	c.getMethod(tag, api, methods, version, pi, pi.Post, path, "post")
	c.getMethod(tag, api, methods, version, pi, pi.Put, path, "put")
	c.getMethod(tag, api, methods, version, pi, pi.Delete, path, "delete")
	c.getMethod(tag, api, methods, version, pi, pi.Head, path, "head")
	c.getMethod(tag, api, methods, version, pi, pi.Options, path, "options")
	c.getMethod(tag, api, methods, version, pi, pi.Patch, path, "patch")
}

func (c *APISpecification) getMethod(tag spec.Tag, api *APIGroup, methods *[]Method, version string, pathitem *spec.PathItem, operation *spec.Operation, path, methodname string) {
	if operation == nil {
		log().Tracef("Skipping %s %s - Operation is nil.", path, methodname)

		return
	}

	if isPrivate(operation.Extensions) {
		log().Debugf("Skipping %s %s - Operation is private", path, methodname)

		return
	}

	log().Tracef("  Operation tag length: %d", len(operation.Tags))

	// Filter and sort by matching current top-level tag with the operation tags.
	// If Tagging is not used by spec, then process each operation without filtering.
	if len(operation.Tags) == 0 {
		if tag.Name != "" {
			log().Tracef("Skipping %s - Operation does not contain a tag member, and tagging is in use.", operation.Summary)

			return
		}

		method := c.processMethod(api, pathitem, operation, path, methodname, version)
		*methods = append(*methods, *method)
	} else {
		log().Trace("    > Check tags")
		for _, t := range operation.Tags {
			log().Tracef("      - Compare tag %q with %q", tag.Name, t)
			if tag.Name == "" || t == tag.Name {
				method := c.processMethod(api, pathitem, operation, path, methodname, version)
				*methods = append(*methods, *method)
			}
		}
	}
}

func (c *APISpecification) getSecurityDefinitions(s *spec.Swagger) {
	if c.SecurityDefinitions == nil {
		c.SecurityDefinitions = make(map[string]SecurityScheme)
	}

	for n, d := range s.SecurityDefinitions {
		stype := d.Type

		def := &SecurityScheme{
			Description:   string(formatter.Markdown([]byte(d.Description))),
			Type:          stype,  // basic, apiKey or oauth2
			ParamName:     d.Name, // name of header to be used if ParamLocation is 'header'
			ParamLocation: d.In,   // Either query or header
		}

		if stype == "apiKey" {
			def.IsAPIKey = true
		}

		if stype == "basic" {
			def.IsBasic = true
		}

		if stype == "oauth2" {
			def.IsOAuth2 = true
			def.OAuth2Flow = d.Flow                   // implicit, password (explicit) application or accessCode
			def.AuthorizationURL = d.AuthorizationURL // Only for implicit or accesscode flow
			def.TokenURL = d.TokenURL                 // Only for implicit, accesscode or password flow
			def.Scopes = make(map[string]string)

			for s, n := range d.Scopes {
				def.Scopes[s] = n
			}
		}

		c.SecurityDefinitions[n] = *def
	}
}

func (c *APISpecification) getDefaultSecurity(s *spec.Swagger) {
	c.DefaultSecurity = make(map[string]Security)
	c.processSecurity(s.Security, c.DefaultSecurity)
}

func (c *APISpecification) processMethod(api *APIGroup, pathItem *spec.PathItem, o *spec.Operation, path, methodname, version string) *Method {
	var (
		opname    string
		gotOpname bool
	)

	operationName := methodname
	if opname, gotOpname = o.Extensions[opNameExt].(string); gotOpname {
		operationName = opname
	}

	// Construct an ID for the Method. Choose from operation ID, x-operationName, summary and lastly method name.
	id := o.ID // OperationID
	if id == "" {
		// No ID, use x-operationName, if we have it...
		if gotOpname {
			id = titleToKebab(opname)
		} else {
			id = titleToKebab(o.Summary) // No opname, use summary
			if id == "" {
				id = methodname // Last chance. Method name.
			}
		}
	}

	navigationName := operationName
	if api.MethodNavigationByName {
		navigationName = o.Summary
	}

	sortkey := api.getMethodSortKey(path, methodname, operationName, navigationName, o.Summary)

	method := &Method{
		ID:             camelToKebab(id),
		Name:           o.Summary,
		Description:    string(formatter.Markdown([]byte(o.Description))),
		Method:         methodname,
		Path:           path,
		Responses:      make(map[int]Response),
		NavigationName: navigationName,
		OperationName:  operationName,
		APIGroup:       api,
		SortKey:        sortkey,
	}

	if len(o.Consumes) > 0 {
		method.Consumes = o.Consumes
	} else {
		method.Consumes = api.Consumes
	}

	if len(o.Produces) > 0 {
		method.Produces = o.Produces
	} else {
		method.Produces = api.Produces
	}

	// If Tagging is not used by spec to select, group and order API paths to document, then
	// complete the missing names.
	// First try the vendor extension x-pathName, falling back to summary if not set.
	// Note, that the APIGroup will get the last pathName set on the path methods added to the group (by tag).
	//
	if pathname, ok := pathItem.Extensions[pathNameExt].(string); ok {
		api.Name = pathname
		api.ID = titleToKebab(api.Name)
	}

	if api.Name == "" {
		name := o.Summary
		if name == "" {
			log().Panicf("Error: Operation %q does not have an operationId or summary member.", id)
		}

		api.Name = name
		api.ID = titleToKebab(name)
	}

	if c.ResourceList == nil {
		c.ResourceList = make(map[string]map[string]*Resource)
	}

	c.processParameters(pathItem.Parameters, method, version)

	c.processParameters(o.Parameters, method, version)

	// Compile resources from response declaration
	if o.Responses == nil {
		log().Panicf("Error: Operation %s %s is missing a responses declaration.", methodname, path)
	}
	// FIXME - Dies if there are no responses...
	for status, response := range o.Responses.StatusCodeResponses {
		log().Tracef("Response for status %d", status)

		// Discover if the resource is already declared, and pick it up
		// if it is (keyed on version number)
		if response.Schema != nil {
			if _, ok := c.ResourceList[version]; !ok {
				c.ResourceList[version] = make(map[string]*Resource)
			}
		}

		r := response
		rsp := c.buildResponse(&r, method, version)
		rsp.StatusDescription = httpStatusDescription(status)
		method.Responses[status] = *rsp
	}

	if o.Responses.Default != nil {
		rsp := c.buildResponse(o.Responses.Default, method, version)
		method.DefaultResponse = rsp
	}

	// If no Security given for operation, then the global defaults are appled.
	method.Security = make(map[string]Security)
	if !c.processSecurity(o.Security, method.Security) {
		method.Security = c.DefaultSecurity
	}

	return method
}

func (c *APISpecification) processParameters(params []spec.Parameter, method *Method, version string) {
	for _, param := range params {
		p := Parameter{
			Name:        param.Name,
			In:          param.In,
			Description: string(formatter.Markdown([]byte(param.Description))),
			Required:    param.Required,
		}
		p.setType(param)
		p.setEnums(param)

		switch strings.ToLower(param.In) {
		case "formdata":
			method.FormParams = append(method.FormParams, p)
		case "path":
			method.PathParams = append(method.PathParams, p)
		case "body":
			if param.Schema == nil {
				log().Panicf("Error: 'in body' parameter %s is missing a schema declaration.", param.Name)
			}

			var body map[string]interface{}
			p.Resource, body, p.IsArray = c.resourceFromSchema(param.Schema, method, nil, true)
			p.Resource.Schema = jsonResourceToString(body, p.IsArray)
			p.Resource.origin = RequestBody
			method.BodyParam = &p
			c.crossLinkMethodAndResource(p.Resource, method, version)
		case "header":
			method.HeaderParams = append(method.HeaderParams, p)
		case "query":
			method.QueryParams = append(method.QueryParams, p)
		}
	}
}

func (c *APISpecification) buildResponse(resp *spec.Response, method *Method, version string) *Response {
	var response *Response

	if resp != nil {
		var (
			vres        *Resource
			r           *Resource
			isArray     bool
			exampleJSON map[string]interface{}
		)

		if resp.Schema != nil {
			r, exampleJSON, isArray = c.resourceFromSchema(resp.Schema, method, nil, false)

			if r != nil {
				r.Schema = jsonResourceToString(exampleJSON, false)
				r.origin = MethodResponse
				vres = c.crossLinkMethodAndResource(r, method, version)
			}
		}

		response = &Response{
			Description: string(formatter.Markdown([]byte(resp.Description))),
			Resource:    vres,
			IsArray:     isArray,
		}
		method.Resources = append(method.Resources, response.Resource) // Add the resource to the method which uses it

		response.compileHeaders(resp)
	}

	return response
}

func (c *APISpecification) crossLinkMethodAndResource(resource *Resource, method *Method, version string) *Resource {
	log().Tracef("++ Resource version %s  ID %s", version, resource.ID)

	if _, ok := c.ResourceList[version]; !ok {
		c.ResourceList[version] = make(map[string]*Resource)
	}

	// Look for a pre-declared resource with the response ID, and use that or create the first one...
	var (
		resFound bool
		vres     *Resource
	)

	if vres, resFound = c.ResourceList[version][resource.ID]; !resFound {
		log().Trace("   - Creating new resource")

		vres = resource
	}

	// Add to the compiled list of methods which use this resource.
	if vres.Methods == nil {
		vres.Methods = make(map[string]*Method)
	}

	vres.Methods[method.ID] = method // Use a map to collapse duplicates.

	// Store resource in resource-list of the specification, considering precident.
	if resource.origin == RequestBody {
		// Resource is a Request Body - the lowest precident
		log().Trace("   - Resource origin is a request body")

		// If this is the first time the resource has been seen, it's okay to store this in
		// the global list. A request body resource is a filtered (excludes read-only) resource,
		// and has a lower precident than a response resource.
		if !resFound {
			log().Trace("     - Not seen before, so storing in global list")

			c.ResourceList[version][resource.ID] = vres
		}
	} else {
		log().Trace("   - Resource origin is a response, so storing in global list")

		// This is a response resource (which has the highest precident). If an existing
		// request-body resource was found in the cache, then it is replaced by the
		// response resource (but maintaining the method list associated with the resource).
		if resFound && vres.origin == RequestBody {
			resource.Methods = vres.Methods
			vres = resource
		}

		c.ResourceList[version][resource.ID] = vres // If we've already got the resource, this does nothing
	}

	return vres
}

func (c *APISpecification) processSecurity(s []map[string][]string, security map[string]Security) bool {
	count := 0

	for _, sec := range s {
		for n, scopes := range sec {
			// Lookup security name in definitions
			if scheme, ok := c.SecurityDefinitions[n]; ok {
				count++

				// Add security
				security[scheme.Type] = Security{
					Scheme: &scheme,
					Scopes: make(map[string]string),
				}

				if scheme.IsOAuth2 {
					// Populate method specific scopes by cross referencing SecurityDefinitions
					for _, scope := range scopes {
						if scopeDesc, ok := scheme.Scopes[scope]; ok {
							security[scheme.Type].Scopes[scope] = scopeDesc
						}
					}
				}
			}
		}
	}

	return count != 0
}

func (c *APISpecification) resourceFromSchema(s *spec.Schema, method *Method, fqNS []string, isRequestResource bool) (*Resource, map[string]interface{}, bool) {
	if s == nil {
		return nil, nil, false
	}

	stype := checkPropertyType(s)
	log().Tracef("resourceFromSchema: Schema type: %s", stype)
	log().Tracef("FQNS: %s", fqNS)
	log().Trace("CHECK schema type and items")

	// It is possible for a response to be an array of
	//     objects, and it it possible to declare this in several ways:
	// 1. As :
	//      "schema": {
	//        "$ref": "model"
	//      }
	//      Where the model declares itself of type array (of objects)
	// 2. Or :
	//    "schema": {
	//        "type": "array",
	//        "items": {
	//            "$ref": "model"
	//        }
	//    }
	//
	//  In the second version, "items" points to a schema. So what we have done to align these
	//  two cases is to keep the top level "type" in the second case, and apply it to items.schema.Type,
	//  reseting our schema variable to items.schema.

	if s.Type == nil {
		s.Type = append(s.Type, "object")
	}

	originalS := s

	if s.Items != nil {
		stringorarray := s.Type

		// Jump to nearest schema for items, depending on how it was declared
		if s.Items.Schema != nil { // API Spec - items: { properties: {} }
			s = s.Items.Schema
			log().Tracef("got s.Items.Schema for %s", s.Title)
		} else { // API Spec - items: { $ref: "" }
			s = &s.Items.Schemas[0]
			log().Tracef("got s.Items.Schemas[0] for %s", s.Title)
		}

		if s.Type == nil {
			log().Tracef("Got array of objects or object. Name %s", s.Title)
			s.Type = stringorarray // Put back original type
		} else if s.Type.Contains(arrayType) {
			log().Tracef("Got array for %s", s.Title)
			s.Type = stringorarray // Put back original type
		} else if stringorarray.Contains(arrayType) && len(s.Properties) == 0 {
			// if we get here then we can assume the type is supposed to be an array of primitives
			// Store the actual primitive type in the second element of the Type array.
			s.Type = spec.StringOrArray([]string{arrayType, s.Type[0]})
		} else {
			s.Type = stringorarray // Put back original type
			log().Trace("putting s.Type back")
		}

		log().Tracef("REMAP SCHEMA (Type is now %s)", s.Type)
	}

	if len(s.Format) > 0 {
		s.Type[len(s.Type)-1] = s.Format
	}

	id := titleToKebab(s.Title)

	if len(fqNS) == 0 && id == "" {
		log().Panicf("Error: %s %s references a model definition that does not have a title member.", strings.ToUpper(method.Method), method.Path)
	}

	// Ignore ID (from title element) for all but child-objects...
	// This prevents the title-derived ID being added onto the end of the FQNS.property as
	// FQNS.property.ID, if title is given for the property in the spec.
	if len(fqNS) > 0 && !s.Type.Contains("object") {
		id = ""
	}

	var isArray bool

	if strings.EqualFold(s.Type[0], arrayType) {
		fqNSlen := len(fqNS)
		if fqNSlen > 0 {
			fqNS = append(fqNS[0:fqNSlen-1], fqNS[fqNSlen-1]+"[]")
		}

		isArray = true
	}

	myFQNS := fqNS

	var chopped bool

	if id == "" && len(myFQNS) > 0 {
		id = myFQNS[len(myFQNS)-1]
		myFQNS = append([]string{}, myFQNS[0:len(myFQNS)-1]...)
		chopped = true

		log().Tracef("Chopped %s from myFQNS leaving %s", id, myFQNS)
	}

	resourceFQNS := myFQNS
	// If we are dealing with an object, then adjust the resource FQNS and id
	// so that the last element of the FQNS is chopped off and used as the ID
	if !chopped && s.Type.Contains("object") {
		if len(resourceFQNS) > 0 {
			id = resourceFQNS[len(resourceFQNS)-1]
			resourceFQNS = resourceFQNS[:len(resourceFQNS)-1]

			log().Tracef("Got an object, so slicing %s from resourceFQNS leaving %s", id, myFQNS)
		}
	}

	// If there is no description... the case where we have an array of objects. See issue/11
	var description string
	if originalS.Description != "" {
		description = string(formatter.Markdown([]byte(originalS.Description)))
	} else {
		description = originalS.Title
	}

	log().Tracef("Create resource %s [%s]", id, s.Title)

	if isArray {
		log().Trace("- Is Arrays")
	}

	r := &Resource{
		ID:          id,
		Title:       s.Title,
		Description: description,
		Type:        s.Type,
		Properties:  make(map[string]*Resource),
		FQNS:        resourceFQNS,
	}

	if s.Example != nil {
		example, err := jsonMarshalIndent(&s.Example)
		if err != nil {
			log().Errorf("Error encoding example json: %s", err)
		}

		r.Example = string(example)
	}

	if len(s.Enum) > 0 {
		for _, e := range s.Enum {
			r.Enum = append(r.Enum, fmt.Sprintf("%s", e))
		}
	}

	r.ReadOnly = originalS.ReadOnly

	if ops, ok := originalS.Extensions[excludeOpExt].([]interface{}); ok && isRequestResource {
		// Mark resource property as being excluded from operations with this name.
		// This filtering only takes effect in a request body, just like readOnly, so when isRequestResource is true
		for _, op := range ops {
			if c, ok := op.(string); ok {
				r.ExcludeFromOperations = append(r.ExcludeFromOperations, c)
			}
		}
	}

	required := make(map[string]bool)
	jsonRepresentation := make(map[string]interface{})

	log().Trace("Call compileproperties...")
	c.compileproperties(s, r, method, id, required, jsonRepresentation, myFQNS, chopped, isRequestResource)

	for allof := range s.AllOf {
		c.compileproperties(&s.AllOf[allof], r, method, id, required, jsonRepresentation, myFQNS, chopped, isRequestResource)
	}

	log().Trace("resourceFromSchema done")

	return r, jsonRepresentation, isArray
}

// Takes a Schema object and adds properties to the Resource object.
// It uses the 'required' map to set when properties are required and builds a JSON
// representation of the resource.
func (c *APISpecification) compileproperties(s *spec.Schema, r *Resource, method *Method, id string, required map[string]bool, jsonRep map[string]interface{}, myFQNS []string,
	chopped, isRequestResource bool) {
	// First, grab the required members
	for _, n := range s.Required {
		required[n] = true
	}

	for name, property := range s.Properties {
		p := property
		c.processProperty(&p, name, r, method, id, required, jsonRep, myFQNS, chopped, isRequestResource)
	}

	// Special case to deal with AdditionalProperties (which really just boils down to declaring a
	// map of 'type' (string, int, object etc).
	if s.AdditionalProperties != nil && s.AdditionalProperties.Allows {
		name := "<key>"
		ap := s.AdditionalProperties.Schema

		if len(ap.Type) == 0 {
			ap.Type = spec.StringOrArray([]string{"map", "object"})
		} else {
			ap.Type = spec.StringOrArray([]string{"map", ap.Type[0]}) // massage type so that it is a map of 'type'
		}

		c.processProperty(ap, name, r, method, id, required, jsonRep, myFQNS, chopped, isRequestResource)
	}
}

func (c *APISpecification) processProperty(s *spec.Schema, name string, r *Resource, method *Method, id string, required map[string]bool, jsonRep map[string]interface{}, myFQNS []string,
	chopped, isRequestResource bool) {
	newFQNS := prepareNamespace(myFQNS, id, name, chopped)

	var (
		jsonResource map[string]interface{}
		resource     *Resource
	)

	log().Tracef("A call resourceFromSchema for property %s", name)

	resource, jsonResource, _ = c.resourceFromSchema(s, method, newFQNS, isRequestResource)

	skip := isRequestResource && resource.ReadOnly
	if !skip && resource.ExcludeFromOperations != nil {
		log().Tracef("Exclude [%s] in operation [%s] if in list: %s", name, method.OperationName, resource.ExcludeFromOperations)

		for _, opname := range resource.ExcludeFromOperations {
			if opname == method.OperationName {
				log().Tracef("[%s] is excluded", name)

				skip = true

				break
			}
		}
	}

	if skip {
		return
	}

	r.Properties[name] = resource
	jsonRep[name] = jsonResource

	if _, ok := required[name]; ok {
		r.Properties[name].Required = true
	}

	log().Tracef("resource property %s type: %s", name, r.Properties[name].Type[0])

	if !strings.EqualFold(r.Properties[name].Type[0], "object") {
		// Arrays of objects need to be handled as a special case
		if strings.EqualFold(r.Properties[name].Type[0], arrayType) {
			log().Tracef("Processing an array property %s", name)

			if s.Items != nil {
				if s.Items.Schema != nil {
					// Some outputs (example schema, member description) are generated differently
					// if the array member references an object or a primitive type
					r.Properties[name].Description = string(formatter.Markdown([]byte(s.Description)))

					// If here, we have no jsonResource returned from resourceFromSchema, then the property
					// is an array of primitive, so construct either an array of string or array of object
					// as appropriate.
					if len(jsonResource) > 0 {
						var arrayObj []map[string]interface{}
						arrayObj = append(arrayObj, jsonResource)
						jsonRep[name] = arrayObj
					} else {
						var arrayObj []string
						// We stored the real type of the primitive in Type array index 1 (see the note in
						// resourceFromSchema). There is a special case of an array of object where EVERY
						// member of the object is read-only and filtered out due to isRequestResource being true.
						// In this case, we will fall into this section of code, so we must check the length
						// of the .Type array, as array len will be 1 [0] in this case, and 2 [1] for an array of
						// primitives case.
						// In the case where object members are readonly, the JSON produced will have a
						// value of nil. This shouldn't happen often, as a more correct spec will declare the
						// array member as readOnly!
						//
						if len(r.Properties[name].Type) > 1 {
							// Got an array of primitives
							arrayObj = append(arrayObj, r.Properties[name].Type[1])
						}

						jsonRep[name] = arrayObj
					}
				} else { // array and property.Items.Schema is NIL
					var arrayObj []map[string]interface{}
					arrayObj = append(arrayObj, jsonResource)
					jsonRep[name] = arrayObj
				}
			} else { // array and Items are nil
				var arrayObj []map[string]interface{}
				arrayObj = append(arrayObj, jsonResource)
				jsonRep[name] = arrayObj
			}
		} else if strings.EqualFold(r.Properties[name].Type[0], "map") { // not array, so a map?
			if strings.EqualFold(r.Properties[name].Type[1], "object") {
				jsonRep[name] = jsonResource // A map of objects
			} else {
				jsonRep[name] = r.Properties[name].Type[1] // map of primitive
			}
		} else {
			// We're NOT an array, map or object, so a primitive
			jsonRep[name] = r.Properties[name].Type[0]
		}
	} else {
		// We're an object
		jsonRep[name] = jsonResource
	}
}

func (p *Parameter) setType(src spec.Parameter) {
	if src.Type == arrayType {
		if src.CollectionFormat == "" {
			src.CollectionFormat = "csv"
		}

		p.Type = append(p.Type, src.Type)
		p.CollectionFormat = src.CollectionFormat
		p.CollectionFormatDescription = collectionFormatDescription(src.CollectionFormat)
	}

	var (
		ptype  string
		format string
	)

	if src.Type == arrayType {
		ptype = src.Items.Type
		format = src.Items.Format
	} else {
		ptype = src.Type
		format = src.Format
	}

	if format != "" {
		ptype = format
	}

	p.Type = append(p.Type, ptype)
}

func (p *Parameter) setEnums(src spec.Parameter) {
	var ea []interface{}

	if src.Type == arrayType {
		ea = src.Items.Enum
	} else {
		ea = src.Enum
	}

	es := make([]string, 0)
	for _, e := range ea {
		es = append(es, fmt.Sprintf("%s", e))
	}

	p.Enum = es
}

func (r *Response) compileHeaders(sr *spec.Response) {
	if sr.Headers == nil {
		return
	}

	for name, params := range sr.Headers {
		header := &Header{
			Description: string(formatter.Markdown([]byte(params.Description))),
			Name:        name,
		}

		htype := getType(params)

		if params.Type == arrayType {
			if params.CollectionFormat == "" {
				params.CollectionFormat = "csv"
			}

			header.Type = append(header.Type, params.Type)
			header.CollectionFormat = params.CollectionFormat
			header.CollectionFormatDescription = collectionFormatDescription(params.CollectionFormat)
		}

		format := getFormat(params)
		if format != "" {
			htype = format
		}

		header.Type = append(header.Type, htype)
		header.Enum = getEnums(params)

		r.Headers = append(r.Headers, *header)
	}
}

func (api *APIGroup) getMethodSortKey(path, method, operation, navigation, summary string) string {
	// Handle a list of sort-by values, so that ordering can be fixed.
	// Sorting by path alone does not work because ordering changes around GET/POST/PUT Etc
	var key string

	for _, sortby := range api.MethodSortBy {
		switch sortby {
		case "path":
			key += path
		case "method":
			key += method
		case "operation":
			key += operation
		case "navigation":
			key += navigation
		case "summary":
			key += summary
		}

		key += "~"
	}

	if key == "" {
		key = summary
	}

	return key
}

func getTags(specification *spec.Swagger) []spec.Tag {
	tags := make([]spec.Tag, 0)
	tags = append(tags, specification.Tags...)

	if len(tags) == 0 {
		tags = append(tags, spec.Tag{})
	}

	return tags
}

func jsonResourceToString(jsonres map[string]interface{}, isArray bool) string {
	// If the resource is an array, then append json object to outer array, else serialize the object.
	var example []byte

	if isArray {
		var arrayObj []map[string]interface{}
		arrayObj = append(arrayObj, jsonres)
		example, _ = jsonMarshalIndent(arrayObj)
	} else {
		example, _ = jsonMarshalIndent(jsonres)
	}

	return string(example)
}

func checkPropertyType(s *spec.Schema) string {
	/*
	   (string) (len=12) "string_array": (spec.Schema) {
	    SchemaProps: (spec.SchemaProps) {
	     Description: (string) (len=16) "Array of strings",
	     Type: (spec.StringOrArray) (len=1 cap=1) { (string) (len=5) "array" },
	     Items: (*spec.SchemaOrArray)(0xc8205bb000)({
	      Schema: (*spec.Schema)(0xc820202480)({
	       SchemaProps: (spec.SchemaProps) {
	        Type: (spec.StringOrArray) (len=1 cap=1) { (string) (len=6) "string" },
	       },
	      }),
	     }),
	    },
	   }
	*/
	ptype := "primitive"

	if s.Type == nil {
		ptype = "object"
	}

	sOrig := s.Type

	if s.Items != nil {
		if s.Type.Contains(arrayType) {
			if s.Items.Schema != nil {
				s = s.Items.Schema
			} else {
				s = &s.Items.Schemas[0] // - Main schema [1] = Additional properties? See online swagger editior.
			}

			if s.Type == nil {
				ptype = "array of objects"
				if s.SchemaProps.Type != nil {
					ptype = "array of SOMETHING"
				}
			} else if s.Type.Contains(arrayType) {
				ptype = "array of primitives"
			} else {
				ptype = fmt.Sprintf("%s", sOrig)
			}
		} else {
			ptype = "Some object"
		}
	}

	return ptype
}

func prepareNamespace(myFQNS []string, id, name string, chopped bool) []string {
	newFQNS := append([]string{}, myFQNS...) // create slice

	if chopped && id != "" {
		log().Tracef("Append ID onto newFQNZ %s + %q", newFQNS, id)
		newFQNS = append(newFQNS, id)
	}

	newFQNS = append(newFQNS, name)

	return newFQNS
}

// titleToKebab convert a Title string to kebab.
func titleToKebab(s string) string {
	return strings.ReplaceAll(
		string(kababExclude.ReplaceAll([]byte(strings.ToLower(s)), []byte(""))),
		" ", "-")
}

// camelToKebab converts camel case to kebab.
func camelToKebab(s string) string {
	return strings.ReplaceAll(snaker.CamelToSnake(s), "_", "-")
}

func loadSpec(location string) (*loads.Document, error) {
	log().Infof("Importing OpenAPI specifications from %s", location)

	raw, err := swag.LoadFromFileOrHTTP(location)
	if err != nil {
		log().Errorf("Error: go-openapi/swag failed to load spec [%s]: %s", location, err)

		return nil, err
	}

	document, err := loads.Analyzed(json.RawMessage(replace(raw)), "")
	if err != nil {
		log().Errorf("Error: go-openapi/loads failed to analyze spec: %s", err)

		return nil, err
	}

	document, err = document.Expanded()
	if err != nil {
		log().Errorf("Error: go-openapi/loads failed to expand spec: %s", err)
	}

	return document, err
}

// jsonMarshalIndent Wrapper around MarshalIndent to prevent < > & from being escaped.
func jsonMarshalIndent(v interface{}) ([]byte, error) {
	b, err := json.MarshalIndent(v, "", "    ")

	b = bytes.ReplaceAll(b, []byte("\\u003c"), []byte("<"))
	b = bytes.ReplaceAll(b, []byte("\\u003e"), []byte(">"))
	b = bytes.ReplaceAll(b, []byte("\\u0026"), []byte("&"))

	return b, err
}

func isLocalSpecURL(specURL string) bool {
	match, err := regexp.MatchString("(?i)^https?://.+", specURL)
	if err != nil {
		log().Panicf("Attempted to match against an invalid regexp: %s", err)
	}

	return !match
}

func normalizeSpecLocation(specLocation string) string {
	if isLocalSpecURL(specLocation) {
		log().Debugf("SpecDir = %s", viper.GetString(config.SpecDir))

		base, err := filepath.Abs(viper.GetString(config.SpecDir))
		if err != nil {
			log().Errorf("Error forming specification path: %s", err)
		}

		base = filepath.ToSlash(base)
		log().Debugf("SpecDir (base) = %s", base)

		return filepath.Join(base, specLocation)
	}

	return specLocation
}

// OpenAPI/Swagger/go-openAPI define a Header object and an Items object. A
// Header _can_ be an Items object, if it is an array. Annoyingly, a Header
// object is the same as Items but with an additional Description member.
// It would have been nice to treat Header.Items as though it were Header in
// the case of an array...
// Solve both problems by defining accessor methods that will do the "right thing"
// in the case of an array.
func getType(h spec.Header) string {
	if h.Type == arrayType {
		return h.Items.Type
	}

	return h.Type
}

func getFormat(h spec.Header) string {
	if h.Type == arrayType {
		return h.Items.Format
	}

	return h.Format
}

func getEnums(h spec.Header) []string {
	var ea []interface{}

	if h.Type == arrayType {
		ea = h.Items.Enum
	} else {
		ea = h.Enum
	}

	es := make([]string, 0)
	for _, e := range ea {
		es = append(es, fmt.Sprintf("%s", e))
	}

	return es
}

func isPrivate(exts spec.Extensions) bool {
	if pv, ok := exts.GetString(visibilityExt); ok {
		return pv == "private"
	}

	return false
}

func collectionFormatDescription(format string) string {
	return collectionTable[format]
}
