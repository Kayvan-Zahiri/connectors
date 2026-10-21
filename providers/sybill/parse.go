package sybill

import (
	"github.com/amp-labs/connectors/internal/jsonquery"
	"github.com/spyzhov/ajson"
)

// nextPageCursor returns pagination.nextCursor while pagination.hasMore is true.
// The docs say to check hasMore rather than nextCursor to decide whether more pages exist.
// https://api.sybill.ai/docs/pagination.html
func nextPageCursor(node *ajson.Node) (string, error) {
	query := jsonquery.New(node, "pagination")

	hasMore, err := query.BoolWithDefault("hasMore", false)
	if err != nil {
		return "", err
	}

	if !hasMore {
		return "", nil
	}

	return query.StrWithDefault("nextCursor", "")
}
