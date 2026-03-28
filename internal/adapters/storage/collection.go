package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/ye-kart/reqflow/internal/domain"
	"gopkg.in/yaml.v3"
)

// collectionFile is the YAML-serializable representation of a collection file.
type collectionFile struct {
	Name        string                `yaml:"name"`
	Description string                `yaml:"description,omitempty"`
	Version     string                `yaml:"version,omitempty"`
	Variables   map[string]string     `yaml:"variables,omitempty"`
	Auth        *authFile             `yaml:"auth,omitempty"`
	Headers     []headerFile          `yaml:"headers,omitempty"`
	Folders     []folderFile          `yaml:"folders,omitempty"`
	Requests    []savedRequestFile    `yaml:"requests,omitempty"`
}

// folderFile is the YAML-serializable representation of a folder.
type folderFile struct {
	Name        string             `yaml:"name"`
	Description string             `yaml:"description,omitempty"`
	Variables   map[string]string  `yaml:"variables,omitempty"`
	Auth        *authFile          `yaml:"auth,omitempty"`
	Headers     []headerFile       `yaml:"headers,omitempty"`
	Folders     []folderFile       `yaml:"folders,omitempty"`
	Requests    []savedRequestFile `yaml:"requests,omitempty"`
}

// savedRequestFile is the YAML-serializable representation of a saved request.
type savedRequestFile struct {
	Name        string               `yaml:"name"`
	Description string               `yaml:"description,omitempty"`
	Method      string               `yaml:"method"`
	URL         string               `yaml:"url"`
	Body        string               `yaml:"body,omitempty"`
	ContentType string               `yaml:"content_type,omitempty"`
	Response    *exampleResponseFile `yaml:"response,omitempty"`
}

// exampleResponseFile is the YAML-serializable representation of an example response.
type exampleResponseFile struct {
	StatusCode int          `yaml:"status_code"`
	Headers    []headerFile `yaml:"headers,omitempty"`
	Body       string       `yaml:"body,omitempty"`
}

// headerFile is the YAML-serializable representation of a header.
type headerFile struct {
	Key   string `yaml:"key"`
	Value string `yaml:"value"`
}

// authFile is the YAML-serializable representation of an auth config.
type authFile struct {
	Type   string          `yaml:"type"`
	Basic  *basicAuthFile  `yaml:"basic,omitempty"`
	Bearer *bearerAuthFile `yaml:"bearer,omitempty"`
	APIKey *apiKeyAuthFile `yaml:"apikey,omitempty"`
}

type basicAuthFile struct {
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

type bearerAuthFile struct {
	Token  string `yaml:"token"`
	Prefix string `yaml:"prefix,omitempty"`
}

type apiKeyAuthFile struct {
	Key      string `yaml:"key"`
	Value    string `yaml:"value"`
	Location string `yaml:"location,omitempty"`
}

// ReadCollection parses a YAML collection file and returns a domain.Collection.
func (f *Filesystem) ReadCollection(path string) (domain.Collection, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return domain.Collection{}, fmt.Errorf("reading collection file: %w", err)
	}

	var cf collectionFile
	if err := yaml.Unmarshal(data, &cf); err != nil {
		return domain.Collection{}, fmt.Errorf("parsing collection YAML: %w", err)
	}

	return collectionFromFile(cf), nil
}

// WriteCollection writes a domain.Collection to a YAML file.
func (f *Filesystem) WriteCollection(path string, c domain.Collection) error {
	cf := collectionToFile(c)

	data, err := yaml.Marshal(&cf)
	if err != nil {
		return fmt.Errorf("marshaling collection YAML: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("creating directory: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("writing collection file: %w", err)
	}

	return nil
}

// ListCollections returns the names (without .yaml extension) of all collection
// files in the given directory.
func (f *Filesystem) ListCollections(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("reading directory: %w", err)
	}

	var names []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasSuffix(name, ".yaml") || strings.HasSuffix(name, ".yml") {
			names = append(names, strings.TrimSuffix(strings.TrimSuffix(name, ".yaml"), ".yml"))
		}
	}

	return names, nil
}

// --- conversion helpers ---

func collectionFromFile(cf collectionFile) domain.Collection {
	col := domain.Collection{
		Name:        cf.Name,
		Description: cf.Description,
		Version:     cf.Version,
		Auth:        authFromFile(cf.Auth),
		Headers:     headersFromFile(cf.Headers),
		Variables:   variablesFromMap(cf.Variables, domain.ScopeCollection),
		Requests:    requestsFromFile(cf.Requests),
		Folders:     foldersFromFile(cf.Folders),
	}
	return col
}

func collectionToFile(c domain.Collection) collectionFile {
	return collectionFile{
		Name:        c.Name,
		Description: c.Description,
		Version:     c.Version,
		Auth:        authToFile(c.Auth),
		Headers:     headersToFile(c.Headers),
		Variables:   variablesToMap(c.Variables),
		Requests:    requestsToFile(c.Requests),
		Folders:     foldersToFile(c.Folders),
	}
}

func foldersFromFile(ffs []folderFile) []domain.Folder {
	if len(ffs) == 0 {
		return nil
	}
	folders := make([]domain.Folder, len(ffs))
	for i, ff := range ffs {
		folders[i] = domain.Folder{
			Name:        ff.Name,
			Description: ff.Description,
			Auth:        authFromFile(ff.Auth),
			Headers:     headersFromFile(ff.Headers),
			Variables:   variablesFromMap(ff.Variables, domain.ScopeCollection),
			Requests:    requestsFromFile(ff.Requests),
			Folders:     foldersFromFile(ff.Folders),
		}
	}
	return folders
}

func foldersToFile(folders []domain.Folder) []folderFile {
	if len(folders) == 0 {
		return nil
	}
	ffs := make([]folderFile, len(folders))
	for i, f := range folders {
		ffs[i] = folderFile{
			Name:        f.Name,
			Description: f.Description,
			Auth:        authToFile(f.Auth),
			Headers:     headersToFile(f.Headers),
			Variables:   variablesToMap(f.Variables),
			Requests:    requestsToFile(f.Requests),
			Folders:     foldersToFile(f.Folders),
		}
	}
	return ffs
}

func requestsFromFile(rfs []savedRequestFile) []domain.SavedRequest {
	if len(rfs) == 0 {
		return nil
	}
	requests := make([]domain.SavedRequest, len(rfs))
	for i, rf := range rfs {
		var body []byte
		if rf.Body != "" {
			body = []byte(rf.Body)
		}
		requests[i] = domain.SavedRequest{
			Name:        rf.Name,
			Description: rf.Description,
			Config: domain.RequestConfig{
				Method:      domain.HTTPMethod(rf.Method),
				URL:         rf.URL,
				Body:        body,
				ContentType: rf.ContentType,
			},
			Response: exampleResponseFromFile(rf.Response),
		}
	}
	return requests
}

func requestsToFile(requests []domain.SavedRequest) []savedRequestFile {
	if len(requests) == 0 {
		return nil
	}
	rfs := make([]savedRequestFile, len(requests))
	for i, r := range requests {
		rfs[i] = savedRequestFile{
			Name:        r.Name,
			Description: r.Description,
			Method:      string(r.Config.Method),
			URL:         r.Config.URL,
			Body:        string(r.Config.Body),
			ContentType: r.Config.ContentType,
			Response:    exampleResponseToFile(r.Response),
		}
	}
	return rfs
}

func exampleResponseFromFile(rf *exampleResponseFile) *domain.ExampleResponse {
	if rf == nil {
		return nil
	}
	return &domain.ExampleResponse{
		StatusCode: rf.StatusCode,
		Headers:    headersFromFile(rf.Headers),
		Body:       rf.Body,
	}
}

func exampleResponseToFile(resp *domain.ExampleResponse) *exampleResponseFile {
	if resp == nil {
		return nil
	}
	return &exampleResponseFile{
		StatusCode: resp.StatusCode,
		Headers:    headersToFile(resp.Headers),
		Body:       resp.Body,
	}
}

func headersFromFile(hfs []headerFile) []domain.Header {
	if len(hfs) == 0 {
		return nil
	}
	headers := make([]domain.Header, len(hfs))
	for i, hf := range hfs {
		headers[i] = domain.Header{Key: hf.Key, Value: hf.Value}
	}
	return headers
}

func headersToFile(headers []domain.Header) []headerFile {
	if len(headers) == 0 {
		return nil
	}
	hfs := make([]headerFile, len(headers))
	for i, h := range headers {
		hfs[i] = headerFile{Key: h.Key, Value: h.Value}
	}
	return hfs
}

func authFromFile(af *authFile) *domain.AuthConfig {
	if af == nil {
		return nil
	}
	auth := &domain.AuthConfig{
		Type: domain.AuthType(af.Type),
	}
	if af.Basic != nil {
		auth.Basic = &domain.BasicAuthConfig{
			Username: af.Basic.Username,
			Password: af.Basic.Password,
		}
	}
	if af.Bearer != nil {
		auth.Bearer = &domain.BearerAuthConfig{
			Token:  af.Bearer.Token,
			Prefix: af.Bearer.Prefix,
		}
	}
	if af.APIKey != nil {
		auth.APIKey = &domain.APIKeyAuthConfig{
			Key:      af.APIKey.Key,
			Value:    af.APIKey.Value,
			Location: domain.APIKeyLocation(af.APIKey.Location),
		}
	}
	return auth
}

func authToFile(auth *domain.AuthConfig) *authFile {
	if auth == nil {
		return nil
	}
	af := &authFile{
		Type: string(auth.Type),
	}
	if auth.Basic != nil {
		af.Basic = &basicAuthFile{
			Username: auth.Basic.Username,
			Password: auth.Basic.Password,
		}
	}
	if auth.Bearer != nil {
		af.Bearer = &bearerAuthFile{
			Token:  auth.Bearer.Token,
			Prefix: auth.Bearer.Prefix,
		}
	}
	if auth.APIKey != nil {
		af.APIKey = &apiKeyAuthFile{
			Key:      auth.APIKey.Key,
			Value:    auth.APIKey.Value,
			Location: string(auth.APIKey.Location),
		}
	}
	return af
}

func variablesFromMap(m map[string]string, scope domain.VariableScope) []domain.Variable {
	if len(m) == 0 {
		return nil
	}
	// Sort keys for deterministic ordering.
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	vars := make([]domain.Variable, len(keys))
	for i, k := range keys {
		vars[i] = domain.Variable{Key: k, Value: m[k], Scope: scope}
	}
	return vars
}

func variablesToMap(vars []domain.Variable) map[string]string {
	if len(vars) == 0 {
		return nil
	}
	m := make(map[string]string, len(vars))
	for _, v := range vars {
		m[v.Key] = v.Value
	}
	return m
}
