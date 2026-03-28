package mockserver

import (
	"net/url"
	"strings"

	"github.com/ye-kart/reqflow/internal/domain"
)

// route is an internal representation of a matchable request route.
type route struct {
	method   string
	pattern  string // URL path pattern, e.g. "/users/{{id}}"
	response *domain.ExampleResponse
}

// buildRoutes converts a slice of SavedRequests into internal route structs.
func buildRoutes(requests []domain.SavedRequest) []route {
	routes := make([]route, 0, len(requests))
	for _, req := range requests {
		if req.Response == nil {
			continue
		}
		routes = append(routes, route{
			method:   string(req.Config.Method),
			pattern:  extractPath(req.Config.URL),
			response: req.Response,
		})
	}
	return routes
}

// matchRoute finds the first route matching the given method and path.
// Returns nil if no match is found.
func matchRoute(method, path string, routes []route) *domain.ExampleResponse {
	for _, r := range routes {
		if !strings.EqualFold(r.method, method) {
			continue
		}
		if pathMatches(r.pattern, path) {
			return r.response
		}
	}
	return nil
}

// extractPath extracts the path component from a URL string.
// If the URL cannot be parsed, it is returned as-is.
func extractPath(rawURL string) string {
	// Handle plain paths that don't have a scheme.
	if strings.HasPrefix(rawURL, "/") {
		return rawURL
	}
	u, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}
	p := u.Path
	if p == "" {
		return "/"
	}
	return p
}

// pathMatches checks whether a URL path matches a pattern that may contain
// template parameters like {{id}}. Each {{...}} segment matches exactly one
// non-empty path segment.
func pathMatches(pattern, path string) bool {
	patternParts := strings.Split(strings.Trim(pattern, "/"), "/")
	pathParts := strings.Split(strings.Trim(path, "/"), "/")

	if len(patternParts) != len(pathParts) {
		return false
	}

	for i, pp := range patternParts {
		if isTemplateParam(pp) {
			// Template param matches any non-empty segment.
			if pathParts[i] == "" {
				return false
			}
			continue
		}
		if pp != pathParts[i] {
			return false
		}
	}
	return true
}

// isTemplateParam returns true if the segment is a {{...}} template parameter.
func isTemplateParam(segment string) bool {
	return strings.HasPrefix(segment, "{{") && strings.HasSuffix(segment, "}}")
}
