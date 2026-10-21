package main

import (
	"github.com/amp-labs/connectors/common"
	"github.com/amp-labs/connectors/scripts/openapi/internal/api/spec"
	"github.com/getkin/kin-openapi/openapi3"
)

// nullableField types a field by its non-null schema. Circleback declares most optional values as
// `anyOf: [{type: X}, {type: null}]`, which the extractor otherwise reports as other.
// Timestamps are strings with `format: date-time`, and numbers may hold fractions (duration is
// 1306.09 seconds), so they are float.
func nullableField(field spec.Field) spec.Field {
	if field.Origin.Schema == nil {
		return field
	}

	property, ok := field.Origin.Schema.Properties[field.Name]
	if !ok || property.Value == nil {
		return field
	}

	schema := nonNullSchema(property.Value)
	if schema == nil || schema.Type == nil {
		return field
	}

	switch {
	case schema.Type.Is(openapi3.TypeString) && schema.Format == "date-time":
		field.Type = schema.Format
		field.ValueType = common.ValueTypeDateTime
	case schema.Type.Is(openapi3.TypeString) && len(schema.Enum) != 0:
		field.Type = openapi3.TypeString
		field.ValueType = common.ValueTypeSingleSelect
		field.ValueOptions = enumValues(schema.Enum)
	case schema.Type.Is(openapi3.TypeString):
		field.Type = openapi3.TypeString
		field.ValueType = common.ValueTypeString
	case schema.Type.Is(openapi3.TypeNumber):
		field.Type = openapi3.TypeNumber
		field.ValueType = common.ValueTypeFloat
	case schema.Type.Is(openapi3.TypeBoolean):
		field.Type = openapi3.TypeBoolean
		field.ValueType = common.ValueTypeBoolean
	}

	return field
}

// nonNullSchema returns the schema itself, or its only non-null anyOf option.
func nonNullSchema(schema *openapi3.Schema) *openapi3.Schema {
	if len(schema.AnyOf) == 0 {
		return schema
	}

	var found *openapi3.Schema

	for _, option := range schema.AnyOf {
		if option.Value == nil || option.Value.Type == nil || option.Value.Type.Is(openapi3.TypeNull) {
			continue
		}

		if found != nil {
			return nil
		}

		found = option.Value
	}

	return found
}

func enumValues(enum []any) []string {
	values := make([]string, 0, len(enum))

	for _, value := range enum {
		if text, ok := value.(string); ok {
			values = append(values, text)
		}
	}

	return values
}
