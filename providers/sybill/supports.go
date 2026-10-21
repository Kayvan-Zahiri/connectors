package sybill

import (
	"fmt"
	"slices"
	"strings"

	"github.com/amp-labs/connectors/common"
	"github.com/amp-labs/connectors/internal/components"
)

const (
	objectNameConversations = "conversations"
	objectNameDeals         = "deals"
	objectNameAccounts      = "accounts"
)

var supportedObjects = []string{ // nolint:gochecknoglobals
	objectNameConversations,
	objectNameDeals,
	objectNameAccounts,
}

// Each object's records are keyed by their own identifier field.
// https://api.sybill.ai/docs/data-models.html
var idFields = map[string]string{ // nolint:gochecknoglobals
	objectNameConversations: "conversationId",
	objectNameDeals:         "dealId",
	objectNameAccounts:      "accountId",
}

// SupportedReadObjects lists the objects this connector reads.
func SupportedReadObjects() []string {
	return slices.Clone(supportedObjects)
}

func supportedOperations() components.EndpointRegistryInput {
	return components.EndpointRegistryInput{
		common.ModuleRoot: {
			{
				Endpoint: fmt.Sprintf("{%s}", strings.Join(supportedObjects, ",")),
				Support:  components.ReadSupport,
			},
		},
	}
}
