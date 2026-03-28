package importer

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/ye-kart/reqflow/internal/domain"
)

// insomniaExport represents an Insomnia v4 JSON export document.
type insomniaExport struct {
	Type         string             `json:"_type"`
	ExportFormat int                `json:"__export_format"`
	Resources    []insomniaResource `json:"resources"`
}

type insomniaResource struct {
	ID             string            `json:"_id"`
	Type           string            `json:"_type"`
	ParentID       string            `json:"parentId"`
	Name           string            `json:"name"`
	Method         string            `json:"method,omitempty"`
	URL            string            `json:"url,omitempty"`
	Headers        []insomniaHeader  `json:"headers,omitempty"`
	Body           *insomniaBody     `json:"body,omitempty"`
	Authentication *insomniaAuth     `json:"authentication,omitempty"`
}

type insomniaHeader struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type insomniaBody struct {
	MimeType string `json:"mimeType"`
	Text     string `json:"text"`
}

type insomniaAuth struct {
	Type     string `json:"type"`
	Token    string `json:"token,omitempty"`
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
}

// ParseInsomnia parses an Insomnia v4 JSON export into a domain.Collection.
// Resources of type "request" become requests; "request_group" become folders.
func ParseInsomnia(data []byte) (domain.Collection, error) {
	var export insomniaExport
	if err := json.Unmarshal(data, &export); err != nil {
		return domain.Collection{}, fmt.Errorf("invalid Insomnia JSON: %w", err)
	}

	// Index resources by ID.
	byID := make(map[string]*insomniaResource, len(export.Resources))
	for i := range export.Resources {
		byID[export.Resources[i].ID] = &export.Resources[i]
	}

	// Find the workspace name.
	workspaceName := "Insomnia Import"
	workspaceIDs := make(map[string]bool)
	for _, r := range export.Resources {
		if r.Type == "workspace" {
			workspaceName = r.Name
			workspaceIDs[r.ID] = true
		}
	}

	// Build a tree: map parentID -> children.
	children := make(map[string][]insomniaResource)
	for _, r := range export.Resources {
		if r.Type == "request" || r.Type == "request_group" {
			children[r.ParentID] = append(children[r.ParentID], r)
		}
	}

	col := domain.Collection{
		Name: workspaceName,
	}

	// Populate from workspace roots.
	rootParents := make([]string, 0)
	for id := range workspaceIDs {
		rootParents = append(rootParents, id)
	}

	// If no workspace found, collect orphan requests under the root.
	if len(rootParents) == 0 {
		// Find all unique parent IDs that don't correspond to known resources.
		for parentID := range children {
			if _, exists := byID[parentID]; !exists {
				rootParents = append(rootParents, parentID)
			}
		}
	}

	for _, rootID := range rootParents {
		buildInsomniaTree(&col, children, rootID)
	}

	return col, nil
}

func buildInsomniaTree(col *domain.Collection, children map[string][]insomniaResource, parentID string) {
	for _, r := range children[parentID] {
		switch r.Type {
		case "request":
			col.Requests = append(col.Requests, convertInsomniaRequest(r))
		case "request_group":
			folder := buildInsomniaFolder(r, children)
			col.Folders = append(col.Folders, folder)
		}
	}
}

func buildInsomniaFolder(r insomniaResource, children map[string][]insomniaResource) domain.Folder {
	folder := domain.Folder{
		Name: r.Name,
	}

	for _, child := range children[r.ID] {
		switch child.Type {
		case "request":
			folder.Requests = append(folder.Requests, convertInsomniaRequest(child))
		case "request_group":
			folder.Folders = append(folder.Folders, buildInsomniaFolder(child, children))
		}
	}

	return folder
}

func convertInsomniaRequest(r insomniaResource) domain.SavedRequest {
	config := domain.RequestConfig{
		Method: domain.HTTPMethod(strings.ToUpper(r.Method)),
		URL:    r.URL,
	}

	for _, h := range r.Headers {
		config.Headers = append(config.Headers, domain.Header{Key: h.Name, Value: h.Value})
	}

	if r.Body != nil && r.Body.Text != "" {
		config.Body = []byte(r.Body.Text)
		config.ContentType = r.Body.MimeType
	}

	if r.Authentication != nil {
		config.Auth = convertInsomniaAuth(r.Authentication)
	}

	return domain.SavedRequest{
		Name:   r.Name,
		Config: config,
	}
}

func convertInsomniaAuth(a *insomniaAuth) *domain.AuthConfig {
	if a == nil {
		return nil
	}

	switch a.Type {
	case "bearer":
		return &domain.AuthConfig{
			Type: domain.AuthBearer,
			Bearer: &domain.BearerAuthConfig{
				Token: a.Token,
			},
		}
	case "basic":
		return &domain.AuthConfig{
			Type: domain.AuthBasic,
			Basic: &domain.BasicAuthConfig{
				Username: a.Username,
				Password: a.Password,
			},
		}
	default:
		return nil
	}
}
