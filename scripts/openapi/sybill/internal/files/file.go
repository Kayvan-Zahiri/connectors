package files

import (
	_ "embed"

	"github.com/amp-labs/connectors/scripts/openapi/internal/api"
	"github.com/getkin/kin-openapi/openapi3"
)

var (
	// Static file containing openapi spec, from https://api.sybill.ai/docs/openapi.yaml.
	//
	//go:embed openapi.json
	apiFile []byte

	OpenApiFile = api.NewFile(apiFile)                 // nolint:gochecknoglobals
	OutputDir   = "providers/sybill/internal/metadata" // nolint:gochecknoglobals
)

// Document loads the spec for lookups the list extractor does not cover, such as detail-only schemas.
func Document() (*openapi3.T, error) {
	return openapi3.NewLoader().LoadFromData(apiFile)
}
