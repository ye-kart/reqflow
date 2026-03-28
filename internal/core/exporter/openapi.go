package exporter

import (
	"encoding/json"
	"net/url"
	"strings"

	"github.com/ye-kart/reqflow/internal/domain"
	"gopkg.in/yaml.v3"
)

// openAPISpec represents the top-level OpenAPI 3.0 document.
type openAPISpec struct {
	OpenAPI string                           `yaml:"openapi"`
	Info    openAPIInfo                      `yaml:"info"`
	Paths   map[string]map[string]openAPIOp  `yaml:"paths"`
}

type openAPIInfo struct {
	Title       string `yaml:"title"`
	Description string `yaml:"description,omitempty"`
	Version     string `yaml:"version"`
}

type openAPIOp struct {
	Summary     string                `yaml:"summary,omitempty"`
	Description string                `yaml:"description,omitempty"`
	Parameters  []openAPIParam        `yaml:"parameters,omitempty"`
	RequestBody *openAPIRequestBody   `yaml:"requestBody,omitempty"`
	Responses   map[string]openAPIResp `yaml:"responses"`
}

type openAPIParam struct {
	Name     string `yaml:"name"`
	In       string `yaml:"in"`
	Required bool   `yaml:"required,omitempty"`
	Schema   openAPISchema `yaml:"schema"`
}

type openAPIRequestBody struct {
	Content map[string]openAPIMediaType `yaml:"content"`
}

type openAPIMediaType struct {
	Schema  openAPISchema `yaml:"schema"`
	Example interface{}   `yaml:"example,omitempty"`
}

type openAPISchema struct {
	Type       string                   `yaml:"type,omitempty"`
	Properties map[string]openAPISchema `yaml:"properties,omitempty"`
}

type openAPIResp struct {
	Description string `yaml:"description"`
}

// ExportOpenAPI generates an OpenAPI 3.0 specification in YAML format
// from a collection. It extracts paths from request URLs, groups them by
// path, and infers basic schema from request body examples.
func ExportOpenAPI(c domain.Collection) ([]byte, error) {
	version := c.Version
	if version == "" {
		version = "0.1.0"
	}

	spec := openAPISpec{
		OpenAPI: "3.0.3",
		Info: openAPIInfo{
			Title:       c.Name,
			Description: c.Description,
			Version:     version,
		},
		Paths: make(map[string]map[string]openAPIOp),
	}

	// Collect all requests from collection and folders.
	allRequests := collectAllRequests(c)

	for _, r := range allRequests {
		path := extractURLPath(r.Config.URL)
		method := strings.ToLower(string(r.Config.Method))

		if spec.Paths[path] == nil {
			spec.Paths[path] = make(map[string]openAPIOp)
		}

		op := openAPIOp{
			Summary:     r.Name,
			Description: r.Description,
			Responses: map[string]openAPIResp{
				"200": {Description: "Successful response"},
			},
		}

		// Add header parameters.
		for _, h := range r.Config.Headers {
			// Skip Content-Type as it's handled by requestBody.
			if strings.EqualFold(h.Key, "Content-Type") {
				continue
			}
			op.Parameters = append(op.Parameters, openAPIParam{
				Name: h.Key,
				In:   "header",
				Schema: openAPISchema{
					Type: "string",
				},
			})
		}

		// Add request body if present.
		if len(r.Config.Body) > 0 {
			contentType := inferContentType(r.Config)
			body := &openAPIRequestBody{
				Content: map[string]openAPIMediaType{
					contentType: {
						Schema: inferSchema(r.Config.Body),
					},
				},
			}
			// Try to parse body as example.
			var example interface{}
			if json.Unmarshal(r.Config.Body, &example) == nil {
				body.Content[contentType] = openAPIMediaType{
					Schema:  inferSchema(r.Config.Body),
					Example: example,
				}
			}
			op.RequestBody = body
		}

		spec.Paths[path][method] = op
	}

	data, err := yaml.Marshal(spec)
	if err != nil {
		return nil, err
	}

	return data, nil
}

// collectAllRequests gathers requests from the collection root and all folders.
func collectAllRequests(c domain.Collection) []domain.SavedRequest {
	var all []domain.SavedRequest
	all = append(all, c.Requests...)
	for _, f := range c.Folders {
		all = append(all, collectFolderRequests(f)...)
	}
	return all
}

func collectFolderRequests(f domain.Folder) []domain.SavedRequest {
	var all []domain.SavedRequest
	all = append(all, f.Requests...)
	for _, sub := range f.Folders {
		all = append(all, collectFolderRequests(sub)...)
	}
	return all
}

// extractURLPath returns the path component of a URL.
func extractURLPath(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}
	p := u.Path
	if p == "" {
		p = "/"
	}
	return p
}

// inferContentType determines the content type from the request config.
func inferContentType(config domain.RequestConfig) string {
	if config.ContentType != "" {
		return config.ContentType
	}
	for _, h := range config.Headers {
		if strings.EqualFold(h.Key, "Content-Type") {
			return h.Value
		}
	}
	// Default to JSON if body looks like JSON.
	if len(config.Body) > 0 && config.Body[0] == '{' {
		return "application/json"
	}
	return "application/octet-stream"
}

// inferSchema tries to build a basic OpenAPI schema from a JSON body.
func inferSchema(body []byte) openAPISchema {
	var obj map[string]interface{}
	if json.Unmarshal(body, &obj) == nil {
		schema := openAPISchema{
			Type:       "object",
			Properties: make(map[string]openAPISchema),
		}
		for key, val := range obj {
			schema.Properties[key] = inferValueSchema(val)
		}
		return schema
	}
	return openAPISchema{Type: "string"}
}

func inferValueSchema(val interface{}) openAPISchema {
	switch val.(type) {
	case float64:
		return openAPISchema{Type: "number"}
	case bool:
		return openAPISchema{Type: "boolean"}
	case string:
		return openAPISchema{Type: "string"}
	case []interface{}:
		return openAPISchema{Type: "array"}
	case map[string]interface{}:
		return openAPISchema{Type: "object"}
	default:
		return openAPISchema{Type: "string"}
	}
}
