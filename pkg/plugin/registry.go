package plugin

import "sync"

var (
	mu         sync.RWMutex
	protocols  = make(map[string]Protocol)
	auths      = make(map[string]AuthProvider)
	formatters = make(map[string]OutputFormatter)
)

// RegisterProtocol registers a custom protocol plugin.
// If a protocol with the same name already exists, it is overwritten.
func RegisterProtocol(p Protocol) {
	mu.Lock()
	defer mu.Unlock()
	protocols[p.Name()] = p
}

// RegisterAuthProvider registers a custom authentication provider plugin.
// If a provider with the same name already exists, it is overwritten.
func RegisterAuthProvider(a AuthProvider) {
	mu.Lock()
	defer mu.Unlock()
	auths[a.Name()] = a
}

// RegisterFormatter registers a custom output formatter plugin.
// If a formatter with the same name already exists, it is overwritten.
func RegisterFormatter(f OutputFormatter) {
	mu.Lock()
	defer mu.Unlock()
	formatters[f.Name()] = f
}

// GetProtocol returns a registered protocol by name.
func GetProtocol(name string) (Protocol, bool) {
	mu.RLock()
	defer mu.RUnlock()
	p, ok := protocols[name]
	return p, ok
}

// GetAuthProvider returns a registered auth provider by name.
func GetAuthProvider(name string) (AuthProvider, bool) {
	mu.RLock()
	defer mu.RUnlock()
	a, ok := auths[name]
	return a, ok
}

// GetFormatter returns a registered output formatter by name.
func GetFormatter(name string) (OutputFormatter, bool) {
	mu.RLock()
	defer mu.RUnlock()
	f, ok := formatters[name]
	return f, ok
}

// ListProtocols returns the names of all registered protocols.
func ListProtocols() []string {
	mu.RLock()
	defer mu.RUnlock()
	names := make([]string, 0, len(protocols))
	for name := range protocols {
		names = append(names, name)
	}
	return names
}

// ListAuthProviders returns the names of all registered auth providers.
func ListAuthProviders() []string {
	mu.RLock()
	defer mu.RUnlock()
	names := make([]string, 0, len(auths))
	for name := range auths {
		names = append(names, name)
	}
	return names
}

// ListFormatters returns the names of all registered output formatters.
func ListFormatters() []string {
	mu.RLock()
	defer mu.RUnlock()
	names := make([]string, 0, len(formatters))
	for name := range formatters {
		names = append(names, name)
	}
	return names
}

// ResetForTesting clears all registrations. Exported for use by other
// packages in tests; not intended for production use.
func ResetForTesting() {
	resetRegistry()
}

// resetRegistry clears all registrations (for testing).
func resetRegistry() {
	mu.Lock()
	defer mu.Unlock()
	protocols = make(map[string]Protocol)
	auths = make(map[string]AuthProvider)
	formatters = make(map[string]OutputFormatter)
}
