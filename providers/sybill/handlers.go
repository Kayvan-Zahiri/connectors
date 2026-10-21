package sybill

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/amp-labs/connectors/common"
	"github.com/amp-labs/connectors/common/readhelper"
	"github.com/amp-labs/connectors/common/urlbuilder"
	"github.com/amp-labs/connectors/internal/datautils"
	"github.com/amp-labs/connectors/providers/sybill/internal/metadata"
)

const (
	// Every list endpoint accepts a limit of 1 to 50.
	// https://api.sybill.ai/docs/pagination.html#controlling-page-size
	maxPageSize = 50

	// A conversations page that needs its details costs one more request per record, against a
	// limit of 60 requests a minute per key, so it defaults to a small page.
	// https://api.sybill.ai/docs/rate-limiting.html
	defaultDetailPageSize = 10

	// GET /v1/conversations filters only on the meeting's start time (startedAfter/startedBefore),
	// and has no updated-at filter or field. A meeting shows up, and its transcript, summary and
	// recordings fill in, only after it has been recorded and processed, which can be hours after
	// it started. A read starting exactly at Since would miss every meeting that started before the
	// previous read but was not processed yet, so Since is moved back by this window and recent
	// meetings are read again on each incremental read, with whatever content is ready by then.
	// The ingest API caps a meeting at 10 hours (ConversationRequest.endedAt), so a day covers a
	// long meeting plus its processing.
	// https://api.sybill.ai/docs/endpoints.html#conversations
	conversationsLookback = 24 * time.Hour
)

// buildReadRequest lists one page of an object. Pages are cursor-based, and each request repeats
// the same filters and limit, as the cursor does not carry them.
// https://api.sybill.ai/docs/pagination.html#combining-with-filters
//
// Deals and accounts have no updated-at filter or field: their only date filters are createdDate,
// closeDate and lastActivityDate, none of which moves when a record is edited. They are read in
// full on every read.
// https://api.sybill.ai/docs/endpoints.html#deals
// https://api.sybill.ai/docs/endpoints.html#accounts
func (c *Connector) buildReadRequest(ctx context.Context, params common.ReadParams) (*http.Request, error) {
	path, err := metadata.Catalog.ReadPath(params.ObjectName)
	if err != nil {
		return nil, err
	}

	readURL, err := urlbuilder.New(c.ProviderInfo().BaseURL, path)
	if err != nil {
		return nil, err
	}

	readURL.WithQueryParam("limit", strconv.Itoa(pageSize(params)))

	if params.ObjectName == objectNameConversations {
		if !params.Since.IsZero() {
			readURL.WithQueryParam("startedAfter",
				datautils.Time.FormatRFC3339inUTC(params.Since.Add(-conversationsLookback)))
		}

		if !params.Until.IsZero() {
			readURL.WithQueryParam("startedBefore", datautils.Time.FormatRFC3339inUTC(params.Until))
		}
	}

	if params.NextPage != "" {
		readURL.WithQueryParam("cursor", params.NextPage.String())
	}

	return http.NewRequestWithContext(ctx, http.MethodGet, readURL.String(), nil)
}

func pageSize(params common.ReadParams) int {
	defaultSize := maxPageSize
	if needsConversationDetails(params) {
		defaultSize = defaultDetailPageSize
	}

	return min(readhelper.PageSizeWithDefault(params, defaultSize), maxPageSize)
}

// parseReadResponse returns the listed records. Conversations whose requested fields are only on
// the detail endpoint are replaced by their GET /v1/conversations/{conversationId} response, which
// extends the list item with the transcript, summary, recordings and category.
// https://api.sybill.ai/docs/data-models.html#conversation-detail
func (c *Connector) parseReadResponse(
	ctx context.Context,
	params common.ReadParams,
	request *http.Request,
	response *common.JSONHTTPResponse,
) (*common.ReadResult, error) {
	records := common.MakeRecordsFunc(params.ObjectName)
	if needsConversationDetails(params) {
		records = c.makeConversationDetailsFunc(ctx, records)
	}

	return common.ParseResult(
		response,
		records,
		nextPageCursor,
		readhelper.MakeMarshaledDataFuncWithId(nil, readhelper.NewIdField(idFields[params.ObjectName])),
		params.Fields,
	)
}
