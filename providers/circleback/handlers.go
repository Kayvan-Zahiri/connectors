package circleback

import (
	"context"
	"net/http"
	"strconv"

	"github.com/amp-labs/connectors/common"
	"github.com/amp-labs/connectors/common/readhelper"
	"github.com/amp-labs/connectors/common/urlbuilder"
	"github.com/amp-labs/connectors/providers/circleback/internal/metadata"
)

// buildReadRequest lists one page of an object. Later pages follow the `Link: <...>; rel="next"` URL
// as given, since its cursor is only valid with the filters of the first request.
// https://circleback.ai/docs/api#pagination
func (c *Connector) buildReadRequest(ctx context.Context, params common.ReadParams) (*http.Request, error) {
	readURL, err := c.readURL(params)
	if err != nil {
		return nil, err
	}

	return http.NewRequestWithContext(ctx, http.MethodGet, readURL.String(), nil)
}

func (c *Connector) readURL(params common.ReadParams) (*urlbuilder.URL, error) {
	if params.NextPage != "" {
		return urlbuilder.New(params.NextPage.String())
	}

	path, err := metadata.Catalog.ReadPath(params.ObjectName)
	if err != nil {
		return nil, err
	}

	readURL, err := urlbuilder.New(c.ProviderInfo().BaseURL, path)
	if err != nil {
		return nil, err
	}

	switch params.ObjectName {
	case objectNameMeetings:
		// The default, Mine, lists only the meetings the user owns. All adds those shared with them.
		readURL.WithQueryParam("ownership", "All")
	case objectNameActionItems:
		// The default, Me, lists only the user's own action items.
		readURL.WithQueryParam("assigneeType", "Anyone")
	case objectNamePeople:
		// /people is the only list endpoint with a page size parameter.
		if params.PageSize > 0 {
			readURL.WithQueryParam("limit", strconv.Itoa(params.PageSize))
		}
	}

	return readURL, nil
}

func (c *Connector) parseReadResponse(
	ctx context.Context,
	params common.ReadParams,
	request *http.Request,
	response *common.JSONHTTPResponse,
) (*common.ReadResult, error) {
	nextPage := makeNextPageFunc(request, response)

	if params.ObjectName != objectNameMeetings {
		return common.ParseResultFiltered(params, response,
			common.MakeRecordsFunc(""),
			readhelper.MakeIdentityFilterFunc(nextPage),
			readhelper.MakeMarshaledDataFuncWithId(nil, recordIDField(params.ObjectName)),
			params.Fields,
		)
	}

	transcripts := make(transcriptRegistry)
	filter := makeUpdatedAtFilter(nextPage)

	if params.Fields.Has(fieldTranscript) {
		filter = c.withTranscripts(ctx, filter, transcripts)
	}

	return common.ParseResultFiltered(params, response,
		common.MakeRecordsFunc(""),
		filter,
		readhelper.MakeMarshaledDataFuncWithId(transcripts.embed, readhelper.NewIdField("id")),
		params.Fields,
	)
}

// recordIDField names each object's stable identifier. A company is identified by its domain:
// it is the only field the spec requires, and the API looks companies up by it (GET /company/{domain}).
func recordIDField(objectName string) readhelper.IdFieldQuery {
	if objectName == objectNameCompanies {
		return readhelper.NewIdField("domain")
	}

	return readhelper.NewIdField("id")
}
