package sybill

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
			Input:      []string{"messages"},
			Server:     mockserver.Dummy(),
			Comparator: testconn.ComparatorSubsetMetadata,
			Expected: &common.ListObjectMetadataResult{
				Result: map[string]common.ObjectMetadata{},
				Errors: map[string]error{
					"messages": mockutils.ExpectedSubsetErrors{common.ErrObjectNotSupported},
				},
			},
		},
		{
			Name:       "Conversations list and detail fields come from the catalog",
			Input:      []string{"conversations"},
			Server:     mockserver.Dummy(),
			Comparator: testconn.ComparatorSubsetMetadata,
			Expected: &common.ListObjectMetadataResult{
				Result: map[string]common.ObjectMetadata{
					"conversations": {
						DisplayName: "Conversations",
						Fields: map[string]common.FieldMetadata{
							"conversationId": {
								DisplayName:  "Conversation Id",
								ValueType:    common.ValueTypeString,
								ProviderType: "string",
							},
							"startTime": {
								DisplayName:  "Start Time",
								ValueType:    common.ValueTypeDateTime,
								ProviderType: "string",
							},
							"type": {
								DisplayName:  "Type",
								ValueType:    common.ValueTypeSingleSelect,
								ProviderType: "string",
								Values: []common.FieldValue{
									{Value: "INTERNAL", DisplayValue: "INTERNAL"},
									{Value: "EXTERNAL", DisplayValue: "EXTERNAL"},
								},
							},
							"participants": {
								DisplayName:  "Participants",
								ValueType:    common.ValueTypeOther,
								ProviderType: "array",
							},
							"transcript": {
								DisplayName:  "Transcript",
								ValueType:    common.ValueTypeOther,
								ProviderType: "array",
							},
							"summary": {
								DisplayName:  "Summary",
								ValueType:    common.ValueTypeOther,
								ProviderType: "object",
							},
							"recordings": {
								DisplayName:  "Recordings",
								ValueType:    common.ValueTypeOther,
								ProviderType: "object",
							},
						},
					},
				},
				Errors: map[string]error{},
			},
		},
		{
			Name:       "Deals and accounts type nullable fields by their non-null option",
			Input:      []string{"deals", "accounts"},
			Server:     mockserver.Dummy(),
			Comparator: testconn.ComparatorSubsetMetadata,
			Expected: &common.ListObjectMetadataResult{
				Result: map[string]common.ObjectMetadata{
					"deals": {
						DisplayName: "Deals",
						Fields: map[string]common.FieldMetadata{
							"amount": {
								DisplayName:  "Amount",
								ValueType:    common.ValueTypeFloat,
								ProviderType: "number",
							},
							"closed": {
								DisplayName:  "Closed",
								ValueType:    common.ValueTypeBoolean,
								ProviderType: "boolean",
							},
							"closeDate": {
								DisplayName:  "Close Date",
								ValueType:    common.ValueTypeDateTime,
								ProviderType: "string",
							},
						},
					},
					"accounts": {
						DisplayName: "Accounts",
						Fields: map[string]common.FieldMetadata{
							"accountId": {
								DisplayName:  "Account Id",
								ValueType:    common.ValueTypeString,
								ProviderType: "string",
							},
							"latestDeal": {
								DisplayName:  "Latest Deal",
								ValueType:    common.ValueTypeOther,
								ProviderType: "object",
							},
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
