package clarify

import (
	"sort"

	"github.com/amp-labs/connectors/common"
)

// Relationship kinds in xClarifyRelationship. A many-to-one or one-to-one relationship is a foreign
// key stored as an attribute (company_id). One-to-many and many-to-many relationships (a company's
// people, a deal's people, a meeting's users) are not attributes; Read asks for them with `include`.
// https://developer.clarify.ai/docs/guides/schemas-and-relationships#relationships
const (
	relationshipOneToMany  = "one-to-many"
	relationshipManyToMany = "many-to-many"
)

func (r relationshipSchema) isToMany() bool {
	return r.Kind == relationshipOneToMany || r.Kind == relationshipManyToMany
}

// readableObjectName maps a schema entity to the name this connector reads it under, or "" when the
// connector does not read it (attachment, message, meeting_recording...).
func readableObjectName(entity string) string {
	switch {
	case entity == "user":
		return objectNameUsers
	case isRecordObject(entity):
		return entity
	default:
		return ""
	}
}

func (s entitySchema) toManyRelationships() []string {
	names := make([]string, 0)

	for name, property := range s.Properties {
		if property.XClarifyRelationship != nil && property.XClarifyRelationship.isToMany() {
			names = append(names, name)
		}
	}

	sort.Strings(names)

	return names
}

func (s entitySchema) objectMetadata() *common.ObjectMetadata {
	displayName := s.XClarifyLabel.Plural
	if displayName == "" {
		displayName = s.Title
	}

	fields := common.FieldsMetadata{
		// The JSON:API resource id, which Read keeps beside the flattened attributes.
		"id": {DisplayName: "ID", ValueType: common.ValueTypeString, ProviderType: "string"},
	}

	for name, property := range s.Properties {
		if name != "id" {
			fields[name] = property.fieldMetadata(name)
		}
	}

	return common.NewObjectMetadata(displayName, fields)
}

func (p propertySchema) fieldMetadata(name string) common.FieldMetadata {
	displayName := p.Title
	if displayName == "" {
		displayName = name
	}

	field := common.FieldMetadata{
		DisplayName:  displayName,
		ProviderType: p.providerType(),
		ReadOnly:     p.ReadOnly,
	}

	if relationship := p.XClarifyRelationship; relationship != nil {
		field.ProviderType = relationship.Kind
		field.ValueType = relationship.valueType()

		if target := readableObjectName(relationship.Entity); target != "" {
			field.ReferenceTo = []string{target}
		}

		return field
	}

	field.ValueType, field.Values = p.valueType()

	return field
}

// valueType types a relationship field by what Read returns: a to-one relationship is the related
// record's id, a lookup when the connector reads the target and a plain id otherwise; a to-many one
// is a list of ids. The actor fields (_created_by, _updated_by) point at users but can also hold a
// system actor such as "system", which is not a user id.
// https://developer.clarify.ai/docs/guides/schemas-and-relationships#user-identifiers
func (r relationshipSchema) valueType() common.ValueType {
	switch {
	case r.isToMany():
		return common.ValueTypeOther
	case readableObjectName(r.Entity) != "":
		return common.ValueTypeReference
	default:
		return common.ValueTypeString
	}
}

func (p propertySchema) providerType() string {
	if p.Format != "" {
		return p.Format
	}

	return p.Type.primary()
}

// valueType maps JSON Schema types to Ampersand ones. The format decides first: the timestamps are
// typed ["object", "string"] or "object" with format date-time, yet Read returns ISO 8601 strings.
// An enum is a single select (its null option dropped), an array of enum items a multi select.
// Fields defined by a core schema reference (name, email_addresses `{items: [...]}`, rich text)
// and other objects and arrays are other.
func (p propertySchema) valueType() (common.ValueType, []common.FieldValue) {
	if valueType, ok := valueTypeByFormat[p.Format]; ok {
		return valueType, nil
	}

	primary := p.Type.primary()

	switch {
	case primary == "string" && len(p.Enum) != 0:
		return common.ValueTypeSingleSelect, enumValues(p.Enum)
	case primary == "array" && p.Items != nil && len(p.Items.Enum) != 0:
		return common.ValueTypeMultiSelect, enumValues(p.Items.Enum)
	}

	if valueType, ok := valueTypeByType[primary]; ok {
		return valueType, nil
	}

	return common.ValueTypeOther, nil
}

var (
	valueTypeByFormat = map[string]common.ValueType{ // nolint:gochecknoglobals
		"date-time": common.ValueTypeDateTime,
		"date":      common.ValueTypeDate,
	}

	valueTypeByType = map[string]common.ValueType{ // nolint:gochecknoglobals
		"string":  common.ValueTypeString,
		"number":  common.ValueTypeFloat,
		"integer": common.ValueTypeInt,
		"boolean": common.ValueTypeBoolean,
	}
)

func enumValues(enum []any) []common.FieldValue {
	values := make([]common.FieldValue, 0, len(enum))

	for _, value := range enum {
		text, ok := value.(string)
		if !ok {
			continue
		}

		values = append(values, common.FieldValue{Value: text, DisplayValue: text})
	}

	return values
}
