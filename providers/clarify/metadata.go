package clarify

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"sync"

	"github.com/amp-labs/connectors/common"
	"github.com/amp-labs/connectors/common/urlbuilder"
	"github.com/amp-labs/connectors/internal/datautils"
	"github.com/amp-labs/connectors/providers/clarify/internal/metadata"
)

// ListObjectMetadata describes person, company, deal, meeting, task and custom objects live from
// GET /schemas, the workspace's own JSON Schema per object, so custom fields and select options are
// the workspace's current ones. Users have no schema there; their fields come from the catalog
// generated from the OpenAPI spec.
// https://developer.clarify.ai/docs/guides/schemas-and-relationships#schema-fields
func (c *Connector) ListObjectMetadata(
	ctx context.Context, objectNames []string,
) (*common.ListObjectMetadataResult, error) {
	if len(objectNames) == 0 {
		return nil, common.ErrMissingObjects
	}

	result := common.NewListObjectMetadataResult()
	recordObjects := make([]string, 0, len(objectNames))

	for _, objectName := range datautils.NewSetFromList(objectNames).List() {
		switch {
		case objectName == objectNameUsers:
			users, err := metadata.Catalog.Object(objectName)
			if err != nil {
				result.AppendError(objectName, err)

				continue
			}

			result.Result[objectName] = *users
		case isRecordObject(objectName):
			recordObjects = append(recordObjects, objectName)
		default:
			result.AppendError(objectName, common.ErrObjectNotSupported)
		}
	}

	if len(recordObjects) != 0 {
		c.describeRecordObjects(ctx, result, recordObjects)
	}

	return result, nil
}

// describeRecordObjects adds each object's metadata from one GET /schemas, or a per-object error.
func (c *Connector) describeRecordObjects(
	ctx context.Context, result *common.ListObjectMetadataResult, objectNames []string,
) {
	schemas, err := c.fetchSchemas(ctx)
	if err != nil {
		for _, objectName := range objectNames {
			result.AppendError(objectName, err)
		}

		return
	}

	for _, objectName := range objectNames {
		schema, ok := schemas[objectName]
		if !ok {
			result.AppendError(objectName, fmt.Errorf("%w: no schema for %s", common.ErrObjectNotSupported, objectName))

			continue
		}

		result.Result[objectName] = *schema.objectMetadata()
	}
}

// schemaCache keeps the workspace schemas a connector instance has read, so ListObjectMetadata and
// every first page of a read share one GET /schemas (about 240 KB in the test workspace).
// A failed fetch is not cached.
type schemaCache struct {
	mutex   sync.Mutex
	schemas map[string]entitySchema
}

// fetchSchemas returns the workspace's object schemas, reading them on first use.
func (c *Connector) fetchSchemas(ctx context.Context) (map[string]entitySchema, error) {
	c.schemaCache.mutex.Lock()
	defer c.schemaCache.mutex.Unlock()

	if c.schemaCache.schemas != nil {
		return c.schemaCache.schemas, nil
	}

	schemas, err := c.readSchemas(ctx)
	if err != nil {
		return nil, err
	}

	c.schemaCache.schemas = schemas

	return schemas, nil
}

// readSchemas reads every schema in the workspace, following links.next, and keys the object
// schemas (`.../entities/{object}`) by object name. The `.../core/*` base definitions are skipped.
func (c *Connector) readSchemas(ctx context.Context) (map[string]entitySchema, error) {
	schemasURL, err := c.workspaceURL("schemas")
	if err != nil {
		return nil, err
	}

	schemas := make(map[string]entitySchema)
	nextURL := schemasURL.String()

	for nextURL != "" {
		response, err := c.JSONHTTPClient().Get(ctx, nextURL)
		if err != nil {
			return nil, err
		}

		page, err := common.UnmarshalJSON[schemasResponse](response)
		if err != nil {
			return nil, err
		}

		if page == nil {
			return nil, common.ErrEmptyJSONHTTPResponse
		}

		for _, resource := range page.Data {
			if objectName, ok := entityName(resource.ID); ok {
				schemas[objectName] = resource.Attributes
			}
		}

		nextURL = ""
		if page.Links.Next != nil {
			nextURL, err = onOrigin(*page.Links.Next, schemasURL)
			if err != nil {
				return nil, err
			}
		}
	}

	return schemas, nil
}

// onOrigin moves a links.next URL onto the connector's own scheme and host, keeping its path and
// query. Clarify returns these links as http://api.clarify.ai/..., which does not answer (seen on
// resource lists in the live test); see makeNextPageFunc.
func onOrigin(link string, origin *urlbuilder.URL) (string, error) {
	next, err := url.Parse(link)
	if err != nil {
		return "", err
	}

	base, err := origin.ToURL()
	if err != nil {
		return "", err
	}

	next.Scheme = base.Scheme
	next.Host = base.Host

	return next.String(), nil
}

const entitySchemaPathPrefix = "/schemas/entities/"

// entityName extracts the object name from a schema id such as
// https://getclarify.ai/schemas/entities/c_project.
func entityName(schemaID string) (string, bool) {
	index := strings.LastIndex(schemaID, entitySchemaPathPrefix)
	if index < 0 {
		return "", false
	}

	name := schemaID[index+len(entitySchemaPathPrefix):]

	return name, name != "" && !strings.Contains(name, "/")
}

// https://api.clarify.ai/swagger-json (getSchemas)
type schemasResponse struct {
	Data []struct {
		ID         string       `json:"id"`
		Attributes entitySchema `json:"attributes"`
	} `json:"data"`
	Links struct {
		Next *string `json:"next"`
	} `json:"links"`
}

type entitySchema struct {
	Title         string                    `json:"title"`
	Properties    map[string]propertySchema `json:"properties"`
	XClarifyLabel struct {
		Plural string `json:"plural"`
	} `json:"xClarifyLabel"`
}

type propertySchema struct {
	Type                 schemaType          `json:"type"`
	Format               string              `json:"format"`
	Title                string              `json:"title"`
	Enum                 []any               `json:"enum"`
	Items                *propertySchema     `json:"items"`
	ReadOnly             *bool               `json:"readOnly"`
	XClarifyRelationship *relationshipSchema `json:"xClarifyRelationship"`
}

// relationshipSchema marks a foreign-key field, e.g. company_id on person:
// {"kind": "many-to-one", "field": "people", "entity": "company"}.
type relationshipSchema struct {
	Kind   string `json:"kind"`
	Field  string `json:"field"`
	Entity string `json:"entity"`
}

// schemaType is a JSON Schema `type`, which is either one type name or a list of them
// (["string", "null"] for a nullable field).
type schemaType []string

func (t *schemaType) UnmarshalJSON(data []byte) error {
	var single string
	if err := json.Unmarshal(data, &single); err == nil {
		*t = schemaType{single}

		return nil
	}

	var list []string
	if err := json.Unmarshal(data, &list); err != nil {
		return err
	}

	*t = list

	return nil
}

// primary is the type other than null, or "" when there is none or more than one.
func (t schemaType) primary() string {
	primary := ""

	for _, name := range t {
		if name == "null" {
			continue
		}

		if primary != "" {
			return ""
		}

		primary = name
	}

	return primary
}
