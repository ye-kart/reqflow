package importer

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/ye-kart/reqflow/internal/domain"
	"gopkg.in/yaml.v3"
)

// openAPIDoc represents the top-level OpenAPI 3.0 document for import.
type openAPIDoc struct {
	OpenAPI    string                            `json:"openapi"    yaml:"openapi"`
	Info       openAPIDocInfo                    `json:"info"       yaml:"info"`
	Servers    []openAPIServer                   `json:"servers"    yaml:"servers"`
	Paths      map[string]map[string]openAPIDocOp `json:"paths"      yaml:"paths"`
	Components *openAPIComponents                `json:"components" yaml:"components"`
	Security   []map[string][]string             `json:"security"   yaml:"security"`
}

type openAPIDocInfo struct {
	Title   string `json:"title"   yaml:"title"`
	Version string `json:"version" yaml:"version"`
}

type openAPIServer struct {
	URL string `json:"url" yaml:"url"`
}

type openAPIDocOp struct {
	Summary     string `json:"summary"     yaml:"summary"`
	OperationID string `json:"operationId" yaml:"operationId"`
}

type openAPIComponents struct {
	SecuritySchemes map[string]openAPISecurityScheme `json:"securitySchemes" yaml:"securitySchemes"`
}

type openAPISecurityScheme struct {
	Type   string `json:"type"   yaml:"type"`
	Scheme string `json:"scheme" yaml:"scheme"`
	Name   string `json:"name"   yaml:"name"`
	In     string `json:"in"     yaml:"in"`
}

// ParseOpenAPI parses an OpenAPI 3.0 specification (JSON or YAML) into a
// domain.Collection. Each path + method combination becomes a request.
func ParseOpenAPI(data []byte) (domain.Collection, error) {
	var doc openAPIDoc

	// Try JSON first, then YAML.
	if err := json.Unmarshal(data, &doc); err != nil {
		if err2 := yaml.Unmarshal(data, &doc); err2 != nil {
			return domain.Collection{}, fmt.Errorf("failed to parse OpenAPI spec as JSON or YAML: JSON=%w, YAML=%v", err, err2)
		}
	}

	col := domain.Collection{
		Name:    doc.Info.Title,
		Version: doc.Info.Version,
	}

	// Extract base URL from the first server.
	baseURL := ""
	if len(doc.Servers) > 0 {
		baseURL = strings.TrimRight(doc.Servers[0].URL, "/")
		col.Variables = append(col.Variables, domain.Variable{
			Key:   "base_url",
			Value: doc.Servers[0].URL,
			Scope: domain.ScopeCollection,
		})
	}

	// Extract auth from security schemes.
	if doc.Components != nil && len(doc.Security) > 0 {
		col.Auth = resolveSecuritySchemes(doc.Security, doc.Components.SecuritySchemes)
	}

	// Sort paths for deterministic output.
	paths := make([]string, 0, len(doc.Paths))
	for p := range doc.Paths {
		paths = append(paths, p)
	}
	sort.Strings(paths)

	for _, path := range paths {
		methods := doc.Paths[path]

		// Sort methods for deterministic output.
		methodNames := make([]string, 0, len(methods))
		for m := range methods {
			methodNames = append(methodNames, m)
		}
		sort.Strings(methodNames)

		for _, method := range methodNames {
			op := methods[method]
			reqURL := path
			if baseURL != "" {
				reqURL = baseURL + path
			}

			name := op.Summary
			if name == "" {
				name = op.OperationID
			}
			if name == "" {
				name = strings.ToUpper(method) + " " + path
			}

			col.Requests = append(col.Requests, domain.SavedRequest{
				Name: name,
				Config: domain.RequestConfig{
					Method: domain.HTTPMethod(strings.ToUpper(method)),
					URL:    reqURL,
				},
			})
		}
	}

	return col, nil
}

func resolveSecuritySchemes(security []map[string][]string, schemes map[string]openAPISecurityScheme) *domain.AuthConfig {
	if len(security) == 0 || len(schemes) == 0 {
		return nil
	}

	// Use the first security requirement.
	for name := range security[0] {
		scheme, ok := schemes[name]
		if !ok {
			continue
		}

		switch {
		case scheme.Type == "http" && scheme.Scheme == "bearer":
			return &domain.AuthConfig{
				Type:   domain.AuthBearer,
				Bearer: &domain.BearerAuthConfig{},
			}
		case scheme.Type == "http" && scheme.Scheme == "basic":
			return &domain.AuthConfig{
				Type:  domain.AuthBasic,
				Basic: &domain.BasicAuthConfig{},
			}
		case scheme.Type == "apiKey":
			loc := domain.APIKeyInHeader
			if scheme.In == "query" {
				loc = domain.APIKeyInQuery
			}
			return &domain.AuthConfig{
				Type: domain.AuthAPIKey,
				APIKey: &domain.APIKeyAuthConfig{
					Key:      scheme.Name,
					Location: loc,
				},
			}
		}
	}

	return nil
}
