package sybill

import (
	"context"

	"github.com/amp-labs/connectors/common"
	"github.com/amp-labs/connectors/providers/sybill/internal/metadata"
)

// ListObjectMetadata serves the catalog generated from the OpenAPI spec. Sybill has no describe
// endpoint, and the read objects have fixed fields.
func (c *Connector) ListObjectMetadata(
	ctx context.Context, objectNames []string,
) (*common.ListObjectMetadataResult, error) {
	return metadata.Catalog.SelectResult(objectNames)
}
