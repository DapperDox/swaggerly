package discover

import (
	"fmt"
	"strings"

	"github.com/go-openapi/loads"
	"github.com/go-openapi/spec"
	"github.com/go-openapi/swag"
	wraperrors "github.com/pkg/errors"
	"github.com/spf13/viper"

	"github.com/kenjones-cisco/dapperdox/config"
)

const (
	visibilityExt = "x-visibility"

	groupByPrivate = "Private Cloud Services"
	groupByPublic  = "Public Cloud Services"
	groupByCore    = "Platform APIs"

	// key values swagger spec extensions.
	extKeyMCMPComponentType = "x-mcmp-component-type"
	extKeyGroupBy           = "x-groupby"
)

func (d *Discoverer) fetchAPISpecs() map[string][]byte {
	// if there are rewrite configuration defined,
	// ensure we are able to load them
	var rewritesDoc *loads.Document

	if viper.GetString(config.SpecRewrites) != "" {
		log().Info(viper.GetString(config.SpecRewrites))

		data, err := swag.YAMLDoc(viper.GetString(config.SpecRewrites))
		if err != nil {
			log().WithError(err).Errorf("unable to load rewrites from file: %v", viper.GetString(config.SpecRewrites))

			return nil
		}

		rewritesDoc, err = loads.Analyzed(data, "")
		if err != nil {
			log().WithError(err).Error("unable to analyze rewrites")

			return nil
		}
	}

	newSpecs := make(map[string][]byte)

	for _, service := range d.data.services.List() {
		if service.Hostname == "" || inIgnoreList(service.Hostname) {
			log().Warnf("invalid service %q", service.Hostname)

			continue
		}

		for _, port := range service.Ports {
			if !port.Protocol.IsHTTP() {
				continue
			}

			hostName := service.Hostname
			portNum := port.Port

			svcSpec, err := loadSpec(fmt.Sprintf("%s:%d", hostName, portNum))
			if err != nil {
				log().Errorf("unable to load specs for %v: %v", hostName, err)

				continue
			}

			path, data, err := processSpec(hostName, rewritesDoc.Spec(), svcSpec)
			if err != nil {
				log().Errorf("unable to load specs for %v: %v", hostName, err)

				continue
			}

			newSpecs[path] = data
		}
	}

	if len(newSpecs) > 0 {
		return newSpecs
	}

	return nil
}

func processSpec(hostname string, rewritesSpec, svcSpec *spec.Swagger) (string, []byte, error) {
	if svcSpec == nil {
		return "", nil, wraperrors.New("service spec should not be nil")
	}

	removePrivateAPIs(svcSpec)

	removePrivateDefinitions(svcSpec)

	applyGrouping(svcSpec)

	if rewritesSpec != nil {
		applyRewrites(rewritesSpec, svcSpec)
	}

	outdata, err := svcSpec.MarshalJSON()
	if err != nil {
		return "", nil, wraperrors.Wrap(err, "unable to marshal final spec")
	}

	return fmt.Sprintf("%s/%s", strings.TrimRight(viper.GetString(config.SpecDir), "/"), hostname), outdata, nil
}

func removePrivateAPIs(svcSpec *spec.Swagger) {
	pathsRef := svcSpec.Paths

	if pathsRef == nil {
		log().Warning("no API paths defined")

		return
	}

	// iterate over Paths map
	for k, v := range pathsRef.Paths {
		// remove the entire API path from swagger spec if it's marked as `private`
		if isPrivate(v.Extensions) {
			delete(pathsRef.Paths, k)

			continue
		}

		// remove any defined method from the above path that's marked as `private`
		if v.Get != nil && isPrivate(v.Get.Extensions) {
			v.Get = nil
		}

		if v.Put != nil && isPrivate(v.Put.Extensions) {
			v.Put = nil
		}

		if v.Patch != nil && isPrivate(v.Patch.Extensions) {
			v.Patch = nil
		}

		if v.Post != nil && isPrivate(v.Post.Extensions) {
			v.Post = nil
		}

		if v.Delete != nil && isPrivate(v.Delete.Extensions) {
			v.Delete = nil
		}

		// re-insert modified spec.PathItem value into Paths map
		pathsRef.Paths[k] = v
	}
}

func removePrivateDefinitions(svcSpec *spec.Swagger) {
	definitionsRef := svcSpec.Definitions

	for k, v := range definitionsRef {
		if isPrivate(v.Extensions) {
			delete(definitionsRef, k)
		}
	}
}

func applyGrouping(svcSpec *spec.Swagger) {
	// create extensions if non exist
	if svcSpec.Extensions == nil {
		svcSpec.Extensions = make(map[string]interface{})
	}

	// set grouping extension based on value in `x-mcmp-component-type` extensions
	switch svcSpec.Extensions[extKeyMCMPComponentType] {
	case "private-cloud":
		svcSpec.Extensions.Add(extKeyGroupBy, groupByPrivate)
	case "public-cloud":
		svcSpec.Extensions.Add(extKeyGroupBy, groupByPublic)
	case "core":
		svcSpec.Extensions.Add(extKeyGroupBy, groupByCore)
	default:
		svcSpec.Extensions.Add(extKeyGroupBy, groupByCore)
	}

	// set custom grouping extension if defined
	for _, tag := range svcSpec.Tags {
		if customgroup, ok := viper.GetStringMapString(config.SpecGroupings)[tag.Name]; ok {
			svcSpec.Extensions.Add(extKeyGroupBy, customgroup)
		}
	}
}

func applyRewrites(rewrites, svcSpec *spec.Swagger) {
	// replace spec details for the following values already defined
	if len(rewrites.SecurityDefinitions) > 0 {
		svcSpec.SecurityDefinitions = rewrites.SecurityDefinitions
	}

	if len(rewrites.Security) > 0 {
		svcSpec.Security = rewrites.Security
	}

	if len(rewrites.Schemes) > 0 {
		svcSpec.Schemes = rewrites.Schemes
	}

	for k, v := range rewrites.Extensions {
		// note: not leveraging spec.Extensions.Add() since it applies ToLower() to key value,
		//  affecting the case-sensitive key lookup by dapperdox
		//  - https://github.com/DapperDox/dapperdox/blob/e343254818a8c67c29de6b192e11b0fcb0703800/spec/spec.go#L345,L350
		svcSpec.Extensions[k] = v
	}
}

func isPrivate(exts spec.Extensions) bool {
	if pv, ok := exts.GetString(visibilityExt); ok {
		return pv == "private"
	}

	return false
}
