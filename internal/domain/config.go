package domain

import "time"

// OutputFormat determines how responses are displayed.
type OutputFormat string

const (
	OutputPretty  OutputFormat = "pretty"
	OutputJSON    OutputFormat = "json"
	OutputRaw     OutputFormat = "raw"
	OutputMinimal OutputFormat = "minimal"
)

// PluginConfig holds configuration for the plugin system.
type PluginConfig struct {
	Dir     string   // directory to scan for .so plugin files
	Enabled []string // list of enabled plugin names
}

// AppConfig holds global application configuration.
type AppConfig struct {
	Timeout        time.Duration
	DataDir        string
	LogLevel       string
	NoColor        bool
	Output         OutputFormat
	DefaultHeaders []Header
	Plugins        PluginConfig
}
