package history

import (
	"fmt"
	"strings"

	"github.com/ye-kart/reqflow/internal/domain"
)

// Diff represents the differences between two history entries.
type Diff struct {
	StatusChanged bool
	OldStatus     int
	NewStatus     int
	HeaderDiffs   []HeaderDiff
	BodyDiff      string // unified diff of body
}

// HeaderDiff represents a difference in a single response header.
type HeaderDiff struct {
	Key      string
	OldValue string
	NewValue string
	Added    bool
	Removed  bool
}

// Compare computes the differences between entries a (old) and b (new).
func Compare(a, b Entry) Diff {
	var d Diff

	// Status comparison.
	if a.Response.StatusCode != b.Response.StatusCode {
		d.StatusChanged = true
		d.OldStatus = a.Response.StatusCode
		d.NewStatus = b.Response.StatusCode
	}

	// Header comparison.
	d.HeaderDiffs = compareHeaders(a.Response.Headers, b.Response.Headers)

	// Body comparison.
	d.BodyDiff = diffBodies(string(a.Response.Body), string(b.Response.Body))

	return d
}

// compareHeaders produces a list of header diffs between old and new header sets.
func compareHeaders(old, new []domain.Header) []HeaderDiff {
	oldMap := make(map[string]string, len(old))
	for _, h := range old {
		oldMap[h.Key] = h.Value
	}

	newMap := make(map[string]string, len(new))
	for _, h := range new {
		newMap[h.Key] = h.Value
	}

	var diffs []HeaderDiff

	// Check for changed and removed headers.
	for key, oldVal := range oldMap {
		newVal, exists := newMap[key]
		if !exists {
			diffs = append(diffs, HeaderDiff{
				Key:      key,
				OldValue: oldVal,
				Removed:  true,
			})
		} else if oldVal != newVal {
			diffs = append(diffs, HeaderDiff{
				Key:      key,
				OldValue: oldVal,
				NewValue: newVal,
			})
		}
	}

	// Check for added headers.
	for key, newVal := range newMap {
		if _, exists := oldMap[key]; !exists {
			diffs = append(diffs, HeaderDiff{
				Key:      key,
				NewValue: newVal,
				Added:    true,
			})
		}
	}

	return diffs
}

// diffBodies produces a simple unified-style diff between two body strings.
func diffBodies(old, new string) string {
	if old == new {
		return ""
	}

	oldLines := strings.Split(old, "\n")
	newLines := strings.Split(new, "\n")

	var buf strings.Builder
	buf.WriteString(fmt.Sprintf("--- old\n+++ new\n"))

	// Simple line-by-line diff.
	maxLen := len(oldLines)
	if len(newLines) > maxLen {
		maxLen = len(newLines)
	}

	for i := 0; i < maxLen; i++ {
		var oldLine, newLine string
		if i < len(oldLines) {
			oldLine = oldLines[i]
		}
		if i < len(newLines) {
			newLine = newLines[i]
		}

		if oldLine == newLine {
			buf.WriteString(fmt.Sprintf(" %s\n", oldLine))
		} else {
			if i < len(oldLines) {
				buf.WriteString(fmt.Sprintf("-%s\n", oldLine))
			}
			if i < len(newLines) {
				buf.WriteString(fmt.Sprintf("+%s\n", newLine))
			}
		}
	}

	return buf.String()
}
