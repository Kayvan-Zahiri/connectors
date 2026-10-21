package clarify

import (
	"fmt"
	"slices"
	"strings"

	"github.com/amp-labs/connectors/common"
	"github.com/amp-labs/connectors/internal/components"
)

const (
	// Built-in objects, read from GET /v1/workspaces/{workspace}/objects/{object}/resources.
	// https://developer.clarify.ai/docs/getting-started/key-concepts#objects
	objectNamePerson  = "person"
	objectNameCompany = "company"
	objectNameDeal    = "deal"
	objectNameMeeting = "meeting"
	objectNameTask    = "task"

	// Workspace members, read from GET /v1/workspaces/{workspace}/users.
	objectNameUsers = "users"

	// Custom object names are normalized to a `c_` prefix ("Sales Order" becomes `c_sales_order`) and
	// use the same endpoints as built-in objects.
	// https://developer.clarify.ai/docs/guides/schemas-and-relationships#custom-objects
	customObjectPrefix = "c_"
)

var builtInObjects = []string{ // nolint:gochecknoglobals
	objectNamePerson,
	objectNameCompany,
	objectNameDeal,
	objectNameMeeting,
	objectNameTask,
}

// SupportedReadObjects lists the objects this connector reads in every workspace.
// Custom objects (`c_*`) are read too, but exist only in the workspaces that define them.
func SupportedReadObjects() []string {
	return append(slices.Clone(builtInObjects), objectNameUsers)
}

// isRecordObject reports whether the object is read through the generic /objects/{object} endpoints.
func isRecordObject(objectName string) bool {
	return slices.Contains(builtInObjects, objectName) || isCustomObject(objectName)
}

func isCustomObject(objectName string) bool {
	return strings.HasPrefix(objectName, customObjectPrefix) && len(objectName) > len(customObjectPrefix)
}

func supportedOperations() components.EndpointRegistryInput {
	return components.EndpointRegistryInput{
		common.ModuleRoot: {
			{
				Endpoint: fmt.Sprintf("{%s}", strings.Join(SupportedReadObjects(), ",")),
				Support:  components.ReadSupport,
			},
			{
				// Custom objects are named by the workspace; the prefix is all that is known up front.
				Endpoint: customObjectPrefix + "?*",
				Support:  components.ReadSupport,
			},
		},
	}
}
