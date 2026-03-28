package importer

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/ye-kart/reqflow/internal/domain"
)

// postmanCollection represents a Postman Collection v2.1 document.
type postmanCollection struct {
	Info     postmanInfo      `json:"info"`
	Item     []postmanItem    `json:"item"`
	Variable []postmanVar     `json:"variable,omitempty"`
	Auth     *postmanAuth     `json:"auth,omitempty"`
}

type postmanInfo struct {
	Name   string `json:"name"`
	Schema string `json:"schema,omitempty"`
}

type postmanItem struct {
	Name    string          `json:"name"`
	Request *postmanRequest `json:"request,omitempty"`
	Item    []postmanItem   `json:"item,omitempty"`
}

type postmanRequest struct {
	Method string          `json:"method"`
	URL    json.RawMessage `json:"url"`
	Header []postmanKV     `json:"header,omitempty"`
	Body   *postmanBody    `json:"body,omitempty"`
	Auth   *postmanAuth    `json:"auth,omitempty"`
}

type postmanURL struct {
	Raw string `json:"raw"`
}

type postmanKV struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type postmanBody struct {
	Mode string `json:"mode"`
	Raw  string `json:"raw,omitempty"`
}

type postmanAuth struct {
	Type   string      `json:"type"`
	Basic  []postmanKV `json:"basic,omitempty"`
	Bearer []postmanKV `json:"bearer,omitempty"`
	APIKey []postmanKV `json:"apikey,omitempty"`
}

type postmanVar struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// ParsePostmanCollection parses a Postman Collection v2.1 JSON document into
// a domain.Collection.
func ParsePostmanCollection(data []byte) (domain.Collection, error) {
	var pc postmanCollection
	if err := json.Unmarshal(data, &pc); err != nil {
		return domain.Collection{}, fmt.Errorf("invalid Postman collection JSON: %w", err)
	}

	col := domain.Collection{
		Name: pc.Info.Name,
	}

	// Parse variables.
	for _, v := range pc.Variable {
		col.Variables = append(col.Variables, domain.Variable{
			Key:   v.Key,
			Value: v.Value,
			Scope: domain.ScopeCollection,
		})
	}

	// Parse collection-level auth.
	if pc.Auth != nil {
		col.Auth = convertPostmanAuth(pc.Auth)
	}

	// Parse items (requests and folders).
	for _, item := range pc.Item {
		if item.Request != nil {
			req, err := convertPostmanRequest(item)
			if err != nil {
				return domain.Collection{}, err
			}
			col.Requests = append(col.Requests, req)
		} else if len(item.Item) > 0 {
			folder, err := convertPostmanFolder(item)
			if err != nil {
				return domain.Collection{}, err
			}
			col.Folders = append(col.Folders, folder)
		}
	}

	return col, nil
}

func convertPostmanFolder(item postmanItem) (domain.Folder, error) {
	folder := domain.Folder{
		Name: item.Name,
	}

	for _, sub := range item.Item {
		if sub.Request != nil {
			req, err := convertPostmanRequest(sub)
			if err != nil {
				return domain.Folder{}, err
			}
			folder.Requests = append(folder.Requests, req)
		} else if len(sub.Item) > 0 {
			subfolder, err := convertPostmanFolder(sub)
			if err != nil {
				return domain.Folder{}, err
			}
			folder.Folders = append(folder.Folders, subfolder)
		}
	}

	return folder, nil
}

func convertPostmanRequest(item postmanItem) (domain.SavedRequest, error) {
	r := item.Request

	url, err := extractPostmanURL(r.URL)
	if err != nil {
		return domain.SavedRequest{}, fmt.Errorf("parsing URL for %q: %w", item.Name, err)
	}

	config := domain.RequestConfig{
		Method: domain.HTTPMethod(strings.ToUpper(r.Method)),
		URL:    url,
	}

	// Headers.
	for _, h := range r.Header {
		config.Headers = append(config.Headers, domain.Header{Key: h.Key, Value: h.Value})
	}

	// Body.
	if r.Body != nil && r.Body.Mode == "raw" && r.Body.Raw != "" {
		config.Body = []byte(r.Body.Raw)
	}

	// Auth.
	if r.Auth != nil {
		config.Auth = convertPostmanAuth(r.Auth)
	}

	return domain.SavedRequest{
		Name:   item.Name,
		Config: config,
	}, nil
}

// extractPostmanURL handles both string and object URL formats.
func extractPostmanURL(raw json.RawMessage) (string, error) {
	if len(raw) == 0 {
		return "", fmt.Errorf("empty URL")
	}

	// Try string first.
	var s string
	if json.Unmarshal(raw, &s) == nil {
		return s, nil
	}

	// Try object.
	var obj postmanURL
	if err := json.Unmarshal(raw, &obj); err != nil {
		return "", fmt.Errorf("invalid URL format: %w", err)
	}
	return obj.Raw, nil
}

func convertPostmanAuth(a *postmanAuth) *domain.AuthConfig {
	if a == nil {
		return nil
	}

	switch a.Type {
	case "basic":
		kvMap := postmanKVMap(a.Basic)
		return &domain.AuthConfig{
			Type: domain.AuthBasic,
			Basic: &domain.BasicAuthConfig{
				Username: kvMap["username"],
				Password: kvMap["password"],
			},
		}
	case "bearer":
		kvMap := postmanKVMap(a.Bearer)
		return &domain.AuthConfig{
			Type: domain.AuthBearer,
			Bearer: &domain.BearerAuthConfig{
				Token: kvMap["token"],
			},
		}
	case "apikey":
		kvMap := postmanKVMap(a.APIKey)
		loc := domain.APIKeyInHeader
		if kvMap["in"] == "query" {
			loc = domain.APIKeyInQuery
		}
		return &domain.AuthConfig{
			Type: domain.AuthAPIKey,
			APIKey: &domain.APIKeyAuthConfig{
				Key:      kvMap["key"],
				Value:    kvMap["value"],
				Location: loc,
			},
		}
	default:
		return nil
	}
}

func postmanKVMap(kvs []postmanKV) map[string]string {
	m := make(map[string]string, len(kvs))
	for _, kv := range kvs {
		m[kv.Key] = kv.Value
	}
	return m
}

// PostmanCollectionFromDomain converts a domain.Collection to the internal
// Postman structure for serialization (used by the exporter and round-trip tests).
func PostmanCollectionFromDomain(c domain.Collection) postmanCollection {
	pc := postmanCollection{
		Info: postmanInfo{
			Name:   c.Name,
			Schema: "https://schema.getpostman.com/json/collection/v2.1.0/collection.json",
		},
	}

	for _, v := range c.Variables {
		pc.Variable = append(pc.Variable, postmanVar{Key: v.Key, Value: v.Value})
	}

	for _, r := range c.Requests {
		pc.Item = append(pc.Item, postmanItemFromRequest(r))
	}
	for _, f := range c.Folders {
		pc.Item = append(pc.Item, postmanItemFromFolder(f))
	}

	return pc
}

func postmanItemFromRequest(r domain.SavedRequest) postmanItem {
	urlBytes, _ := json.Marshal(postmanURL{Raw: r.Config.URL})

	pr := &postmanRequest{
		Method: string(r.Config.Method),
		URL:    json.RawMessage(urlBytes),
	}

	for _, h := range r.Config.Headers {
		pr.Header = append(pr.Header, postmanKV{Key: h.Key, Value: h.Value})
	}

	if len(r.Config.Body) > 0 {
		pr.Body = &postmanBody{
			Mode: "raw",
			Raw:  string(r.Config.Body),
		}
	}

	if r.Config.Auth != nil {
		pr.Auth = domainAuthToPostman(r.Config.Auth)
	}

	return postmanItem{
		Name:    r.Name,
		Request: pr,
	}
}

func postmanItemFromFolder(f domain.Folder) postmanItem {
	pi := postmanItem{Name: f.Name}
	for _, r := range f.Requests {
		pi.Item = append(pi.Item, postmanItemFromRequest(r))
	}
	for _, sub := range f.Folders {
		pi.Item = append(pi.Item, postmanItemFromFolder(sub))
	}
	return pi
}

func domainAuthToPostman(auth *domain.AuthConfig) *postmanAuth {
	if auth == nil {
		return nil
	}

	switch auth.Type {
	case domain.AuthBasic:
		if auth.Basic == nil {
			return nil
		}
		return &postmanAuth{
			Type: "basic",
			Basic: []postmanKV{
				{Key: "username", Value: auth.Basic.Username},
				{Key: "password", Value: auth.Basic.Password},
			},
		}
	case domain.AuthBearer:
		if auth.Bearer == nil {
			return nil
		}
		return &postmanAuth{
			Type: "bearer",
			Bearer: []postmanKV{
				{Key: "token", Value: auth.Bearer.Token},
			},
		}
	case domain.AuthAPIKey:
		if auth.APIKey == nil {
			return nil
		}
		in := "header"
		if auth.APIKey.Location == domain.APIKeyInQuery {
			in = "query"
		}
		return &postmanAuth{
			Type: "apikey",
			APIKey: []postmanKV{
				{Key: "key", Value: auth.APIKey.Key},
				{Key: "value", Value: auth.APIKey.Value},
				{Key: "in", Value: in},
			},
		}
	default:
		return nil
	}
}
