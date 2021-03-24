package discover

import (
	"encoding/json"
	"net/url"

	"github.com/go-openapi/loads"
	"github.com/go-openapi/spec"
	"github.com/go-openapi/swag"
	wraperrors "github.com/pkg/errors"
	"github.com/spf13/viper"

	"github.com/kenjones-cisco/dapperdox/config"
)

func loadSpec(location string) (*spec.Swagger, error) {
	if location == "" {
		return nil, wraperrors.New("api location has no value")
	}

	u := &url.URL{Host: location, Scheme: "http", Path: "swagger.json"}

	log().Debugf("apiLoader location: %s", u.String())

	data, err := swag.LoadFromFileOrHTTPWithTimeout(u.String(), viper.GetDuration(config.DiscoverySpecLoadTimeout))
	if err != nil {
		return nil, err
	}

	var doc *loads.Document

	doc, err = loads.Analyzed(json.RawMessage(data), "")
	if err != nil {
		return nil, err
	}

	return doc.Spec(), nil
}
