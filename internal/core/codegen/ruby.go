package codegen

import (
	"fmt"
	"strings"

	"github.com/ye-kart/reqflow/internal/domain"
)

// rubyRequestClass maps HTTP methods to Ruby Net::HTTP request classes.
var rubyRequestClass = map[domain.HTTPMethod]string{
	domain.MethodGet:     "Net::HTTP::Get",
	domain.MethodPost:    "Net::HTTP::Post",
	domain.MethodPut:     "Net::HTTP::Put",
	domain.MethodPatch:   "Net::HTTP::Patch",
	domain.MethodDelete:  "Net::HTTP::Delete",
	domain.MethodHead:    "Net::HTTP::Head",
	domain.MethodOptions: "Net::HTTP::Options",
}

// GenerateRuby generates Ruby net/http code for the given HTTP request.
func GenerateRuby(req domain.HTTPRequest) (string, error) {
	var sb strings.Builder

	sb.WriteString("require 'net/http'\n")
	sb.WriteString("require 'uri'\n")
	sb.WriteString("require 'json'\n\n")

	sb.WriteString(fmt.Sprintf("uri = URI.parse(%q)\n", req.URL))
	sb.WriteString("http = Net::HTTP.new(uri.host, uri.port)\n")
	sb.WriteString("http.use_ssl = (uri.scheme == 'https')\n\n")

	class := rubyRequestClass[req.Method]
	if class == "" {
		class = "Net::HTTP::Get"
	}

	sb.WriteString(fmt.Sprintf("request = %s.new(uri.request_uri)\n", class))

	for _, h := range req.Headers {
		sb.WriteString(fmt.Sprintf("request[%q] = %q\n", h.Key, h.Value))
	}

	if len(req.Body) > 0 {
		sb.WriteString(fmt.Sprintf("request.body = '%s'\n", string(req.Body)))
	}

	sb.WriteString("\nresponse = http.request(request)\n")
	sb.WriteString("puts response.body\n")

	return sb.String(), nil
}
