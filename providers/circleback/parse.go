package circleback

import (
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/amp-labs/connectors/common"
	"github.com/amp-labs/connectors/common/readhelper"
	"github.com/amp-labs/connectors/internal/httpkit"
	"github.com/amp-labs/connectors/internal/jsonquery"
	"github.com/spyzhov/ajson"
)

// makeNextPageFunc returns the `rel="next"` URL of the Link header, resolved against the request.
// The link is host-relative and already carries the API prefix: `</api/meetings?cursor=...>`.
// https://circleback.ai/docs/api#pagination
func makeNextPageFunc(request *http.Request, response *common.JSONHTTPResponse) common.NextPageFunc {
	return func(*ajson.Node) (string, error) {
		link := httpkit.HeaderLink(response, "next")
		if link == "" {
			return "", nil
		}

		next, err := url.Parse(link)
		if err != nil {
			return "", fmt.Errorf("invalid next page link %q: %w", link, err)
		}

		return request.URL.ResolveReference(next).String(), nil
	}
}

// makeUpdatedAtFilter keeps meetings whose updatedAt falls within (Since, Until],
// matching ReadParams: Since is exclusive and Until inclusive.
// GET /meetings documents no time filter (only ownership, statuses, cursor, tagIds and
// attendeeProfileIds), so every page is read and filtered here. Paging never stops early,
// because the order of the list is not documented.
// https://circleback.ai/docs/api/openapi.json (paths./meetings.get.parameters)
func makeUpdatedAtFilter(nextPage common.NextPageFunc) common.RecordsFilterFunc {
	boundary := readhelper.NewTimeBoundary(readhelper.ExcludeSince())

	return func(params common.ReadParams, body *ajson.Node, records []*ajson.Node) ([]*ajson.Node, string, error) {
		next, err := nextPage(body)
		if err != nil {
			return nil, "", err
		}

		if params.Since.IsZero() && params.Until.IsZero() {
			return records, next, nil
		}

		filtered := make([]*ajson.Node, 0, len(records))

		for _, record := range records {
			updatedAt, err := jsonquery.New(record).StringRequired("updatedAt")
			if err != nil {
				return nil, "", err
			}

			timestamp, err := time.Parse(time.RFC3339, updatedAt)
			if err != nil {
				return nil, "", fmt.Errorf("invalid updatedAt %q: %w", updatedAt, err)
			}

			if boundary.Contains(params, timestamp) {
				filtered = append(filtered, record)
			}
		}

		return filtered, next, nil
	}
}
