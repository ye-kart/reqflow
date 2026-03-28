package domain

import "time"

// HTTPMethod represents an HTTP method.
type HTTPMethod string

const (
	MethodGet     HTTPMethod = "GET"
	MethodPost    HTTPMethod = "POST"
	MethodPut     HTTPMethod = "PUT"
	MethodPatch   HTTPMethod = "PATCH"
	MethodDelete  HTTPMethod = "DELETE"
	MethodHead    HTTPMethod = "HEAD"
	MethodOptions HTTPMethod = "OPTIONS"
)

// ValidMethods returns all supported HTTP methods.
func ValidMethods() []HTTPMethod {
	return []HTTPMethod{
		MethodGet, MethodPost, MethodPut, MethodPatch,
		MethodDelete, MethodHead, MethodOptions,
	}
}

// IsValid reports whether m is a recognized HTTP method.
func (m HTTPMethod) IsValid() bool {
	for _, valid := range ValidMethods() {
		if m == valid {
			return true
		}
	}
	return false
}

// Header represents an HTTP header key-value pair.
type Header struct {
	Key   string
	Value string
}

// QueryParam represents a URL query parameter.
type QueryParam struct {
	Key   string
	Value string
}

// HTTPRequest represents a fully resolved HTTP request ready to send.
type HTTPRequest struct {
	Method      HTTPMethod
	URL         string
	Headers     []Header
	QueryParams []QueryParam
	Body        []byte
	ContentType string
	NoCookies   bool
}

// TimingInfo holds detailed timing breakdown for an HTTP request.
type TimingInfo struct {
	DNSLookup    time.Duration
	TCPConnect   time.Duration
	TLSHandshake time.Duration
	FirstByte    time.Duration
	Total        time.Duration
}

// HTTPResponse represents an HTTP response received from a server.
type HTTPResponse struct {
	StatusCode int
	Status     string
	Headers    []Header
	Body       []byte
	Duration   time.Duration
	Size       int64
	Timing     TimingInfo
}

// RequestConfig represents the user's request configuration before variable
// resolution and auth application. This is the "intent" that gets transformed
// into an HTTPRequest.
type RequestConfig struct {
	Method      HTTPMethod
	URL         string
	Headers     []Header
	QueryParams []QueryParam
	Body        []byte
	ContentType string
	Auth        *AuthConfig
	Timeout     time.Duration
	NoCookies   bool
	PreScript   string // JavaScript to run before the request
	PostScript  string // JavaScript to run after the response
}
