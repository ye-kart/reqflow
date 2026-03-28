package codegen

import (
	"sort"
	"testing"

	"github.com/ye-kart/reqflow/internal/domain"
)

func TestGenerate_AllLanguagesRegistered(t *testing.T) {
	expected := []string{
		"python", "javascript", "go", "java", "ruby", "php", "csharp", "curl",
	}
	for _, lang := range expected {
		t.Run(lang, func(t *testing.T) {
			req := domain.HTTPRequest{
				Method: domain.MethodGet,
				URL:    "https://example.com",
			}
			_, err := Generate(lang, req)
			if err != nil {
				t.Errorf("Generate(%q) returned error: %v", lang, err)
			}
		})
	}
}

func TestGenerate_UnknownLanguageReturnsError(t *testing.T) {
	req := domain.HTTPRequest{
		Method: domain.MethodGet,
		URL:    "https://example.com",
	}
	_, err := Generate("brainfuck", req)
	if err == nil {
		t.Fatal("expected error for unknown language, got nil")
	}
}

func TestListLanguages_ReturnsAll(t *testing.T) {
	langs := ListLanguages()
	expected := []string{
		"csharp", "curl", "go", "java", "javascript", "php", "python", "ruby",
	}
	sort.Strings(langs)
	if len(langs) != len(expected) {
		t.Fatalf("expected %d languages, got %d: %v", len(expected), len(langs), langs)
	}
	for i, lang := range langs {
		if lang != expected[i] {
			t.Errorf("language[%d] = %q, want %q", i, lang, expected[i])
		}
	}
}
