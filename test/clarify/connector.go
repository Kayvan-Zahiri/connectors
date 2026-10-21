package clarify

import (
	"context"

	"github.com/amp-labs/connectors/common"
	"github.com/amp-labs/connectors/common/scanning/credscanning"
	"github.com/amp-labs/connectors/providers"
	"github.com/amp-labs/connectors/providers/clarify"
	"github.com/amp-labs/connectors/test/utils"
)

// GetClarifyConnector builds the connector from the local creds file: an API key and the
// workspace slug under metadata.workspace.
func GetClarifyConnector(ctx context.Context) *clarify.Connector {
	filePath := credscanning.LoadPath(providers.Clarify)
	reader := utils.MustCreateProvCredJSON(filePath, false)

	conn, err := clarify.NewConnector(common.ConnectorParams{
		AuthenticatedClient: utils.NewAPIKeyClient(ctx, reader, providers.Clarify),
		Workspace:           reader.Get(credscanning.Fields.Workspace),
	})
	if err != nil {
		utils.Fail("error creating Clarify connector", "error", err)
	}

	return conn
}
