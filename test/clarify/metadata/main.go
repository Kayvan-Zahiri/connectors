package main

import (
	"context"

	"github.com/amp-labs/connectors/providers/clarify"
	testClarify "github.com/amp-labs/connectors/test/clarify"
	"github.com/amp-labs/connectors/test/utils"
)

func main() {
	ctx := context.Background()
	conn := testClarify.GetClarifyConnector(ctx)

	metadata, err := conn.ListObjectMetadata(ctx, clarify.SupportedReadObjects())
	if err != nil {
		utils.Fail("error listing metadata for Clarify", "error", err)
	}

	utils.DumpJSON(metadata, nil)
}
