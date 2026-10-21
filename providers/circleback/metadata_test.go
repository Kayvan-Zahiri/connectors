package circleback

import (
	"testing"

	"github.com/amp-labs/connectors/common"
	"github.com/amp-labs/connectors/test/utils/mockutils"
	"github.com/amp-labs/connectors/test/utils/mockutils/mockserver"
	"github.com/amp-labs/connectors/test/utils/testconn"
)

func TestListObjectMetadata(t *testing.T) { // nolint:funlen
	t.Parallel()

	tests := []testconn.TestCaseListObjectMetadata{
		{
			Name:         "At least one object name must be queried",
			Input:        nil,
			Server:       mockserver.Dummy(),
			ExpectedErrs: []error{common.ErrMissingObjects},
		},
		{
			Name:       "Unknown object is reported per object",
			Input:      []string{"calendar/events"},
			Server:     mockserver.Dummy(),
			Comparator: testconn.ComparatorSubsetMetadata,
			Expected: &common.ListObjectMetadataResult{
				Result: map[string]common.ObjectMetadata{},
				Errors: map[string]error{
					"calendar/events": mockutils.ExpectedSubsetErrors{common.ErrObjectNotSupported},
				},
			},
		},
		{
			Name:       "Meetings come from the catalog with the transcript added",
			Input:      []string{"meetings"},
			Server:     mockserver.Dummy(),
			Comparator: testconn.ComparatorSubsetMetadata,
			Expected: &common.ListObjectMetadataResult{
				Result: map[string]common.ObjectMetadata{
					"meetings": {
						DisplayName: "Meetings",
						Fields: map[string]common.FieldMetadata{
							"id": {
								DisplayName:  "Id",
								ValueType:    common.ValueTypeString,
								ProviderType: "string",
							},
							"updatedAt": {
								DisplayName:  "Updated At",
								ValueType:    common.ValueTypeDateTime,
								ProviderType: "date-time",
							},
							"duration": {
								DisplayName:  "Duration",
								ValueType:    common.ValueTypeFloat,
								ProviderType: "number",
							},
							"linkAccess": {
								DisplayName:  "Link Access",
								ValueType:    common.ValueTypeSingleSelect,
								ProviderType: "string",
								Values: []common.FieldValue{
									{Value: "Editor", DisplayValue: "Editor"},
									{Value: "Viewer", DisplayValue: "Viewer"},
									{Value: "LimitedViewer", DisplayValue: "LimitedViewer"},
								},
							},
							"transcript": {
								DisplayName:  "Transcript",
								ValueType:    common.ValueTypeOther,
								ProviderType: "array",
							},
						},
					},
				},
				Errors: map[string]error{},
			},
		},
		{
			Name:       "Other objects come from the catalog without a transcript",
			Input:      []string{"action-items", "tag", "people", "companies"},
			Server:     mockserver.Dummy(),
			Comparator: testconn.ComparatorSubsetMetadata,
			Expected: &common.ListObjectMetadataResult{
				Result: map[string]common.ObjectMetadata{
					"action-items": {
						DisplayName: "Action Items",
						Fields: map[string]common.FieldMetadata{
							"status": {
								DisplayName:  "Status",
								ValueType:    common.ValueTypeSingleSelect,
								ProviderType: "string",
								Values: []common.FieldValue{
									{Value: "PENDING", DisplayValue: "PENDING"},
									{Value: "DONE", DisplayValue: "DONE"},
								},
							},
						},
					},
					"tag": {
						DisplayName: "Tag",
						Fields: map[string]common.FieldMetadata{
							"name": {DisplayName: "Name", ValueType: common.ValueTypeString, ProviderType: "string"},
						},
					},
					"people": {
						DisplayName: "People",
						Fields: map[string]common.FieldMetadata{
							"email": {DisplayName: "Email", ValueType: common.ValueTypeString, ProviderType: "string"},
						},
					},
					"companies": {
						DisplayName: "Companies",
						Fields: map[string]common.FieldMetadata{
							"domain": {DisplayName: "Domain", ValueType: common.ValueTypeString, ProviderType: "string"},
						},
					},
				},
				Errors: map[string]error{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			t.Parallel()

			tt.Run(t, func() (testconn.TestableMetadataReader, error) {
				return constructTestConnector(tt.Server.URL)
			})
		})
	}
}

func constructTestConnector(serverURL string) (*Connector, error) {
	connector, err := NewConnector(common.ConnectorParams{
		Module:              common.ModuleRoot,
		AuthenticatedClient: mockutils.NewClient(),
	})
	if err != nil {
		return nil, err
	}

	connector.SetBaseURL(mockutils.ReplaceURLOrigin(connector.HTTPClient().Base, serverURL))

	return connector, nil
}
