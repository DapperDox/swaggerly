package config

import (
	"fmt"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

// all configuration keys.
const (
	cfgDirKey = "config-dir"
	LogLevel  = "log-level"
	Help      = "help"
	Version   = "version"

	BindAddr           = "bind-addr"
	TLSCert            = "tls-certificate"
	TLSKey             = "tls-key"
	SiteURL            = "site-url"
	ProxyPath          = "proxy.path"
	DocumentRewriteURL = "document.rewrite.url"
	AllowOrigin        = "allow.origin"

	// assets.
	DefaultAssetsDir = "default-assets-dir"
	AssetsDir        = "assets-dir"
	ShowAssets       = "author-show-assets"

	// theme.
	Theme    = "theme"
	ThemeDir = "theme-dir"

	// spec.
	SpecDir         = "spec-dir"
	SpecFilename    = "spec-filename"
	SpecDefaultHost = "spec.default.host"
	SpecRewriteURL  = "spec.rewrite.url"
	SpecRewrites    = "spec.rewrites"
	SpecGroupings   = "spec.groupings"
	ForceSpecList   = "force-specification-list"

	// auto-discovery configs.
	DiscoveryDomain            = "discovery.domain"
	DiscoveryNamespace         = "discovery.namespace"
	DiscoveryInterval          = "discovery.interval"
	DiscoveryRetries           = "discovery.retries"
	DiscoveryInitialDelay      = "discovery.delay.initial"
	DiscoveryMaxDelay          = "discovery.delay.max"
	DiscoverySpecLoadTimeout   = "discovery.spec.timeout"
	DiscoveryServiceIgnoreList = "discovery.ignorelist.services"
)

var defaultConfigPaths = []string{
	"/etc/viper",
	"./",
}

// C holds a reference to struct instance for configurations used within template files.
var C config

type config struct {
	ShowAssets bool `mapstructure:"author-show-assets"`
}

func init() {
	pflag.String(cfgDirKey, "", "Directory of config file")
	pflag.String(LogLevel, "info", "Logging level ('error', 'warn', 'info', 'debug', 'trace')")
	pflag.BoolP(Version, "V", false, "Display version")

	pflag.String(BindAddr, "localhost:3123", "Bind address")
	pflag.String(TLSCert, "", "The fully qualified path to the TLS certificate file. For HTTP over TLS (HTTPS) both a certificate and a key must be provided")
	pflag.String(TLSKey, "", "The fully qualified path to the TLS private key file. For HTTP over TLS (HTTPS) both a certificate and a key must be provided")
	pflag.String(SiteURL, "http://localhost:3123/", "Public URL of the documentation service")

	pflag.String(DefaultAssetsDir, "assets", "Default assets directory")
	pflag.String(AssetsDir, "", "Assets to serve. Effectively the document root")
	pflag.Bool(ShowAssets, false, "Display at the foot of each page the overlay asset paths, in priority order, to check before rendering")

	pflag.String(Theme, "default", "Theme to render documentation")
	pflag.String(ThemeDir, "", "Directory containing installed themes")

	pflag.String(SpecDir, "", "OpenAPI specification (swagger) directory")
	pflag.StringSlice(SpecFilename, []string{}, "The filename of the OpenAPI specification file within the spec-dir. May be multiply defined.")
	pflag.Bool(ForceSpecList, false,
		"Force the homepage to be the summary list of available specifications. The default when serving a single OpenAPI specification is to make the homepage the API summary.")

	initialize()
}

// Init performs the initialization of the configurations via configuration file when found.
func Init() {
	viper.SetConfigName("config")

	confDir := viper.GetString(cfgDirKey)
	if confDir != "" {
		viper.AddConfigPath(confDir)
	}

	for _, p := range defaultConfigPaths {
		viper.AddConfigPath(p)
	}

	if err := viper.ReadInConfig(); err == nil {
		fmt.Printf("Using config: %s\n", viper.ConfigFileUsed())
	}

	if err := viper.Unmarshal(&C); err != nil {
		panic(err)
	}
}

// LoadFixture will load test fixture configuration; for testing only!
func LoadFixture(dir string) error {
	viper.SetConfigName("config")
	viper.AddConfigPath(dir)

	return viper.ReadInConfig()
}

// Restore will reset viper and re-initialize back to the default configurations
// For Testing ONLY!
func Restore() {
	viper.Reset()
	initialize()
}

func initialize() {
	viper.SetDefault(AllowOrigin, []string{"*"})

	viper.SetDefault(SpecFilename, []string{"/swagger.json"})
	viper.SetDefault(SpecDefaultHost, "127.0.0.1")

	viper.SetDefault(DiscoveryDomain, "svc.cluster.local")
	viper.SetDefault(DiscoveryNamespace, "default")
	viper.SetDefault(DiscoveryInterval, "10s")
	viper.SetDefault(DiscoveryRetries, "5")
	viper.SetDefault(DiscoveryInitialDelay, "1s")
	viper.SetDefault(DiscoveryMaxDelay, "10s")
	viper.SetDefault(DiscoverySpecLoadTimeout, "1s")

	_ = viper.BindEnv(cfgDirKey, "CONFIG_DIR")
	_ = viper.BindEnv(LogLevel, "LOGLEVEL")

	_ = viper.BindEnv(BindAddr, "BIND_ADDR")
	_ = viper.BindEnv(TLSCert, "TLS_CERTIFICATE")
	_ = viper.BindEnv(TLSKey, "TLS_KEY")
	_ = viper.BindEnv(SiteURL, "SITE_URL")

	_ = viper.BindEnv(DefaultAssetsDir, "DEFAULT_ASSETS_DIR")
	_ = viper.BindEnv(AssetsDir, "ASSETS_DIR")
	_ = viper.BindEnv(ShowAssets, "AUTHOR_SHOW_ASSETS")

	_ = viper.BindEnv(Theme, "THEME")
	_ = viper.BindEnv(ThemeDir, "THEME_DIR")

	_ = viper.BindEnv(SpecDir, "SPEC_DIR")
	_ = viper.BindEnv(SpecFilename, "SPEC_FILENAME")
	_ = viper.BindEnv(SpecDefaultHost, "SPEC_DEFAULT_HOST")
	_ = viper.BindEnv(ForceSpecList, "FORCE_SPECIFICATION_LIST")

	_ = viper.BindEnv(DiscoveryNamespace, "POD_NAMESPACE")
}
