package codegen

import (
	"fmt"
	"sort"

	"github.com/ye-kart/reqflow/internal/domain"
)

var generators = map[string]func(domain.HTTPRequest) (string, error){
	"python":     GeneratePython,
	"javascript": GenerateJavaScript,
	"go":         GenerateGo,
	"java":       GenerateJava,
	"ruby":       GenerateRuby,
	"php":        GeneratePHP,
	"csharp":     GenerateCSharp,
	"curl":       GenerateCurl,
}

// Generate produces code in the given language for the provided HTTP request.
func Generate(lang string, req domain.HTTPRequest) (string, error) {
	gen, ok := generators[lang]
	if !ok {
		return "", fmt.Errorf("unsupported language: %q (available: %v)", lang, ListLanguages())
	}
	return gen(req)
}

// ListLanguages returns all registered language names in sorted order.
func ListLanguages() []string {
	langs := make([]string, 0, len(generators))
	for lang := range generators {
		langs = append(langs, lang)
	}
	sort.Strings(langs)
	return langs
}
