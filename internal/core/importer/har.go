package importer

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/ye-kart/reqflow/internal/domain"
)

// harArchive represents the top-level HAR (HTTP Archive) 1.2 document.
type harArchive struct {
	Log harLog `json:"log"`
}

type harLog struct {
	Version string     `json:"version"`
	Entries []harEntry `json:"entries"`
}

type harEntry struct {
	Request harRequest `json:"request"`
}

type harRequest struct {
	Method   string      `json:"method"`
	URL      string      `json:"url"`
	Headers  []harHeader `json:"headers"`
	PostData *harPost    `json:"postData,omitempty"`
}

type harHeader struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type harPost struct {
	MimeType string `json:"mimeType"`
	Text     string `json:"text"`
}

// ParseHAR parses an HTTP Archive (HAR) 1.2 JSON document into a
// domain.Collection. Each entry becomes a separate request.
func ParseHAR(data []byte) (domain.Collection, error) {
	var archive harArchive
	if err := json.Unmarshal(data, &archive); err != nil {
		return domain.Collection{}, fmt.Errorf("invalid HAR JSON: %w", err)
	}

	col := domain.Collection{
		Name: "HAR Import",
	}

	for _, entry := range archive.Log.Entries {
		r := entry.Request

		config := domain.RequestConfig{
			Method: domain.HTTPMethod(strings.ToUpper(r.Method)),
			URL:    r.URL,
		}

		for _, h := range r.Headers {
			config.Headers = append(config.Headers, domain.Header{Key: h.Name, Value: h.Value})
		}

		if r.PostData != nil && r.PostData.Text != "" {
			config.Body = []byte(r.PostData.Text)
			config.ContentType = r.PostData.MimeType
		}

		name := deriveRequestName(r.Method, r.URL)

		col.Requests = append(col.Requests, domain.SavedRequest{
			Name:   name,
			Config: config,
		})
	}

	return col, nil
}

// deriveRequestName generates a human-readable name from method and URL.
func deriveRequestName(method, rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return strings.ToUpper(method) + " " + rawURL
	}
	path := u.Path
	if path == "" {
		path = "/"
	}
	return strings.ToUpper(method) + " " + path
}
