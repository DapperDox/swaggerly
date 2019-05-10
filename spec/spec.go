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
package spec

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"regexp"
	"sort"
	"strings"

	"dapperdox/config"
	"dapperdox/logger"
	//"github.com/davecgh/go-spew/spew"
	"github.com/go-openapi/loads"
	"github.com/go-openapi/spec"
	"github.com/serenize/snaker"
	"github.com/shurcooL/github_flavored_markdown"
)

type APISpecification struct {
	ID      string
	APIs    APISet // APIs represents the parsed APIs
	APIInfo Info
	URL     string

	SecurityDefinitions map[string]SecurityScheme
	DefaultSecurity     map[string]Security
	ResourceList        map[string]map[string]*Resource // Version->ResourceName->Resource
	APIVersions         map[string]APISet               // Version->APISet
}

var APISuite map[string]*APISpecification

// GetByName returns an API by name
func (c *APISpecification) GetByName(name string) *APIGroup {
	for _, a := range c.APIs {
		if a.Name == name {
			return &a
		}
	}
	return nil
}

// GetByID returns an API by ID
func (c *APISpecification) GetByID(id string) *APIGroup {
	for _, a := range c.APIs {
		if a.ID == id {
			return &a
		}
	}
	return nil
}

type APISet []APIGroup

type Info struct {
	Title       string
	Description string
}

// APIGroup parents all grouped API methods (Grouping controlled by tagging, if used, or by method path otherwise)
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

type Version struct {
	Version string
	Methods []Method
}

type OAuth2Scheme struct {
	OAuth2Flow       string
	AuthorizationUrl string
	TokenUrl         string
	Scopes           map[string]string
}

type SecurityScheme struct {
	IsApiKey      bool
	IsBasic       bool
	IsOAuth2      bool
	Type          string
	Description   string
	ParamName     string
	ParamLocation string
	OAuth2Scheme
}

type Security struct {
	Scheme *SecurityScheme
	Scopes map[string]string
}

// Method represents an API method
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
	DefaultResponse *Response // A ptr to allow of easy checking of its existance in templates
	Resources       []*Resource
	Security        map[string]Security
	APIGroup        *APIGroup
	SortKey         string
}

// Parameter represents an API method parameter
type Parameter struct {
	Name                        string
	Description                 string
	In                          string
	CollectionFormat            string
	CollectionFormatDescription string
	Required                    bool
	Type                        []string
	Enum                        []string
	Resource                    *Resource // For "in body" parameters
	IsArray                     bool      // "in body" parameter is an array
}

// Response represents an API method response
type Response struct {
	Description       string
	StatusDescription string
	Resource          *Resource
	Headers           []Header
	IsArray           bool
}

type ResourceOrigin int

const (
	RequestBody ResourceOrigin = iota
	MethodResponse
)

// Resource represents an API resource
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

// -----------------------------------------------------------------------------

var sortTypes = map[string]bool{
	"path":       true,
	"method":     true,
	"operation":  true,
	"navigation": true,
	"summary":    true,
}

type SortMethods []Method

func (a SortMethods) Len() int           { return len(a) }
func (a SortMethods) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a SortMethods) Less(i, j int) bool { return a[i].SortKey < a[j].SortKey }

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

// -----------------------------------------------------------------------------
// -----------------------------------------------------------------------------
// -----------------------------------------------------------------------------

func LoadSpecifications(specHost string, collapse bool) error {

	if APISuite == nil {
		APISuite = make(map[string]*APISpecification)
	}

	cfg, err := config.Get()
	if err != nil {
		logger.Errorf(nil, "error configuring app: %s", err)
		return err
	}

	if strings.HasPrefix(specHost, "0.0.0.0") {
		splithost := strings.Split(specHost, ":")
		splithost[0] = "127.0.0.1"
		specHost = strings.Join(splithost, ":")
		logger.Tracef(nil, "Serving specifications from %s\n", specHost)
	}

	for _, specLocation := range cfg.SpecFilename {

		var ok bool
		var specification *APISpecification

		if specification, ok = APISuite[""]; !ok || !collapse {
			specification = &APISpecification{}
		}

		err = specification.Load(specLocation, specHost)
		if err != nil {
			return err
		}

		if collapse {
			//specification.ID = "api"
		}

		APISuite[specification.ID] = specification
	}

	return nil
}

// -----------------------------------------------------------------------------
// Load loads API specs from the supplied host (usually local!)
func (c *APISpecification) Load(specLocation string, specHost string) error {

	if isLocalSpecUrl(specLocation) && !strings.HasPrefix(specLocation, "/") {
		specLocation = "/" + specLocation
	}

	c.URL = specLocation

	document, err := loadSpec(normalizeSpecLocation(specLocation, specHost))
	if err != nil {
		return err
	}
	apispec := document.Spec()

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

	u, err := url.Parse(scheme + "://" + apispec.Host)
	if err != nil {
		return err
	}

	c.APIInfo.Description = string(github_flavored_markdown.Markdown([]byte(apispec.Info.Description)))
	c.APIInfo.Title = apispec.Info.Title

	if len(c.APIInfo.Title) == 0 {
		logger.Errorf(nil, "Error: Specification %s does not have a info.title member.\n", c.URL)
		os.Exit(1)
	}

	logger.Tracef(nil, "Parse OpenAPI specification '%s'\n", c.APIInfo.Title)

	c.ID = TitleToKebab(c.APIInfo.Title)

	c.getSecurityDefinitions(apispec)
	c.getDefaultSecurity(apispec)

	methodNavByName := false // Should methods in the navigation be presented by type (GET, POST) or name (string)?
	if byname, ok := apispec.Extensions["x-navigateMethodsByName"].(bool); ok {
		methodNavByName = byname
	}

	var methodSortBy []string
	if sortByList, ok := apispec.Extensions["x-sortMethodsBy"].([]interface{}); ok {
		for _, sortBy := range sortByList {
			keyname := sortBy.(string)
			if _, ok := sortTypes[keyname]; !ok {
				logger.Errorf(nil, "Error: Invalid x-sortBy value %s\n", keyname)
			} else {
				methodSortBy = append(methodSortBy, keyname)
			}
		}
	}

	//logger.Printf(nil, "DUMP OF ENTIRE SWAGGER SPEC\n")
	//spew.Dump(document)

	// Use the top level TAGS to order the API resources/endpoints
	// If Tags: [] is not defined, or empty, then no filtering or ordering takes place,
	// and all API paths will be documented..
	for _, tag := range getTags(apispec) {
		logger.Tracef(nil, "  In tag loop...\n")
		// Tag matching may not be as expected if multiple paths have the same TAG (which is technically permitted)
		var ok bool

		var api *APIGroup
		groupingByTag := false

		if tag.Name != "" {
			groupingByTag = true
		}

		var name string // Will only populate if Tagging used in spec. processMethod overrides if needed.
		name = tag.Description
		if name == "" {
			name = tag.Name
		}
		logger.Tracef(nil, "    - %s\n", name)

		// If we're grouping by TAGs, then build the API at the tag level
		if groupingByTag {
			api = &APIGroup{
				ID:   TitleToKebab(name),
				Name: name,
				URL:  u,
				Info: &c.APIInfo,
				MethodNavigationByName: methodNavByName,
				MethodSortBy:           methodSortBy,
				Consumes:               apispec.Consumes,
				Produces:               apispec.Produces,
			}
		}

		for path, pathItem := range document.Analyzer.AllPaths() {
			logger.Tracef(nil, "    In path loop...\n")

			if basePathLen > 0 {
				path = basePath + path
			}

			// If not grouping by tag, then build the API at the path level
			if !groupingByTag {
				api = &APIGroup{
					ID:   TitleToKebab(name),
					Name: name,
					URL:  u,
					Info: &c.APIInfo,
					MethodNavigationByName: methodNavByName,
					MethodSortBy:           methodSortBy,
					Consumes:               apispec.Consumes,
					Produces:               apispec.Produces,
				}
			}

			var ver string
			if ver, ok = pathItem.Extensions["x-version"].(string); !ok {
				ver = "latest"
			}
			api.CurrentVersion = ver

			c.getMethods(tag, api, &api.Methods, &pathItem, path, ver) // Current version
			//c.getVersions(tag, api, pathItem.Versions, path)           // All versions

			// If API was populated (will not be if tags do not match), add to set
			if !groupingByTag && len(api.Methods) > 0 {
				logger.Tracef(nil, "    + Adding %s\n", name)

				sort.Sort(SortMethods(api.Methods))
				c.APIs = append(c.APIs, *api) // All APIs (versioned within)
			}
		}

		if groupingByTag && len(api.Methods) > 0 {
			logger.Tracef(nil, "    + Adding %s\n", name)

			sort.Sort(SortMethods(api.Methods))
			c.APIs = append(c.APIs, *api) // All APIs (versioned within)
		}
	}

	// Build a API map, grouping by version
	for _, api := range c.APIs {
		for v, _ := range api.Versions {
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

// -----------------------------------------------------------------------------

func getTags(specification *spec.Swagger) []spec.Tag {
	var tags []spec.Tag

	for _, tag := range specification.Tags {
		tags = append(tags, tag)
	}
	if len(tags) == 0 {
		tags = append(tags, spec.Tag{})
	}
	return tags
}

// -----------------------------------------------------------------------------

func (c *APISpecification) getVersions(tag spec.Tag, api *APIGroup, versions map[string]spec.PathItem, path string) {
	if versions == nil {
		return
	}
	api.Versions = make(map[string][]Method)

	for v, pi := range versions {
		logger.Tracef(nil, "Process version %s\n", v)
		var method []Method
		c.getMethods(tag, api, &method, &pi, path, v)
		api.Versions[v] = method
	}
}

// -----------------------------------------------------------------------------

func (c *APISpecification) getMethods(tag spec.Tag, api *APIGroup, methods *[]Method, pi *spec.PathItem, path string, version string) {

	c.getMethod(tag, api, methods, version, pi, pi.Get, path, "get")
	c.getMethod(tag, api, methods, version, pi, pi.Post, path, "post")
	c.getMethod(tag, api, methods, version, pi, pi.Put, path, "put")
	c.getMethod(tag, api, methods, version, pi, pi.Delete, path, "delete")
	c.getMethod(tag, api, methods, version, pi, pi.Head, path, "head")
	c.getMethod(tag, api, methods, version, pi, pi.Options, path, "options")
	c.getMethod(tag, api, methods, version, pi, pi.Patch, path, "patch")
}

// -----------------------------------------------------------------------------

func (c *APISpecification) getMethod(tag spec.Tag, api *APIGroup, methods *[]Method, version string, pathitem *spec.PathItem, operation *spec.Operation, path, methodname string) {
	if operation == nil {
		logger.Tracef(nil, "Skipping %s %s - Operation is nil.", path, methodname)
		return
	}
	// Filter and sort by matching current top-level tag with the operation tags.
	// If Tagging is not used by spec, then process each operation without filtering.
	taglen := len(operation.Tags)
	logger.Tracef(nil, "  Operation tag length: %d", taglen)
	if taglen == 0 {
		if tag.Name != "" {
			logger.Tracef(nil, "Skipping %s - Operation does not contain a tag member, and tagging is in use.", operation.Summary)
			return
		}
		method := c.processMethod(api, pathitem, operation, path, methodname, version)
		*methods = append(*methods, *method)
	} else {
		logger.Tracef(nil, "    > Check tags")
		for _, t := range operation.Tags {
			logger.Tracef(nil, "      - Compare tag '%s' with '%s'\n", tag.Name, t)
			if tag.Name == "" || t == tag.Name {
				method := c.processMethod(api, pathitem, operation, path, methodname, version)
				*methods = append(*methods, *method)
			}
		}
	}
}

// -----------------------------------------------------------------------------

func (c *APISpecification) getSecurityDefinitions(spec *spec.Swagger) {

	if c.SecurityDefinitions == nil {
		c.SecurityDefinitions = make(map[string]SecurityScheme)
	}

	for n, d := range spec.SecurityDefinitions {
		stype := d.Type

		def := &SecurityScheme{
			Description:   string(github_flavored_markdown.Markdown([]byte(d.Description))),
			Type:          stype,  // basic, apiKey or oauth2
			ParamName:     d.Name, // name of header to be used if ParamLocation is 'header'
			ParamLocation: d.In,   // Either query or header
		}

		if stype == "apiKey" {
			def.IsApiKey = true
		}
		if stype == "basic" {
			def.IsBasic = true
		}
		if stype == "oauth2" {
			def.IsOAuth2 = true
			def.OAuth2Flow = d.Flow                   // implicit, password (explicit) application or accessCode
			def.AuthorizationUrl = d.AuthorizationURL // Only for implicit or accesscode flow
			def.TokenUrl = d.TokenURL                 // Only for implicit, accesscode or password flow
			def.Scopes = make(map[string]string)
			for s, n := range d.Scopes {
				def.Scopes[s] = n
			}
		}

		c.SecurityDefinitions[n] = *def
	}
}

// -----------------------------------------------------------------------------

func (c *APISpecification) getDefaultSecurity(spec *spec.Swagger) {
	c.DefaultSecurity = make(map[string]Security)
	c.processSecurity(spec.Security, c.DefaultSecurity)
}

// -----------------------------------------------------------------------------
func (p *Parameter) setType(src spec.Parameter) {
	if src.Type == "array" {
		if len(src.CollectionFormat) == 0 {
			logger.Errorf(nil, "Error: Request parameter %s is an array without declaring the collectionFormat.\n", src.Name)
			os.Exit(1)
		}
		p.Type = append(p.Type, src.Type)
		p.CollectionFormat = src.CollectionFormat
		p.CollectionFormatDescription = collectionFormatDescription(src.CollectionFormat)
	}
	var ptype string
	var format string

	if src.Type == "array" {
		ptype = src.Items.Type
		format = src.Items.Format
	} else {
		ptype = src.Type
		format = src.Format
	}
	if len(format) > 0 {
		ptype = format
	}
	p.Type = append(p.Type, ptype)
}

func (p *Parameter) setEnums(src spec.Parameter) {
	var ea []interface{}
	if src.Type == "array" {
		ea = src.Items.Enum
	} else {
		ea = src.Enum
	}
	var es = make([]string, 0)
	for _, e := range ea {
		es = append(es, fmt.Sprintf("%s", e))
	}
	p.Enum = es
}

// -----------------------------------------------------------------------------

func (c *APISpecification) processMethod(api *APIGroup, pathItem *spec.PathItem, o *spec.Operation, path, methodname string, version string) *Method {

	var opname string
	var gotOpname bool

	operationName := methodname
	if opname, gotOpname = o.Extensions["x-operationName"].(string); gotOpname {
		operationName = opname
	}

	// Construct an ID for the Method. Choose from operation ID, x-operationName, summary and lastly method name.
	id := o.ID // OperationID
	if id == "" {
		// No ID, use x-operationName, if we have it...
		if gotOpname {
			id = TitleToKebab(opname)
		} else {
			id = TitleToKebab(o.Summary) // No opname, use summary
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
		ID:             CamelToKebab(id),
		Name:           o.Summary,
		Description:    string(github_flavored_markdown.Markdown([]byte(o.Description))),
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
	// XXX Note, that the APIGroup will get the last pathName set on the path methods added to the group (by tag).
	//
	if pathname, ok := pathItem.Extensions["x-pathName"].(string); ok {
		api.Name = pathname
		api.ID = TitleToKebab(api.Name)
	}
	if api.Name == "" {
		name := o.Summary
		if name == "" {
			logger.Errorf(nil, "Error: Operation '%s' does not have an operationId or summary member.", id)
			os.Exit(1)
		}
		api.Name = name
		api.ID = TitleToKebab(name)
	}

	if c.ResourceList == nil {
		c.ResourceList = make(map[string]map[string]*Resource)
	}

	for _, param := range o.Parameters {
		p := Parameter{
			Name:        param.Name,
			In:          param.In,
			Description: string(github_flavored_markdown.Markdown([]byte(param.Description))),
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
				logger.Errorf(nil, "Error: 'in body' parameter %s is missing a schema declaration.\n", param.Name)
				os.Exit(1)
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

	// Compile resources from response declaration

	if o.Responses == nil {
		logger.Errorf(nil, "Error: Operation %s %s is missing a responses declaration.\n", methodname, path)
		os.Exit(1)
	}
	// FIXME - Dies if there are no responses...
	for status, response := range o.Responses.StatusCodeResponses {
		logger.Tracef(nil, "Response for status %d", status)
		//spew.Dump(response)

		// Discover if the resource is already declared, and pick it up
		// if it is (keyed on version number)
		if response.Schema != nil {
			if _, ok := c.ResourceList[version]; !ok {
				c.ResourceList[version] = make(map[string]*Resource)
			}
		}
		rsp := c.buildResponse(&response, method, version)
		(*rsp).StatusDescription = HTTPStatusDescription(status)
		method.Responses[status] = *rsp

	}

	if o.Responses.Default != nil {
		rsp := c.buildResponse(o.Responses.Default, method, version)
		method.DefaultResponse = rsp
	}

	// If no Security given for operation, then the global defaults are appled.
	method.Security = make(map[string]Security)
	if c.processSecurity(o.Security, method.Security) == false {
		method.Security = c.DefaultSecurity
	}

	return method
}

// -----------------------------------------------------------------------------

func (c *APISpecification) buildResponse(resp *spec.Response, method *Method, version string) *Response {
	var response *Response

	if resp != nil {
		var vres *Resource
		var r *Resource
		var is_array bool
		var example_json map[string]interface{}

		if resp.Schema != nil {
			r, example_json, is_array = c.resourceFromSchema(resp.Schema, method, nil, false)

			if r != nil {
				r.Schema = jsonResourceToString(example_json, false)
				r.origin = MethodResponse
				vres = c.crossLinkMethodAndResource(r, method, version)
			}
		}
		response = &Response{
			Description: string(github_flavored_markdown.Markdown([]byte(resp.Description))),
			Resource:    vres,
			IsArray:     is_array,
		}
		method.Resources = append(method.Resources, response.Resource) // Add the resource to the method which uses it

		response.compileHeaders(resp)
	}
	return response
}

// -----------------------------------------------------------------------------

func (c *APISpecification) crossLinkMethodAndResource(resource *Resource, method *Method, version string) *Resource {

	logger.Tracef(nil, "++ Resource version %s  ID %s\n", version, resource.ID)

	if _, ok := c.ResourceList[version]; !ok {
		c.ResourceList[version] = make(map[string]*Resource)
	}

	// Look for a pre-declared resource with the response ID, and use that or create the first one...
	var resFound bool
	var vres *Resource
	if vres, resFound = c.ResourceList[version][resource.ID]; !resFound {
		logger.Tracef(nil, "   - Creating new resource\n")
		vres = resource
	}

	// Add to the compiled list of methods which use this resource.
	if vres.Methods == nil {
		vres.Methods = make(map[string]*Method)
	}
	vres.Methods[method.ID] = method // Use a map to collapse duplicates.

	// Store resource in resouce-list of the specification, considering precident.
	//
	if resource.origin == RequestBody {
		// Resource is a Request Body - the lowest precident
		//
		logger.Tracef(nil, "   - Resource origin is a request body\n")

		// If this is the first time the resource has been seen, it's okay to store this in
		// the global list. A request body resource is a filtered (excludes read-only) resource,
		// and has a lower precident than a response resource.
		if !resFound {
			logger.Tracef(nil, "     - Not seen before, so storing in global list\n")
			c.ResourceList[version][resource.ID] = vres
		}
	} else {
		logger.Tracef(nil, "   - Resource origin is a response, so storing in global list\n")

		// This is a response resource (which has the highest precident). If an existing
		// request-body resource was found in the cache, then it is replaced by the
		// response resource (but maintaining the method list associated with the resource).
		//
		if resFound && vres.origin == RequestBody {
			resource.Methods = vres.Methods
			vres = resource
		}
		c.ResourceList[version][resource.ID] = vres // If we've already got the resource, this does nothing
	}

	return vres
}

// -----------------------------------------------------------------------------
// OpenAPI/Swagger/go-openAPI define a Header object and an Items object. A
// Header _can_ be an Items object, if it is an array. Annoyingly, a Header
// object is the same as Items but with an additional Description member.
// It would have been nice to treat Header.Items as though it were Header in
// the case of an array...
// Solve both problems by defining accessor methods that will do the "right thing"
// in the case of an array.
func getType(h spec.Header) string {
	if h.Type == "array" {
		return h.Items.Type
	} else {
		return h.Type
	}
}
func getFormat(h spec.Header) string {
	if h.Type == "array" {
		return h.Items.Format
	} else {
		return h.Format
	}
}
func getEnums(h spec.Header) []string {
	var ea []interface{}
	if h.Type == "array" {
		ea = h.Items.Enum
	} else {
		ea = h.Enum
	}
	var es = make([]string, 0)
	for _, e := range ea {
		es = append(es, fmt.Sprintf("%s", e))
	}
	return es
}

var collectionTable *map[string]string

func collectionFormatDescription(format string) string {
	if collectionTable == nil {
		collectionTable = &map[string]string{
			"csv":   "comma separated",
			"ssv":   "space separated",
			"tsv":   "tab separated",
			"pipes": "pipe separated",
			"multi": "multiple occurances",
		}
	}
	if desc, ok := (*collectionTable)[format]; ok {
		return desc
	}
	return ""
}

func (r *Response) compileHeaders(sr *spec.Response) {

	if sr.Headers == nil {
		return
	}
	for name, params := range sr.Headers {

		header := &Header{
			Description: string(github_flavored_markdown.Markdown([]byte(params.Description))),
			Name:        name,
		}

		htype := getType(params)
		if params.Type == "array" {
			if len(params.CollectionFormat) == 0 {
				logger.Errorf(nil, "Error: Response header %s is an array without declaring the collectionFormat.\n", name)
				os.Exit(1)
			}
			header.Type = append(header.Type, params.Type)
			header.CollectionFormat = params.CollectionFormat
			header.CollectionFormatDescription = collectionFormatDescription(params.CollectionFormat)
		}
		format := getFormat(params)
		if len(format) > 0 {
			htype = format
		}
		header.Type = append(header.Type, htype)
		header.Enum = getEnums(params)

		r.Headers = append(r.Headers, *header)
	}
}

// -----------------------------------------------------------------------------

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
						if scope_desc, ok := scheme.Scopes[scope]; ok {
							security[scheme.Type].Scopes[scope] = scope_desc
						}
					}
				}
			}
		}
	}
	return count != 0
}

// -----------------------------------------------------------------------------

func jsonResourceToString(jsonres map[string]interface{}, is_array bool) string {

	// If the resource is an array, then append json object to outer array, else serialise the object.
	var example []byte
	if is_array {
		var array_obj []map[string]interface{}
		array_obj = append(array_obj, jsonres)
		example, _ = JSONMarshalIndent(array_obj)
	} else {
		example, _ = JSONMarshalIndent(jsonres)
	}
	return string(example)
}

// -----------------------------------------------------------------------------

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

	s_orig := s.Type

	if s.Items != nil {
		ptype = "UNKNOWN"

		if s.Type.Contains("array") {

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
			} else if s.Type.Contains("array") {
				ptype = "array of primitives"
			} else {
				ptype = fmt.Sprintf("%s", s_orig)
			}
		} else {
			ptype = "Some object"
		}
	}

	return ptype
}

// -----------------------------------------------------------------------------

func (c *APISpecification) resourceFromSchema(s *spec.Schema, method *Method, fqNS []string, isRequestResource bool) (*Resource, map[string]interface{}, bool) {
	if s == nil {
		return nil, nil, false
	}

	stype := checkPropertyType(s)
	logger.Tracef(nil, "resourceFromSchema: Schema type: %s\n", stype)
	logger.Tracef(nil, "FQNS: %s\n", fqNS)
	logger.Tracef(nil, "CHECK schema type and items\n")
	//spew.Dump(s)

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

	original_s := s
	if s.Items != nil {
		stringorarray := s.Type

		// Jump to nearest schema for items, depending on how it was declared
		if s.Items.Schema != nil { // items: { properties: {} }
			s = s.Items.Schema
			logger.Tracef(nil, "got s.Items.Schema for %s\n", s.Title)
		} else { // items: { $ref: "" }
			s = &s.Items.Schemas[0]
			logger.Tracef(nil, "got s.Items.Schemas[0] for %s\n", s.Title)
		}
		if s.Type == nil {
			logger.Tracef(nil, "Got array of objects or object. Name %s\n", s.Title)
			s.Type = stringorarray // Put back original type
		} else if s.Type.Contains("array") {
			logger.Tracef(nil, "Got array for %s\n", s.Title)
			s.Type = stringorarray // Put back original type
		} else if stringorarray.Contains("array") && len(s.Properties) == 0 {
			// if we get here then we can assume the type is supposed to be an array of primitives
			// Store the actual primitive type in the second element of the Type array.
			s.Type = spec.StringOrArray([]string{"array", s.Type[0]})
		} else {
			s.Type = stringorarray // Put back original type
			logger.Tracef(nil, "putting s.Type back\n")
		}
		logger.Tracef(nil, "REMAP SCHEMA (Type is now %s)\n", s.Type)
	}

	if len(s.Format) > 0 {
		s.Type[len(s.Type)-1] = s.Format
	}

	id := TitleToKebab(s.Title)

	if len(fqNS) == 0 && id == "" {
		logger.Errorf(nil, "Error: %s %s references a model definition that does not have a title member.", strings.ToUpper(method.Method), method.Path)
		os.Exit(1)
	}

	// Ignore ID (from title element) for all but child-objects...
	// This prevents the title-derived ID being added onto the end of the FQNS.property as
	// FQNS.property.ID, if title is given for the property in the spec.
	if len(fqNS) > 0 && !s.Type.Contains("object") {
		id = ""
	}

	var is_array bool
	if strings.ToLower(s.Type[0]) == "array" {
		fqNSlen := len(fqNS)
		if fqNSlen > 0 {
			fqNS = append(fqNS[0:fqNSlen-1], fqNS[fqNSlen-1]+"[]")
		}
		is_array = true
	}

	myFQNS := fqNS
	var chopped bool

	if len(id) == 0 && len(myFQNS) > 0 {
		id = myFQNS[len(myFQNS)-1]
		myFQNS = append([]string{}, myFQNS[0:len(myFQNS)-1]...)
		chopped = true
		logger.Tracef(nil, "Chopped %s from myFQNS leaving %s\n", id, myFQNS)
	}

	resourceFQNS := myFQNS
	// If we are dealing with an object, then adjust the resource FQNS and id
	// so that the last element of the FQNS is chopped off and used as the ID
	if !chopped && s.Type.Contains("object") {
		if len(resourceFQNS) > 0 {
			id = resourceFQNS[len(resourceFQNS)-1]
			resourceFQNS = resourceFQNS[:len(resourceFQNS)-1]
			logger.Tracef(nil, "Got an object, so slicing %s from resourceFQNS leaving %s\n", id, myFQNS)
		}
	}

	// If there is no description... the case where we have an array of objects. See issue/11
	var description string
	if original_s.Description != "" {
		description = string(github_flavored_markdown.Markdown([]byte(original_s.Description)))
	} else {
		description = original_s.Title
	}

	logger.Tracef(nil, "Create resource %s [%s]\n", id, s.Title)
	if is_array {
		logger.Tracef(nil, "- Is Arrays\n")
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
		example, err := JSONMarshalIndent(&s.Example)
		if err != nil {
			logger.Errorf(nil, "Error encoding example json: %s", err)
		}
		r.Example = string(example)
	}

	if len(s.Enum) > 0 {
		for _, e := range s.Enum {
			r.Enum = append(r.Enum, fmt.Sprintf("%s", e))
		}
	}

	r.ReadOnly = original_s.ReadOnly
	if ops, ok := original_s.Extensions["x-excludeFromOperations"].([]interface{}); ok && isRequestResource {
		// Mark resource property as being excluded from operations with this name.
		// This filtering only takes effect in a request body, just like readOnly, so when isRequestResource is true
		for _, op := range ops {
			if c, ok := op.(string); ok {
				r.ExcludeFromOperations = append(r.ExcludeFromOperations, c)
			}
		}
	}

	required := make(map[string]bool)
	json_representation := make(map[string]interface{})

	logger.Tracef(nil, "Call compileproperties...\n")
	c.compileproperties(s, r, method, id, required, json_representation, myFQNS, chopped, isRequestResource)

	for allof := range s.AllOf {
		c.compileproperties(&s.AllOf[allof], r, method, id, required, json_representation, myFQNS, chopped, isRequestResource)
	}

	logger.Tracef(nil, "resourceFromSchema done\n")

	return r, json_representation, is_array
}

// -----------------------------------------------------------------------------
// Takes a Schema object and adds properties to the Resource object.
// It uses the 'required' map to set when properties are required and builds a JSON
// representation of the resource.
//
func (c *APISpecification) compileproperties(s *spec.Schema, r *Resource, method *Method, id string, required map[string]bool, json_rep map[string]interface{}, myFQNS []string, chopped bool, isRequestResource bool) {

	// First, grab the required members
	for _, n := range s.Required {
		required[n] = true
	}

	for name, property := range s.Properties {
		c.processProperty(&property, name, r, method, id, required, json_rep, myFQNS, chopped, isRequestResource)
	}

	// Special case to deal with AdditionalProperties (which really just boils down to declaring a
	// map of 'type' (string, int, object etc).
	if s.AdditionalProperties != nil && s.AdditionalProperties.Allows {
		name := "<key>"
		ap := s.AdditionalProperties.Schema
		ap.Type = spec.StringOrArray([]string{"map", ap.Type[0]}) // massage type so that it is a map of 'type'

		c.processProperty(ap, name, r, method, id, required, json_rep, myFQNS, chopped, isRequestResource)
	}
}

// -----------------------------------------------------------------------------

func (c *APISpecification) processProperty(s *spec.Schema, name string, r *Resource, method *Method, id string, required map[string]bool, json_rep map[string]interface{}, myFQNS []string, chopped bool, isRequestResource bool) {

	newFQNS := prepareNamespace(myFQNS, id, name, chopped)

	var json_resource map[string]interface{}
	var resource *Resource

	logger.Tracef(nil, "A call resourceFromSchema for property %s\n", name)
	resource, json_resource, _ = c.resourceFromSchema(s, method, newFQNS, isRequestResource)

	skip := isRequestResource && resource.ReadOnly
	if !skip && resource.ExcludeFromOperations != nil {

		logger.Tracef(nil, "Exclude [%s] in operation [%s] if in list: %s\n", name, method.OperationName, resource.ExcludeFromOperations)

		for _, opname := range resource.ExcludeFromOperations {
			if opname == method.OperationName {
				logger.Tracef(nil, "[%s] is excluded\n", name)
				skip = true
				break
			}
		}
	}
	if skip {
		return
	}

	r.Properties[name] = resource
	json_rep[name] = json_resource

	if _, ok := required[name]; ok {
		r.Properties[name].Required = true
	}
	logger.Tracef(nil, "resource property %s type: %s\n", name, r.Properties[name].Type[0])

	if strings.ToLower(r.Properties[name].Type[0]) != "object" {
		// Arrays of objects need to be handled as a special case
		if strings.ToLower(r.Properties[name].Type[0]) == "array" {
			logger.Tracef(nil, "Processing an array property %s", name)
			if s.Items != nil {
				if s.Items.Schema != nil {
					// Some outputs (example schema, member description) are generated differently
					// if the array member references an object or a primitive type
					r.Properties[name].Description = string(github_flavored_markdown.Markdown([]byte(s.Description)))

					// If here, we have no json_resource returned from resourceFromSchema, then the property
					// is an array of primitive, so construct either an array of string or array of object
					// as appropriate.
					if len(json_resource) > 0 {
						var array_obj []map[string]interface{}
						array_obj = append(array_obj, json_resource)
						json_rep[name] = array_obj
					} else {
						var array_obj []string
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
							array_obj = append(array_obj, r.Properties[name].Type[1])
						}
						json_rep[name] = array_obj
					}
				} else { // array and property.Items.Schema is NIL
					var array_obj []map[string]interface{}
					array_obj = append(array_obj, json_resource)
					json_rep[name] = array_obj
				}
			} else { // array and Items are nil
				var array_obj []map[string]interface{}
				array_obj = append(array_obj, json_resource)
				json_rep[name] = array_obj
			}
		} else if strings.ToLower(r.Properties[name].Type[0]) == "map" { // not array, so a map?
			if strings.ToLower(r.Properties[name].Type[1]) == "object" {
				json_rep[name] = json_resource // A map of objects
			} else {
				json_rep[name] = r.Properties[name].Type[1] // map of primitive
			}
		} else {
			// We're NOT an array, map or object, so a primitive
			json_rep[name] = r.Properties[name].Type[0]
		}
	} else {
		// We're an object
		json_rep[name] = json_resource
	}
	return
}

// -----------------------------------------------------------------------------

func prepareNamespace(myFQNS []string, id string, name string, chopped bool) []string {

	newFQNS := append([]string{}, myFQNS...) // create slice

	if chopped && len(id) > 0 {
		logger.Tracef(nil, "Append ID onto newFQNZ %s + '%s'", newFQNS, id)
		newFQNS = append(newFQNS, id)
	}

	newFQNS = append(newFQNS, name)

	return newFQNS
}

// -----------------------------------------------------------------------------

var kababExclude = regexp.MustCompile("[^\\w\\s]") // Any non word or space character

func TitleToKebab(s string) string {
	s = strings.ToLower(s)
	s = string(kababExclude.ReplaceAll([]byte(s), []byte("")))
	s = strings.Replace(s, " ", "-", -1)
	return s
}

// -----------------------------------------------------------------------------

func CamelToKebab(s string) string {
	s = snaker.CamelToSnake(s)
	s = strings.Replace(s, "_", "-", -1)
	return s
}

// -----------------------------------------------------------------------------

func loadSpec(url string) (*loads.Document, error) {

	logger.Infof(nil, "Importing OpenAPI specifications from %s", url)

	document, err := loads.Spec(url)
	if err != nil {
		//logger.Errorf(nil, "Error: go-openapi/loads filed to load spec url [%s]: %s", url, err)
		return nil, err
	}

	// Because Title is required for resources but optional in the Open API spec, add the Title here for any definitions missing one
	definitions := document.Spec().Definitions
	for name, definition := range definitions {
		if definition.Title == "" {
			definition.Title = name
			definitions[name] = definition
			logger.Tracef(nil, "Definition for %s missing Title, setting Title to \"%s\"", name, definition.Title)
		}
	}
	
	//options := &spec.ExpandOptions{
	//	RelativeBase: "/Users/csmith1/src/go/src/dapperdox-demo/specifications",
	//}

	// TODO Allow relative references https://github.com/go-openapi/spec/issues/14
	err = spec.ExpandSpec(document.Spec(), nil)
	if err != nil {
		//logger.Errorf(nil, "Error: go-openapi/spec filed to expand spec: %s", err)
		return nil, err
	}

	return document, nil
}

// -----------------------------------------------------------------------------
// Wrapper around MarshalIndent to prevent < > & from being escaped
func JSONMarshalIndent(v interface{}) ([]byte, error) {
	b, err := json.MarshalIndent(v, "", "    ")

	b = bytes.Replace(b, []byte("\\u003c"), []byte("<"), -1)
	b = bytes.Replace(b, []byte("\\u003e"), []byte(">"), -1)
	b = bytes.Replace(b, []byte("\\u0026"), []byte("&"), -1)
	return b, err
}

// -----------------------------------------------------------------------------

func isLocalSpecUrl(specUrl string) bool {
	match, err := regexp.MatchString("(?i)^https?://.+", specUrl)
	if err != nil {
		panic(fmt.Sprintf("Attempted to match against an invalid regexp: %s", err))
	}
	return !match
}

// -----------------------------------------------------------------------------

func normalizeSpecLocation(specLocation string, specHost string) string {
	if isLocalSpecUrl(specLocation) {
		return "http://" + specHost + specLocation
	} else {
		return specLocation
	}
}
