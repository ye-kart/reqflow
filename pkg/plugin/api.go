// Package plugin defines the public extension API for reqflow.
// Third-party plugins implement these interfaces to add custom protocols,
// authentication providers, and output formatters.
package plugin

import "context"

// Protocol defines a custom request protocol (e.g., gRPC, MQTT).
type Protocol interface {
	Name() string
	Execute(ctx context.Context, config map[string]interface{}) (map[string]interface{}, error)
}

// AuthProvider defines a custom authentication mechanism.
type AuthProvider interface {
	Name() string
	Type() string // auth type identifier
	Apply(headers map[string]string, config map[string]string) (map[string]string, error)
}

// OutputFormatter defines a custom output format (e.g., XML, CSV).
type OutputFormatter interface {
	Name() string
	Format(data map[string]interface{}) ([]byte, error)
}

// Plugin is the base interface that all plugins must implement.
type Plugin interface {
	Name() string
	Version() string
	Init(config map[string]string) error
}
