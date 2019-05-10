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
package guides

import (
	//"github.com/davecgh/go-spew/spew"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"dapperdox/logger"
	"dapperdox/navigation"
	"dapperdox/render"
	"dapperdox/render/asset"
	"dapperdox/spec"
	"github.com/gorilla/pat"
)

// ---------------------------------------------------------------------------
// Register routes for guide pages
func Register(r *pat.Router) {

	logger.Infof(nil, "Registering guides")

	// specification specific guides
	for _, specification := range spec.APISuite {
		logger.Debugf(nil, "- Specification guides for '%s'", specification.APIInfo.Title)
		register(r, "assets/templates", specification)
	}

	// Top level guides
	logger.Debugf(nil, "- Root guides")
	register(r, "assets/templates", nil)

	logger.Debugf(nil, "\n")
}

// ---------------------------------------------------------------------------
func register(r *pat.Router, base string, specification *spec.APISpecification) {

	root_node := "/guides"
	route_base := "/guides"
	if specification != nil {
		root_node = "/" + specification.ID + "/templates" + root_node
		route_base = "/" + specification.ID + route_base
	}

	path_base := base + root_node

	guidesNavigation := &navigation.NavigationNode{}

	guidesNavigation.Children = make([]*navigation.NavigationNode, 0)
	guidesNavigation.ChildMap = make(map[string]*navigation.NavigationNode)

	logger.Tracef(nil, "  - Walk compiled asset tree %s", path_base)

	for _, path := range asset.AssetNames() {
		if !strings.HasPrefix(path, path_base) { // Only keep assets we want
			continue
		}
		ext := filepath.Ext(path)

		switch ext {
		case ".tmpl", ".md":
			logger.Debugf(nil, "    - File "+path)

			// Convert path/filename to route
			route := route_base + StripBasepathAndExtension(path, path_base)
			absresource := StripBasepathAndExtension(path, base)
			resource := strings.TrimPrefix(absresource, "/")

			logger.Tracef(nil, "      = URL  "+route)

			buildNavigation(guidesNavigation, path, path_base, route, ext)

			r.Path(route).Methods("GET").HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				sid := "TOP LEVEL"
				if specification != nil {
					sid = specification.ID
				}
				logger.Tracef(nil, "Fetching guide from '%s' for spec ID %s\n", resource, sid)
				render.HTML(w, http.StatusOK, resource, render.DefaultVars(req, specification, render.Vars{"Guide": resource}))
			})
		}
	}

	sortNavigation(guidesNavigation)

	// Register default route for this guide set
	r.Path(route_base).Methods("GET").HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		uri := findFirstGuideUri(guidesNavigation)
		logger.Infof(nil, "Redirect to %s\n", uri)
		http.Redirect(w, req, uri, 302)
	})

	// Register the guides navigation with the renderer
	render.SetGuidesNavigation(specification, &guidesNavigation.Children)
}

// ---------------------------------------------------------------------------

func findFirstGuideUri(tree *navigation.NavigationNode) string {
	var uri string
	for i := range tree.Children {
		node := tree.Children[i]
		uri = node.Uri
		if uri == "" {
			if len(node.Children) > 0 {
				uri = findFirstGuideUri(node)
			}
		}
		if uri != "" {
			break
		}
	}
	return uri
}

// ---------------------------------------------------------------------------
func sortNavigation(tree *navigation.NavigationNode) {

	for i := range tree.Children {
		node := tree.Children[i]

		if len(node.Children) > 0 {
			sort.Sort(navigation.ByOrder(node.Children))
		}
	}
	sort.Sort(navigation.ByOrder(tree.Children))
}

// ---------------------------------------------------------------------------
func dumpit(tree *navigation.NavigationNode) {
	for i := range tree.Children {
		node := tree.Children[i]

		logger.Tracef(nil, "Sorted name = %s\n", node.Name)
		for j := range node.Children {
			node2 := node.Children[j]
			logger.Tracef(nil, "       name = %s\n", node2.Name)
		}
	}
}

// ---------------------------------------------------------------------------
func StripBasepathAndExtension(name string, basepath string) string {
	// Strip base path and file extension
	route := strings.TrimSuffix(strings.TrimPrefix(name, basepath), filepath.Ext(name))

	return route
}

// ---------------------------------------------------------------------------
func buildNavigation(nav *navigation.NavigationNode, path string, path_base string, route string, ext string) {

	logger.Tracef(nil, "      - Look for metadata asset %s\n", path)

	// See if guide has been marked up with nagivation metadata...
	hierarchy := asset.MetaData(path, "Navigation")
	sortOrder := asset.MetaData(path, "SortOrder")

	if len(hierarchy) > 0 {
		logger.Tracef(nil, "      * Got navigation metadata %s for file %s\n", hierarchy, path)
	} else {
		// No Meta Data set on guide, so use the directory structure
		hierarchy = strings.TrimPrefix(strings.TrimSuffix(path, ext), path_base+"/")
		logger.Tracef(nil, "      * No navigation metadata for "+hierarchy+". Using path")
	}

	// Break hierarchy into bits
	split := strings.Split(hierarchy, "/")
	parts := len(split)

	if parts > 2 {
		logger.Errorf(nil, "Error: Guide '"+hierarchy+"' contains too many nagivation levels (%d)", parts)
		os.Exit(1)
	}

	if sortOrder == "" {
		sortOrder = route
	}

	current := nav.ChildMap
	currentList := &nav.Children

	// Build tree for this navigation item
	for i := range split {

		name := split[i]
		id := strings.Replace(strings.ToLower(name), " ", "-", -1)
		id = strings.Replace(id, ".", "-", -1)

		if i < parts-1 {
			// Have we already created this branch node?
			if currentItem, ok := current[id]; !ok {
				// create new branch node
				current[id] = &navigation.NavigationNode{
					Id:        id,
					SortOrder: sortOrder,
					Name:      name,
					ChildMap:  make(map[string]*navigation.NavigationNode),
					Children:  make([]*navigation.NavigationNode, 0),
				}
				*currentList = append(*currentList, current[id])
				logger.Tracef(nil, "      + Adding %s = %s to branch\n", id, current[id].Name)
			} else {
				// Update the branch node sort order, if the leaf has a lower sort
				if sortOrder < currentItem.SortOrder {
					currentItem.SortOrder = sortOrder
				}
			}
			// Step down branch
			currentList = &current[id].Children // Get parent list before stepping into child

			current = current[id].ChildMap
		} else {
			// Leaf node
			if currentItem, ok := current[id]; !ok {
				current[id] = &navigation.NavigationNode{
					Id:        id,
					SortOrder: sortOrder,
					Uri:       route,
					Name:      name,
					ChildMap:  make(map[string]*navigation.NavigationNode),
					Children:  make([]*navigation.NavigationNode, 0),
				}
				*currentList = append(*currentList, current[id])
				logger.Tracef(nil, "      + Adding %s = %s to leaf node [a] Sort %s\n", current[id].Uri, current[id].Name, sortOrder)
			} else {
				// The page is a leaf node, but sits at a branch node. This means that the branch
				// node has content! Set the uri, and adjust the sort order, if necessary.
				currentItem.Uri = route
				if sortOrder < currentItem.SortOrder {
					currentItem.SortOrder = sortOrder
				}
				logger.Tracef(nil, "      + Adding %s = %s to leaf node [b] Sort %s\n", currentItem.Uri, currentItem.Name, sortOrder)
			}
		}
	}
}

// ---------------------------------------------------------------------------
