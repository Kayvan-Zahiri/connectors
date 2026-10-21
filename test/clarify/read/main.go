package main

import (
	"context"
	"log/slog"

	"github.com/amp-labs/connectors"
	"github.com/amp-labs/connectors/common"
	"github.com/amp-labs/connectors/providers/clarify"
	testClarify "github.com/amp-labs/connectors/test/clarify"
	"github.com/amp-labs/connectors/test/utils"
)

func main() {
	ctx := context.Background()
	conn := testClarify.GetClarifyConnector(ctx)

	for _, objectName := range clarify.SupportedReadObjects() {
		slog.Info("Reading", "object", objectName)

		res, err := conn.Read(ctx, common.ReadParams{
			ObjectName: objectName,
			Fields:     connectors.Fields("id", "_created_at", "_updated_at"),
			PageSize:   2,
		})
		if err != nil {
			utils.Fail("error reading from Clarify", "object", objectName, "error", err)
		}

		utils.DumpJSON(res, nil)
	}
}
