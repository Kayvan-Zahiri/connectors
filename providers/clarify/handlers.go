package clarify

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/amp-labs/connectors/common"
	"github.com/amp-labs/connectors/common/readhelper"
	"github.com/amp-labs/connectors/common/urlbuilder"
	"github.com/amp-labs/connectors/internal/datautils"
)

const (
	apiVersion = "v1"

	// List endpoints accept page[limit] up to 500 and return 422 above that.
	// https://developer.clarify.ai/docs/api-basics/pagination#request
	maxPageSize     = 500
	pageSizeParam   = "page[limit]"
	pageOffsetParam = "page[offset]"

	// Comma-separated relationship names to embed under each resource's `relationships`.
	// https://developer.clarify.ai/docs/api-basics/filtering#including-relationships
	includeParam = "include"

	// Date and datetime filters take the operator as a second bracket; the operator names are
	// case-sensitive. Both are inclusive of whole seconds; see resourcesURL.
	// https://developer.clarify.ai/docs/api-basics/filtering#operators-by-field-type
	updatedSinceParam = "filter[_updated_at][Is on or after]"
	updatedUntilParam = "filter[_updated_at][Is before]"

	// The deleted-resources endpoint filters only on _deleted_at, in epoch seconds.
	// https://api.clarify.ai/swagger-json (getDeletedResources)
	deletedSinceParam = "filter[_deleted_at][Greater than]"
	deletedUntilParam = "filter[_deleted_at][Less than or equal]"
)

// Query values are form-encoded, which turns the spaces in operator names into `+`.
// The API's own links percent-encode them, so the connector does the same.
var queryEncodingExceptions = map[string]string{"+": "%20"} // nolint:gochecknoglobals

// buildReadRequest lists one page of an object.
//
//   - person, company, deal, meeting, task and custom objects: GET .../objects/{object}/resources,
//     filtered on _updated_at for Since and Until. Requested one-to-many and many-to-many
//     relationships (a deal's people, a meeting's attendees) are not attributes; they are asked for
//     with `include`.
//   - the same objects with ReadParams.Deleted: GET .../objects/{object}/deleted-resources, records
//     deleted in the last 30 days, filtered on _deleted_at.
//   - users: GET .../users, which takes no filter; parseReadResponse applies Since and Until.
//
// NextPage is the previous request with page[offset] advanced, so it carries every parameter.
func (c *Connector) buildReadRequest(ctx context.Context, params common.ReadParams) (*http.Request, error) {
	readURL, err := c.readURL(ctx, params)
	if err != nil {
		return nil, err
	}

	return http.NewRequestWithContext(ctx, http.MethodGet, readURL, nil)
}

func (c *Connector) readURL(ctx context.Context, params common.ReadParams) (string, error) {
	if params.NextPage != "" {
		return params.NextPage.String(), nil
	}

	if params.ObjectName == objectNameUsers {
		if params.Deleted {
			return "", common.ErrOperationNotSupportedForObject
		}

		usersURL, err := c.workspaceURL("users")
		if err != nil {
			return "", err
		}

		withPageSize(usersURL, params)

		return usersURL.String(), nil
	}

	if params.Deleted {
		return c.deletedResourcesURL(params)
	}

	return c.resourcesURL(ctx, params)
}

func (c *Connector) resourcesURL(ctx context.Context, params common.ReadParams) (string, error) {
	resourcesURL, err := c.workspaceURL("objects", params.ObjectName, "resources")
	if err != nil {
		return "", err
	}

	withPageSize(resourcesURL, params)

	include, err := c.requestedRelationships(ctx, params)
	if err != nil {
		return "", err
	}

	if len(include) != 0 {
		resourcesURL.WithQueryParam(includeParam, strings.Join(include, ","))
	}

	// Records carry _updated_at in milliseconds, but the filter drops the milliseconds of the value
	// it is given. In the live test a record at 06:42:47.281Z came back for both "Is after" that
	// timestamp and "Is on or after 06:42:47.282Z", and for neither "Is on or before" that timestamp
	// nor "Is before 06:42:47.282Z". So the provider gets a window widened to whole seconds, and
	// parseReadResponse trims it to exactly (Since, Until].
	if !params.Since.IsZero() {
		resourcesURL.WithQueryParam(updatedSinceParam,
			datautils.Time.FormatRFC3339inUTC(params.Since.Truncate(time.Second)))
	}

	if !params.Until.IsZero() {
		resourcesURL.WithQueryParam(updatedUntilParam,
			datautils.Time.FormatRFC3339inUTC(params.Until.Truncate(time.Second).Add(time.Second)))
	}

	return resourcesURL.String(), nil
}

// requestedRelationships returns the requested fields that the object's schema marks as one-to-many
// or many-to-many relationships. It reads GET /schemas once per read; later pages reuse the include.
func (c *Connector) requestedRelationships(ctx context.Context, params common.ReadParams) ([]string, error) {
	schemas, err := c.fetchSchemas(ctx)
	if err != nil {
		return nil, err
	}

	schema, ok := schemas[params.ObjectName]
	if !ok {
		return nil, nil
	}

	include := make([]string, 0)

	for _, name := range schema.toManyRelationships() {
		if params.Fields.Has(name) {
			include = append(include, name)
		}
	}

	return include, nil
}

func (c *Connector) deletedResourcesURL(params common.ReadParams) (string, error) {
	deletedURL, err := c.workspaceURL("objects", params.ObjectName, "deleted-resources")
	if err != nil {
		return "", err
	}

	withPageSize(deletedURL, params)

	// The filter takes whole seconds. Since rounds down and Until rounds up, so a window never
	// misses a deletion inside it; it may repeat one from the second at its edge.
	if !params.Since.IsZero() {
		deletedURL.WithQueryParam(deletedSinceParam, strconv.FormatInt(params.Since.Unix(), 10))
	}

	if !params.Until.IsZero() {
		deletedURL.WithQueryParam(deletedUntilParam, strconv.FormatInt(params.Until.Add(time.Second-1).Unix(), 10))
	}

	return deletedURL.String(), nil
}

func (c *Connector) workspaceURL(path ...string) (*urlbuilder.URL, error) {
	segments := append([]string{apiVersion, "workspaces", c.workspace}, path...)

	workspaceURL, err := urlbuilder.New(c.ProviderInfo().BaseURL, segments...)
	if err != nil {
		return nil, err
	}

	workspaceURL.AddEncodingExceptions(queryEncodingExceptions)

	return workspaceURL, nil
}

func withPageSize(readURL *urlbuilder.URL, params common.ReadParams) {
	pageSize := min(readhelper.PageSizeWithDefault(params, maxPageSize), maxPageSize)
	readURL.WithQueryParam(pageSizeParam, strconv.Itoa(pageSize))
}

func (c *Connector) parseReadResponse(
	ctx context.Context,
	params common.ReadParams,
	request *http.Request,
	response *common.JSONHTTPResponse,
) (*common.ReadResult, error) {
	nextPage := makeNextPageFunc(request.URL)

	// Deleted records are windowed on _deleted_at by the provider alone.
	filter := makeUpdatedAtFilter(nextPage)
	if params.Deleted {
		filter = readhelper.MakeIdentityFilterFunc(nextPage)
	}

	return common.ParseResultFiltered(params, response,
		common.MakeRecordsFunc("data"),
		filter,
		readhelper.MakeMarshaledDataFuncWithId(flattenRecord, readhelper.NewIdField("id")),
		params.Fields,
	)
}
