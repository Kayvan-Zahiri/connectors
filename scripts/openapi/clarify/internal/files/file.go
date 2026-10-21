package files

import (
	_ "embed"
	"encoding/json"

	"github.com/amp-labs/connectors/internal/goutils"
	"github.com/amp-labs/connectors/scripts/openapi/internal/api"
)

var (
	// Static file containing openapi spec, as published at https://api.clarify.ai/swagger-json.
	//
	//go:embed openapi.json
	apiFile []byte

	OpenApiFile = api.NewFile(withoutBooleanRequired(apiFile)) // nolint:gochecknoglobals
	OutputDir   = "providers/clarify/internal/metadata"        // nolint:gochecknoglobals
)

// withoutBooleanRequired drops `"required": true|false` from property schemas. The published spec
// puts a boolean there in 392 places (e.g. the `page[limit]` parameter schema), where OpenAPI 3
// requires an array of property names, and kin-openapi refuses to load the document because of it.
// The required arrays on object schemas are kept.
func withoutBooleanRequired(data []byte) []byte {
	var document any

	goutils.MustBeNil(json.Unmarshal(data, &document))

	dropBooleanRequired(document)

	cleaned, err := json.Marshal(document)
	goutils.MustBeNil(err)

	return cleaned
}

func dropBooleanRequired(node any) {
	switch typed := node.(type) {
	case map[string]any:
		if _, ok := typed["required"].(bool); ok {
			delete(typed, "required")
		}

		for _, child := range typed {
			dropBooleanRequired(child)
		}
	case []any:
		for _, child := range typed {
			dropBooleanRequired(child)
		}
	}
}
