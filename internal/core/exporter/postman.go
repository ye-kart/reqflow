package exporter

import (
	"encoding/json"

	"github.com/ye-kart/reqflow/internal/core/importer"
	"github.com/ye-kart/reqflow/internal/domain"
)

// ExportPostman generates a Postman Collection v2.1 JSON document from a
// domain.Collection. The output is compatible with Postman import.
func ExportPostman(c domain.Collection) ([]byte, error) {
	pc := importer.PostmanCollectionFromDomain(c)
	return json.MarshalIndent(pc, "", "  ")
}

// We need to re-export the internal conversion function from the importer
// package. Since the importer owns the Postman types, the exporter delegates
// to it for the domain -> Postman struct conversion.
var _ = domain.Collection{} // ensure domain import is used
