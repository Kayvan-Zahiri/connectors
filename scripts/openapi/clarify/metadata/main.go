package main

import (
	"strings"

	"github.com/amp-labs/connectors/common"
	"github.com/amp-labs/connectors/internal/goutils"
	"github.com/amp-labs/connectors/scripts/openapi/clarify/internal/files"
	"github.com/amp-labs/connectors/scripts/openapi/internal/api"
	"github.com/amp-labs/connectors/scripts/openapi/internal/api/filters"
	mapping "github.com/amp-labs/connectors/scripts/openapi/internal/api/map"
	"github.com/amp-labs/connectors/scripts/openapi/internal/api/map/mappingfields"
	"github.com/amp-labs/connectors/scripts/openapi/internal/api/output"
	"github.com/amp-labs/connectors/scripts/openapi/internal/api/pipeline"
	"github.com/amp-labs/connectors/scripts/openapi/internal/api/reducers"
	"github.com/amp-labs/connectors/scripts/openapi/internal/api/spec"
)

// The catalog only describes workspace users. Records of person, company, deal, meeting, task and
// custom objects all come from the generic /objects/{object}/resources endpoint, whose spec schema
// has no per-object fields; the connector describes them live from GET /schemas.
func main() {
	extractor, err := files.OpenApiFile.Extractor()
	goutils.MustBeNil(err)

	getSchemas, err := extractor.ExtractListSchemas(api.GET,
		api.List.WithArrayLocator(api.ArrayLocationAtData),
		api.List.WithPropertyFlattener(func(objectName, fieldName string) bool {
			// Read promotes the JSON:API `attributes` to top-level fields.
			return fieldName == "attributes"
		}),
	)
	goutils.MustBeNil(err)

	// The path keeps the {workspace} slug, which the connector substitutes, so NoIdPath is not used.
	readObjects := filters.NewRules(
		"/workspaces/{workspace}/users",
	)

	pipe := pipeline.NewSchemaPipe(getSchemas).
		Filter(filters.KeepWithPath(filters.NewAllowPathStrategy(readObjects))).
		Reduce(reducers.ShortestNameFromUrlWithFullFallback).
		Map(mapping.DisplayNameFromObjectName).
		Map(mapping.DisplayNameFormat(mapping.CapitalizeFirstLetterEveryWord)).
		Map(withoutResourceMembers).
		Map(mapping.Fields(
			timestampField,
			mappingfields.DisplayNameFormat(func(_, name string) string {
				return mapping.CapitalizeFirstLetterEveryWord(
					strings.ReplaceAll(strings.TrimPrefix(name, "_"), "_", " "))
			}),
		))

	output.PrintUnusedRules("readObjects", readObjects)
	goutils.MustBeNil(output.WriteMetadata(files.OutputDir, pipe))
	goutils.MustBeNil(output.WriteQueryParamStats(files.OutputDir, pipe))
	goutils.MustBeNil(output.WriteEndpoints(files.OutputDir, &pipe, nil, nil, nil))
}

// withoutResourceMembers drops the JSON:API `type` (always "user") and `relationships` (absent from
// live /users responses) members, which Read does not return as fields.
func withoutResourceMembers(schema spec.Schema) spec.Schema {
	delete(schema.Fields, "type")
	delete(schema.Fields, "relationships")

	return schema
}

// timestampField types the user timestamps, which the spec declares as a oneOf of a date-time
// string, a plain string and an object, as datetimes.
func timestampField(field spec.Field) spec.Field {
	switch field.Name {
	case "_created_at", "_updated_at", "last_active_at":
		field.ValueType = common.ValueTypeDateTime
	}

	return field
}
