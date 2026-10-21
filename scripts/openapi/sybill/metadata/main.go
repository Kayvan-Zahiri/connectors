package main

import (
	"slices"

	"github.com/amp-labs/connectors/common"
	"github.com/amp-labs/connectors/internal/goutils"
	"github.com/amp-labs/connectors/scripts/openapi/internal/api"
	"github.com/amp-labs/connectors/scripts/openapi/internal/api/filters"
	mapping "github.com/amp-labs/connectors/scripts/openapi/internal/api/map"
	"github.com/amp-labs/connectors/scripts/openapi/internal/api/map/mappingfields"
	"github.com/amp-labs/connectors/scripts/openapi/internal/api/output"
	"github.com/amp-labs/connectors/scripts/openapi/internal/api/pipeline"
	"github.com/amp-labs/connectors/scripts/openapi/internal/api/reducers"
	"github.com/amp-labs/connectors/scripts/openapi/internal/api/spec"
	"github.com/amp-labs/connectors/scripts/openapi/sybill/internal/files"
	"github.com/getkin/kin-openapi/openapi3"
)

func main() {
	extractor, err := files.OpenApiFile.Extractor()
	goutils.MustBeNil(err)

	// Every list response holds one array, named after the object, beside a `pagination` object.
	getSchemas, err := extractor.ExtractListSchemas(api.GET,
		api.List.WithAutoSelectArrayItems(),
	)
	goutils.MustBeNil(err)

	document, err := files.Document()
	goutils.MustBeNil(err)

	// Read fetches GET /v1/conversations/{conversationId} for each listed conversation,
	// so conversations carry the detail fields (transcript, summary, recordings, category) too.
	conversationDetail := document.Components.Schemas["ConversationDetail"].Value

	// Only the collections this connector reads.
	readObjects := filters.NewRules(
		"/v1/conversations",
		"/v1/deals",
		"/v1/accounts",
	)

	pipe := pipeline.NewSchemaPipe(getSchemas).
		Filter(filters.KeepWithPath(filters.AndPathMatcher{
			filters.NoIdPath{}, filters.NewAllowPathStrategy(readObjects),
		})).
		Reduce(reducers.ShortestNameFromUrlWithFullFallback).
		Map(mapping.DisplayNameFromObjectName).
		Map(mapping.DisplayNameFormat(
			mapping.CamelCaseToSpaceSeparated,
			mapping.CapitalizeFirstLetterEveryWord,
		)).
		Map(addFields("conversations", conversationDetail)).
		Map(mapping.Fields(
			typedField,
			mappingfields.DisplayNameFormat(func(_, name string) string {
				return mapping.CapitalizeFirstLetterEveryWord(mapping.CamelCaseToSpaceSeparated(name))
			}),
		))

	output.PrintUnusedRules("readObjects", readObjects)
	goutils.MustBeNil(output.WriteMetadata(files.OutputDir, pipe))
	goutils.MustBeNil(output.WriteQueryParamStats(files.OutputDir, pipe))
	goutils.MustBeNil(output.WriteEndpoints(files.OutputDir, &pipe, nil, nil, nil))
}

// addFields adds the properties of source that the object's list schema lacks.
func addFields(objectName string, source *openapi3.Schema) pipeline.MapFunc[spec.Schema, spec.Schema] {
	return func(schema spec.Schema) spec.Schema {
		if schema.ObjectName != objectName {
			return schema
		}

		for name := range source.Properties {
			if _, exists := schema.Fields[name]; exists {
				continue
			}

			schema.Fields[name] = spec.Field{
				URLPath:     schema.UrlPath,
				Name:        name,
				DisplayName: name,
				Origin: spec.FieldOrigin{
					ItemsSchema: source,
					Schema:      source,
				},
			}
		}

		return schema
	}
}

// typedField types a field by its schema. Most Sybill fields are nullable, declared as
// `anyOf: [<type>, null]`, which the extractor leaves untyped; the non-null option decides here.
// An enum, or a union of enums (conversation category), is a single select.
func typedField(field spec.Field) spec.Field {
	if field.Origin.Schema == nil {
		return field
	}

	property, ok := field.Origin.Schema.Properties[field.Name]
	if !ok || property.Value == nil {
		return field
	}

	options := nonNullOptions(property.Value)

	types := make([]string, 0, len(options))
	formats := make([]string, 0, len(options))
	values := make([]string, 0)

	for _, option := range options {
		types = appendUnique(types, option.Type.Slice()...)
		formats = appendUnique(formats, option.Format)

		for _, value := range option.Enum {
			if text, ok := value.(string); ok {
				values = appendUnique(values, text)
			}
		}
	}

	if len(types) != 1 {
		field.ValueType = common.ValueTypeOther

		return field
	}

	field.Type = types[0]
	field.ValueOptions = values
	field.ValueType = valueType(types[0], formats, values)

	return field
}

func valueType(typeName string, formats, values []string) common.ValueType {
	switch typeName {
	case openapi3.TypeString:
		if len(values) != 0 {
			return common.ValueTypeSingleSelect
		}

		if len(formats) == 1 && formats[0] == "date-time" {
			return common.ValueTypeDateTime
		}

		return common.ValueTypeString
	case openapi3.TypeNumber:
		return common.ValueTypeFloat
	case openapi3.TypeInteger:
		return common.ValueTypeInt
	case openapi3.TypeBoolean:
		return common.ValueTypeBoolean
	default:
		return common.ValueTypeOther
	}
}

// nonNullOptions returns the schema itself, or each typed non-null option of its anyOf.
func nonNullOptions(schema *openapi3.Schema) []*openapi3.Schema {
	if len(schema.AnyOf) == 0 {
		return []*openapi3.Schema{schema}
	}

	options := make([]*openapi3.Schema, 0, len(schema.AnyOf))

	for _, option := range schema.AnyOf {
		if option.Value == nil || option.Value.Type == nil || option.Value.Type.Is(openapi3.TypeNull) {
			continue
		}

		options = append(options, option.Value)
	}

	return options
}

func appendUnique(list []string, items ...string) []string {
	for _, item := range items {
		if !slices.Contains(list, item) {
			list = append(list, item)
		}
	}

	return list
}
