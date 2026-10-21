package sybill

import (
	"context"
	"strings"

	"github.com/amp-labs/amp-common/simultaneously"
	"github.com/amp-labs/connectors/common"
	"github.com/amp-labs/connectors/common/urlbuilder"
	"github.com/amp-labs/connectors/internal/datautils"
	"github.com/amp-labs/connectors/internal/jsonquery"
	"github.com/amp-labs/connectors/providers/sybill/internal/metadata"
	"github.com/spyzhov/ajson"
)

// Detail requests run a few at a time, as every request counts against 60 a minute per key.
// https://api.sybill.ai/docs/rate-limiting.html
const maxConcurrentDetailFetches = 2

// Fields only GET /v1/conversations/{conversationId} returns.
// https://api.sybill.ai/docs/data-models.html#conversation-detail
var conversationDetailFields = datautils.NewSet( // nolint:gochecknoglobals
	"category",
	"summary",
	"transcript",
	"recordings",
)

func needsConversationDetails(params common.ReadParams) bool {
	if params.ObjectName != objectNameConversations {
		return false
	}

	for field := range params.Fields {
		if conversationDetailFields.Has(strings.ToLower(field)) {
			return true
		}
	}

	return false
}

// makeConversationDetailsFunc returns, in list order, the detail response of each listed conversation.
func (c *Connector) makeConversationDetailsFunc(
	ctx context.Context, listRecords common.NodeRecordsFunc,
) common.NodeRecordsFunc {
	return func(node *ajson.Node) ([]*ajson.Node, error) {
		listed, err := listRecords(node)
		if err != nil {
			return nil, err
		}

		details := make([]*ajson.Node, len(listed))
		jobs := make([]simultaneously.Job, 0, len(listed))

		for index, record := range listed {
			identifier, err := jsonquery.New(record).StringRequired(idFields[objectNameConversations])
			if err != nil {
				return nil, err
			}

			jobs = append(jobs, func(ctx context.Context) error {
				detail, err := c.fetchConversation(ctx, identifier)
				if err != nil {
					return err
				}

				details[index] = detail

				return nil
			})
		}

		if err := simultaneously.DoCtx(ctx, maxConcurrentDetailFetches, jobs...); err != nil {
			return nil, err
		}

		return details, nil
	}
}

// https://api.sybill.ai/docs/endpoints.html#get-v1conversationsconversationid
func (c *Connector) fetchConversation(ctx context.Context, identifier string) (*ajson.Node, error) {
	path, err := metadata.Catalog.ReadPath(objectNameConversations)
	if err != nil {
		return nil, err
	}

	detailURL, err := urlbuilder.New(c.ProviderInfo().BaseURL, path, identifier)
	if err != nil {
		return nil, err
	}

	response, err := c.JSONHTTPClient().Get(ctx, detailURL.String())
	if err != nil {
		return nil, err
	}

	body, ok := response.Body()
	if !ok {
		return nil, common.ErrEmptyJSONHTTPResponse
	}

	return body, nil
}
