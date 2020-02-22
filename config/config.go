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
package config

import (
	"reflect"
	"strings"

	"github.com/DapperDox/dapperdox/logger"
	"github.com/ian-kent/gofigure"
)

type config struct {
	gofigure           interface{} `order:"env,flag"`
	BindAddr           string      `env:"BIND_ADDR" flag:"bind-addr" flagDesc:"Bind address"`
	AssetsDir          string      `env:"ASSETS_DIR" flag:"assets-dir" flagDesc:"Assets to serve. Effectively the document root."`
	DefaultAssetsDir   string      `env:"DEFAULT_ASSETS_DIR" flag:"default-assets-dir" flagDesc:"Default assets."`
	SpecDir            string      `env:"SPEC_DIR" flag:"spec-dir" flagDesc:"OpenAPI specification (swagger) directory"`
	SpecFilename       []string    `env:"SPEC_FILENAME" flag:"spec-filename" flagDesc:"The filename of the OpenAPI specification file within the spec-dir. May be multiply defined. Defaults to spec/swagger.json"`
	Theme              string      `env:"THEME" flag:"theme" flagDesc:"Theme to render documentation"`
	ThemeDir           string      `env:"THEME_DIR" flag:"theme-dir" flagDesc:"Directory containing installed themes"`
	LogLevel           string      `env:"LOGLEVEL" flag:"log-level" flagDesc:"Log level"`
	SiteURL            string      `env:"SITE_URL" flag:"site-url" flagDesc:"Public URL of the documentation service"`
	SpecRewriteURL     []string    `env:"SPEC_REWRITE_URL" flag:"spec-rewrite-url" flagDesc:"The URLs in the swagger specifications to be rewritten as site-url"`
	DocumentRewriteURL []string    `env:"DOCUMENT_REWRITE_URL" flag:"document-rewrite-url" flagDesc:"Specify a document URL that is to be rewritten. May be multiply defined. Format is from=to."`
	ForceSpecList      bool        `env:"FORCE_SPECIFICATION_LIST" flag:"force-specification-list" flagDesc:"Force the homepage to be the summary list of available specifications. The default when serving a single OpenAPI specification is to make the homepage the API summary."`
	ShowAssets         bool        `env:"AUTHOR_SHOW_ASSETS" flag:"author-show-assets" flagDesc:"Display at the foot of each page the overlay asset paths, in priority order, that DapperDox will check before rendering."`
	ProxyPath          []string    `env:"PROXY_PATH" flag:"proxy-path" flagDesc:"Give a path to proxy though to another service. May be multiply defined. Format is local-path=scheme://host/dst-path."`
	TLSCertificate     string      `env:"TLS_CERTIFICATE" flag:"tls-certificate" flagDesc:"The fully qualified path to the TLS certificate file. For HTTP over TLS (HTTPS) both a certificate and a key must be provided."`
	TLSKey             string      `env:"TLS_KEY" flag:"tls-key" flagDesc:"The fully qualified path to the TLS private key file. For HTTP over TLS (HTTPS) both a certificate and a key must be provided."`
}

var cfg *config

// Get configures the application and returns the configuration
func Get() (*config, error) {
	if cfg != nil {
		return cfg, nil
	}

	cfg = &config{
		BindAddr:         "localhost:3123",
		SpecDir:          "",
		DefaultAssetsDir: "assets",
		LogLevel:         "info",
		SiteURL:          "http://localhost:3123/",
		ShowAssets:       false,
	}

	err := gofigure.Gofigure(cfg)
	if err != nil {
		return nil, err
	}

	if len(cfg.SpecFilename) == 0 {
		cfg.SpecFilename = append(cfg.SpecFilename, "/swagger.json")
	}

	cfg.print()

	return cfg, nil
}

func (c *config) print() {
	logger.Println(nil, "Configuration:")

	s := reflect.ValueOf(c).Elem()
	t := s.Type()

	ml := 0
	for i := 0; i < s.NumField(); i++ {
		if !s.Field(i).CanSet() {
			continue
		}
		if l := len(t.Field(i).Name); l > ml {
			ml = l
		}
	}

	for i := 0; i < s.NumField(); i++ {
		f := s.Field(i)
		if !s.Field(i).CanSet() {
			continue
		}
		logger.Printf(nil, "\t%s%s: %s\n", strings.Repeat(" ", ml-len(t.Field(i).Name)), t.Field(i).Name, f.Interface())
	}
}
