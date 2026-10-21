package clarify

import (
	"fmt"
	"maps"
	"net/url"
	"strconv"
	"time"

	"github.com/amp-labs/connectors/common"
	"github.com/amp-labs/connectors/common/readhelper"
	"github.com/amp-labs/connectors/common/urlbuilder"
	"github.com/amp-labs/connectors/internal/jsonquery"
	"github.com/spyzhov/ajson"
)

// makeNextPageFunc pages with page[offset]. links.next is null on the last page, so it says whether
// there is another one, but the link itself is not followed: the API returns it with an `http://`
// scheme, which api.clarify.ai does not serve (seen in the live test). The next page is this
// request with page[offset] moved to meta.offset + meta.limit, as the API's own link does.
// https://developer.clarify.ai/docs/api-basics/pagination#response
func makeNextPageFunc(requestURL *url.URL) common.NextPageFunc {
	return func(node *ajson.Node) (string, error) {
		next, err := jsonquery.New(node, "links").StrWithDefault("next", "")
		if err != nil || next == "" {
			return "", err
		}

		meta := jsonquery.New(node, "meta")

		offset, err := meta.IntegerRequired("offset")
		if err != nil {
			return "", err
		}

		limit, err := meta.IntegerRequired("limit")
		if err != nil {
			return "", err
		}

		nextURL, err := urlbuilder.FromRawURL(requestURL)
		if err != nil {
			return "", err
		}

		nextURL.AddEncodingExceptions(queryEncodingExceptions)
		nextURL.WithQueryParam(pageOffsetParam, strconv.FormatInt(offset+limit, 10))

		return nextURL.String(), nil
	}
}

// makeUpdatedAtFilter keeps records whose _updated_at falls within (Since, Until], matching
// ReadParams. For record objects it trims the whole-second window the provider was asked for
// (see resourcesURL). GET /users documents only page and sortOrder parameters, so for users it is
// the only filter and every page is read: the list is ordered by _id, and an old user may have
// been updated recently.
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
			updatedAt, err := jsonquery.New(record, "attributes").StringRequired("_updated_at")
			if err != nil {
				return nil, "", err
			}

			timestamp, err := time.Parse(time.RFC3339, updatedAt)
			if err != nil {
				return nil, "", fmt.Errorf("invalid _updated_at %q: %w", updatedAt, err)
			}

			if boundary.Contains(params, timestamp) {
				filtered = append(filtered, record)
			}
		}

		return filtered, next, nil
	}
}

// flattenRecord promotes a JSON:API resource's attributes and included relationships to top-level
// fields:
//
//	{"id": "5f8b...", "type": "deal",
//	 "attributes": {"name": "Acme renewal", "company_id": "c0a8..."},
//	 "relationships": {"people": {"data": [{"type": "person", "id": "9d1e..."}]}}}
//
// becomes {"id": "5f8b...", "name": "Acme renewal", "company_id": "c0a8...", "people": ["9d1e..."]}.
// A to-one relationship becomes its id. Names are kept verbatim, as GET /schemas names them.
// The resource's `type` is only the object name and is dropped, since meetings have a `type`
// attribute of their own. The resource `id` is never overwritten (attributes carry it again as
// `_id`); an included relationship, which the read asked for by name, replaces an attribute of the
// same name. Everything stays available in Raw.
func flattenRecord(node *ajson.Node) (map[string]any, error) {
	resource, err := jsonquery.Convertor.ObjectToMap(node)
	if err != nil {
		return nil, err
	}

	attributes, _ := resource["attributes"].(map[string]any)
	relationships, _ := resource["relationships"].(map[string]any)

	record := make(map[string]any, len(attributes)+len(relationships)+1)

	maps.Copy(record, attributes)

	for key, value := range relationships {
		if linkage, ok := value.(map[string]any); ok {
			if data, present := linkage["data"]; present {
				record[key] = relatedIDs(data)
			}
		}
	}

	record["id"] = resource["id"]

	return record, nil
}

// relatedIDs reduces JSON:API resource linkage to ids: an array to a list, an object to one id.
func relatedIDs(data any) any {
	switch linkage := data.(type) {
	case []any:
		ids := make([]any, 0, len(linkage))

		for _, item := range linkage {
			if identifier, ok := item.(map[string]any); ok {
				ids = append(ids, identifier["id"])
			}
		}

		return ids
	case map[string]any:
		return linkage["id"]
	default:
		return nil
	}
}
