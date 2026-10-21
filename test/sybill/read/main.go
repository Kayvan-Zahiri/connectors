package main

import (
	"context"
	"os/signal"
	"syscall"

	"github.com/amp-labs/connectors"
	"github.com/amp-labs/connectors/common"
	connTest "github.com/amp-labs/connectors/test/sybill"
	"github.com/amp-labs/connectors/test/utils"
	"github.com/amp-labs/connectors/test/utils/testscenario"
)

func main() {
	// Handle Ctrl-C gracefully.
	ctx, done := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer done()

	// Set up slog logging.
	utils.SetupLogging()

	conn := connTest.GetConnector(ctx)

	testscenario.ReadThroughPages(ctx, conn, common.ReadParams{
		ObjectName: "conversations",
		Fields:     connectors.Fields("conversationId", "title", "startTime", "crm", "summary", "transcript"),
		PageSize:   3,
	})

	testscenario.ReadThroughPages(ctx, conn, common.ReadParams{
		ObjectName: "deals",
		Fields:     connectors.Fields("dealId", "name", "stage", "accountName"),
		PageSize:   10,
	})

	testscenario.ReadThroughPages(ctx, conn, common.ReadParams{
		ObjectName: "accounts",
		Fields:     connectors.Fields("accountId", "name", "website", "latestDeal"),
		PageSize:   10,
	})
}
