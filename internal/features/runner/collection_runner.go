package runner

import (
	"context"
	"time"

	"github.com/ye-kart/reqflow/internal/core/auth"
	"github.com/ye-kart/reqflow/internal/core/variable"
	"github.com/ye-kart/reqflow/internal/domain"
	"github.com/ye-kart/reqflow/internal/ports/driven"
)

// CollectionRunner executes all requests in a collection sequentially.
type CollectionRunner struct {
	httpClient driven.HTTPClient
}

// NewCollectionRunner creates a new CollectionRunner with the given HTTP client.
func NewCollectionRunner(hc driven.HTTPClient) *CollectionRunner {
	return &CollectionRunner{httpClient: hc}
}

// flatRequest pairs a SavedRequest with its folder path and effective config
// (auth, headers, variables) after merging collection and folder defaults.
type flatRequest struct {
	request    domain.SavedRequest
	folderPath string
	auth       *domain.AuthConfig
	headers    []domain.Header
	vars       []domain.Variable
}

// RunCollection executes all (or filtered) requests in a collection and returns
// aggregate results.
func (cr *CollectionRunner) RunCollection(ctx context.Context, collection domain.Collection, opts domain.CollectionRunOptions) (domain.CollectionRunResult, error) {
	start := time.Now()

	result := domain.CollectionRunResult{
		CollectionName: collection.Name,
	}

	// Flatten the collection into a list of requests with resolved config.
	requests := flattenCollection(collection, opts.FolderName)
	result.TotalRequests = len(requests)

	// Merge variable layers: collection vars < opts.Vars (highest precedence).
	baseVars := variable.Resolve(collection.Variables)
	for k, v := range opts.Vars {
		baseVars[k] = v
	}

	for i, fr := range requests {
		select {
		case <-ctx.Done():
			result.Skipped += len(requests) - i
			break
		default:
		}

		// Apply delay between requests (not before the first one).
		if i > 0 && opts.Delay > 0 {
			time.Sleep(opts.Delay)
		}

		// Merge folder-level vars on top of base vars.
		reqVars := make(map[string]string, len(baseVars))
		for k, v := range baseVars {
			reqVars[k] = v
		}
		for _, v := range fr.vars {
			reqVars[v.Key] = v.Value
		}

		rr := cr.executeRequest(ctx, fr, reqVars)
		result.Results = append(result.Results, rr)

		if rr.Passed {
			result.Passed++
		} else {
			result.Failed++
			if opts.StopOnFailure {
				result.Skipped += len(requests) - i - 1
				break
			}
		}
	}

	result.Duration = time.Since(start)
	return result, nil
}

// executeRequest sends a single SavedRequest and returns its result.
func (cr *CollectionRunner) executeRequest(ctx context.Context, fr flatRequest, vars map[string]string) domain.RequestRunResult {
	start := time.Now()

	rr := domain.RequestRunResult{
		RequestName: fr.request.Name,
		FolderPath:  fr.folderPath,
	}

	// Build the config: start with request config, layer on inherited headers.
	config := fr.request.Config

	// Merge headers: inherited headers first, then request-specific headers.
	merged := mergeHeaders(fr.headers, config.Headers)
	config.Headers = merged

	// Apply auth: request > folder > collection (already resolved in flatRequest).
	if config.Auth == nil {
		config.Auth = fr.auth
	}

	// Interpolate URL and build request.
	config.URL = variable.Interpolate(config.URL, vars)

	req := domain.HTTPRequest{
		Method:      config.Method,
		URL:         config.URL,
		Headers:     config.Headers,
		Body:        config.Body,
		ContentType: config.ContentType,
	}

	// Apply auth to the request.
	if config.Auth != nil {
		var err error
		req, err = auth.Apply(req, config.Auth)
		if err != nil {
			rr.Error = err
			rr.Duration = time.Since(start)
			return rr
		}
	}

	resp, err := cr.httpClient.Do(ctx, req)
	if err != nil {
		rr.Error = err
		rr.Duration = time.Since(start)
		return rr
	}

	rr.Response = resp
	rr.Duration = time.Since(start)
	rr.Passed = resp.StatusCode >= 200 && resp.StatusCode < 400

	return rr
}

// flattenCollection walks the collection tree and returns a flat list of
// requests with their inherited config. If folderName is non-empty, only
// requests in that folder (top-level match by name) are returned.
func flattenCollection(col domain.Collection, folderName string) []flatRequest {
	var requests []flatRequest

	if folderName == "" {
		// Root-level requests.
		for _, r := range col.Requests {
			requests = append(requests, flatRequest{
				request:    r,
				folderPath: "",
				auth:       col.Auth,
				headers:    col.Headers,
			})
		}
		// Walk all folders.
		for _, f := range col.Folders {
			requests = append(requests, flattenFolder(f, "", col.Auth, col.Headers, col.Variables)...)
		}
	} else {
		// Filter to matching folder.
		for _, f := range col.Folders {
			if f.Name == folderName {
				requests = append(requests, flattenFolder(f, "", col.Auth, col.Headers, col.Variables)...)
			}
		}
	}

	return requests
}

// flattenFolder recursively walks a folder tree and collects requests.
func flattenFolder(folder domain.Folder, parentPath string, parentAuth *domain.AuthConfig, parentHeaders []domain.Header, parentVars []domain.Variable) []flatRequest {
	path := folder.Name
	if parentPath != "" {
		path = parentPath + "/" + folder.Name
	}

	// Resolve effective auth: folder overrides parent.
	effectiveAuth := parentAuth
	if folder.Auth != nil {
		effectiveAuth = folder.Auth
	}

	// Resolve effective headers: merge parent + folder (folder overrides same key).
	effectiveHeaders := mergeHeaders(parentHeaders, folder.Headers)

	// Resolve effective vars: folder overrides parent.
	effectiveVars := mergeVars(parentVars, folder.Variables)

	var requests []flatRequest
	for _, r := range folder.Requests {
		requests = append(requests, flatRequest{
			request:    r,
			folderPath: path,
			auth:       effectiveAuth,
			headers:    effectiveHeaders,
			vars:       effectiveVars,
		})
	}

	for _, sub := range folder.Folders {
		requests = append(requests, flattenFolder(sub, path, effectiveAuth, effectiveHeaders, effectiveVars)...)
	}

	return requests
}

// mergeHeaders combines two header slices. If both contain the same key,
// the override value wins.
func mergeHeaders(base, override []domain.Header) []domain.Header {
	if len(base) == 0 {
		return override
	}
	if len(override) == 0 {
		return base
	}

	overrideKeys := make(map[string]string, len(override))
	for _, h := range override {
		overrideKeys[h.Key] = h.Value
	}

	var merged []domain.Header
	for _, h := range base {
		if _, overridden := overrideKeys[h.Key]; !overridden {
			merged = append(merged, h)
		}
	}
	merged = append(merged, override...)
	return merged
}

// mergeVars combines two variable slices. Override values take precedence.
func mergeVars(base, override []domain.Variable) []domain.Variable {
	if len(override) == 0 {
		return base
	}

	seen := make(map[string]bool)
	var merged []domain.Variable

	// Add overrides first (they take precedence).
	for _, v := range override {
		seen[v.Key] = true
		merged = append(merged, v)
	}
	// Add base vars that weren't overridden.
	for _, v := range base {
		if !seen[v.Key] {
			merged = append(merged, v)
		}
	}
	return merged
}
