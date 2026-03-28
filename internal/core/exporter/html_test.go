package exporter

import (
	"strings"
	"testing"
)

func TestExportHTML_ValidHTMLStructure(t *testing.T) {
	col := testCollection()

	html, err := ExportHTML(col)
	if err != nil {
		t.Fatalf("ExportHTML() error = %v", err)
	}

	content := string(html)

	required := []string{"<html", "<body", "</html>", "</body>", "<style"}
	for _, tag := range required {
		if !strings.Contains(content, tag) {
			t.Errorf("HTML output missing required tag %q", tag)
		}
	}
}

func TestExportHTML_ContainsSameContentAsMarkdown(t *testing.T) {
	col := testCollection()

	html, err := ExportHTML(col)
	if err != nil {
		t.Fatalf("ExportHTML() error = %v", err)
	}

	content := string(html)

	// Should contain the same key content as markdown
	checks := []string{
		"Pet Store API",
		"List Users",
		"Create User",
		"Health Check",
		"List Pets",
		"/users",
		"/pets",
	}
	for _, check := range checks {
		if !strings.Contains(content, check) {
			t.Errorf("HTML output missing content %q", check)
		}
	}
}

func TestExportHTML_HasNavigationSidebar(t *testing.T) {
	col := testCollection()

	html, err := ExportHTML(col)
	if err != nil {
		t.Fatalf("ExportHTML() error = %v", err)
	}

	content := string(html)

	if !strings.Contains(content, "nav") {
		t.Error("HTML output missing navigation sidebar element")
	}
}

func TestExportHTML_HasCSS(t *testing.T) {
	col := testCollection()

	html, err := ExportHTML(col)
	if err != nil {
		t.Fatalf("ExportHTML() error = %v", err)
	}

	content := string(html)

	if !strings.Contains(content, "<style") {
		t.Error("HTML output missing <style> tag")
	}
}
