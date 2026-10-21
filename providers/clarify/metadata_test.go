package clarify

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/amp-labs/connectors/common"
	"github.com/amp-labs/connectors/test/utils/mockutils"
	"github.com/amp-labs/connectors/test/utils/mockutils/mockcond"
	"github.com/amp-labs/connectors/test/utils/mockutils/mockserver"
	"github.com/amp-labs/connectors/test/utils/testconn"
	"github.com/amp-labs/connectors/test/utils/testutils"
)

func TestListObjectMetadata(t *testing.T) { // nolint:funlen
	t.Parallel()

	responseSchemas := testutils.DataFromFile(t, "schemas.json")
	responseNotFound := testutils.DataFromFile(t, "error-workspace-not-found.json")

	schemasServer := func() *httptest.Server {
		return mockserver.Conditional{
			Setup: mockserver.ContentJSON(),
			If:    mockcond.And{mockcond.MethodGET(), mockcond.Path("/v1/workspaces/acme/schemas")},
			Then:  mockserver.Response(http.StatusOK, responseSchemas),
		}.Server()
	}

	tests := []testconn.TestCaseListObjectMetadata{
		{
			Name:       "Built-in objects are described from the workspace schemas",
			Input:      []string{"person", "deal", "meeting"},
			Server:     schemasServer(),
			Comparator: testconn.ComparatorSubsetMetadata,
			Expected: &common.ListObjectMetadataResult{
				Result: map[string]common.ObjectMetadata{
					"person": {
						DisplayName: "People",
						Fields: map[string]common.FieldMetadata{
							"id": {
								DisplayName:  "ID",
								ValueType:    common.ValueTypeString,
								ProviderType: "string",
							},
							"job_title": {
								DisplayName:  "Job title",
								ValueType:    common.ValueTypeString,
								ProviderType: "string",
							},
							"company_id": {
								DisplayName:  "Company",
								ValueType:    common.ValueTypeReference,
								ProviderType: "many-to-one",
								ReferenceTo:  []string{"company"},
							},
							"deals": {
								DisplayName:  "Deals",
								ValueType:    common.ValueTypeOther,
								ProviderType: "many-to-many",
								ReferenceTo:  []string{"deal"},
							},
							"messages": {
								DisplayName:  "Messages",
								ValueType:    common.ValueTypeOther,
								ProviderType: "many-to-many",
								ReadOnly:     new(true),
							},
							"_created_by": {
								DisplayName:  "Created by",
								ValueType:    common.ValueTypeString,
								ProviderType: "many-to-one",
								ReadOnly:     new(true),
							},
							"_updated_at": {
								DisplayName:  "Updated at",
								ValueType:    common.ValueTypeDateTime,
								ProviderType: "date-time",
								ReadOnly:     new(true),
							},
							"email_addresses": {
								DisplayName: "Email addresses",
								ValueType:   common.ValueTypeOther,
							},
						},
					},
					"deal": {
						DisplayName: "Deals",
						Fields: map[string]common.FieldMetadata{
							"amount": {
								DisplayName:  "Amount",
								ValueType:    common.ValueTypeFloat,
								ProviderType: "number",
							},
							"close_date": {
								DisplayName:  "Close date",
								ValueType:    common.ValueTypeDate,
								ProviderType: "date",
							},
							"stage": {
								DisplayName:  "Stage",
								ValueType:    common.ValueTypeSingleSelect,
								ProviderType: "string",
								Values: []common.FieldValue{
									{Value: "Lead", DisplayValue: "Lead"},
									{Value: "In progress", DisplayValue: "In progress"},
									{Value: "Won", DisplayValue: "Won"},
									{Value: "Lost", DisplayValue: "Lost"},
								},
							},
							"owner_id": {
								DisplayName:  "Owner",
								ValueType:    common.ValueTypeReference,
								ProviderType: "many-to-one",
								ReferenceTo:  []string{"users"},
							},
							"people": {
								DisplayName:  "People",
								ValueType:    common.ValueTypeOther,
								ProviderType: "many-to-many",
								ReferenceTo:  []string{"person"},
							},
						},
					},
					"meeting": {
						DisplayName: "Meetings",
						Fields: map[string]common.FieldMetadata{
							"type": {
								DisplayName:  "Type",
								ValueType:    common.ValueTypeSingleSelect,
								ProviderType: "string",
								Values: []common.FieldValue{
									{Value: "Sales: Discovery", DisplayValue: "Sales: Discovery"},
									{Value: "Sales: Closing", DisplayValue: "Sales: Closing"},
									{Value: "CS: Onboarding", DisplayValue: "CS: Onboarding"},
									{Value: "CS: Account Review", DisplayValue: "CS: Account Review"},
									{Value: "CS: Support", DisplayValue: "CS: Support"},
								},
							},
							"start": {
								DisplayName:  "Start time",
								ValueType:    common.ValueTypeDateTime,
								ProviderType: "date-time",
								ReadOnly:     new(true),
							},
							"recurrence_rule_id": {
								DisplayName:  "Recurrence rule",
								ValueType:    common.ValueTypeString,
								ProviderType: "many-to-one",
								ReadOnly:     new(true),
							},
						},
					},
				},
				Errors: map[string]error{},
			},
		},
		{
			Name:       "Custom object missing from the workspace schemas is reported per object",
			Input:      []string{"c_contract", "task"},
			Server:     schemasServer(),
			Comparator: testconn.ComparatorSubsetMetadata,
			Expected: &common.ListObjectMetadataResult{
				Result: map[string]common.ObjectMetadata{
					"task": {
						DisplayName: "Tasks",
						Fields: map[string]common.FieldMetadata{
							"assignee_id": {
								DisplayName:  "Assignee",
								ValueType:    common.ValueTypeReference,
								ProviderType: "many-to-one",
								ReferenceTo:  []string{"users"},
							},
						},
					},
				},
				Errors: map[string]error{
					"c_contract": mockutils.ExpectedSubsetErrors{common.ErrObjectNotSupported},
				},
			},
		},
		{
			Name:  "Schemas request failure is reported for each record object",
			Input: []string{"company", "users"},
			Server: mockserver.Fixed{
				Setup:  mockserver.ContentJSON(),
				Always: mockserver.Response(http.StatusNotFound, responseNotFound),
			}.Server(),
			Comparator: testconn.ComparatorSubsetMetadata,
			Expected: &common.ListObjectMetadataResult{
				Result: map[string]common.ObjectMetadata{
					"users": {
						DisplayName: "Users",
						Fields: map[string]common.FieldMetadata{
							"email": {
								DisplayName:  "Email",
								ValueType:    common.ValueTypeString,
								ProviderType: "string",
							},
						},
					},
				},
				Errors: map[string]error{
					"company": mockutils.ExpectedSubsetErrors{
						common.ErrNotFound,
						testutils.StringError("Workspace not found"),
					},
				},
			},
		},
		{
			Name:         "At least one object name must be queried",
			Input:        nil,
			Server:       mockserver.Dummy(),
			ExpectedErrs: []error{common.ErrMissingObjects},
		},
		{
			Name:       "Unknown object is reported per object",
			Input:      []string{"comments"},
			Server:     mockserver.Dummy(),
			Comparator: testconn.ComparatorSubsetMetadata,
			Expected: &common.ListObjectMetadataResult{
				Result: map[string]common.ObjectMetadata{},
				Errors: map[string]error{
					"comments": mockutils.ExpectedSubsetErrors{common.ErrObjectNotSupported},
				},
			},
		},
		{
			Name:       "Users come from the static catalog",
			Input:      []string{"users"},
			Server:     mockserver.Dummy(),
			Comparator: testconn.ComparatorSubsetMetadata,
			Expected: &common.ListObjectMetadataResult{
				Result: map[string]common.ObjectMetadata{
					"users": {
						DisplayName: "Users",
						Fields: map[string]common.FieldMetadata{
							"id": {
								DisplayName:  "Id",
								ValueType:    common.ValueTypeString,
								ProviderType: "string",
							},
							"email": {
								DisplayName:  "Email",
								ValueType:    common.ValueTypeString,
								ProviderType: "string",
							},
							"_updated_at": {
								DisplayName: "Updated At",
								ValueType:   common.ValueTypeDateTime,
							},
							"last_active_at": {
								DisplayName: "Last Active At",
								ValueType:   common.ValueTypeDateTime,
							},
							"is_service_account": {
								DisplayName:  "Is Service Account",
								ValueType:    common.ValueTypeBoolean,
								ProviderType: "boolean",
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
		Workspace:           "acme",
	})
	if err != nil {
		return nil, err
	}

	connector.SetBaseURL(mockutils.ReplaceURLOrigin(connector.HTTPClient().Base, serverURL))

	return connector, nil
}
