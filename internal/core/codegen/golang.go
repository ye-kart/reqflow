package codegen

import (
	"fmt"
	"strings"

	"github.com/ye-kart/reqflow/internal/domain"
)

// GenerateGo generates Go net/http code for the given HTTP request.
func GenerateGo(req domain.HTTPRequest) (string, error) {
	var sb strings.Builder

	method := string(req.Method)

	// Body argument for NewRequest.
	bodyArg := "nil"
	if len(req.Body) > 0 {
		bodyArg = fmt.Sprintf("strings.NewReader(`%s`)", string(req.Body))
	}

	sb.WriteString(fmt.Sprintf("req, _ := http.NewRequest(%q, %q, %s)\n", method, req.URL, bodyArg))

	for _, h := range req.Headers {
		sb.WriteString(fmt.Sprintf("req.Header.Set(%q, %q)\n", h.Key, h.Value))
	}

	sb.WriteString("resp, _ := http.DefaultClient.Do(req)\n")
	sb.WriteString("defer resp.Body.Close()\n")
	sb.WriteString("body, _ := io.ReadAll(resp.Body)\n")
	sb.WriteString("fmt.Println(string(body))\n")

	return sb.String(), nil
}
