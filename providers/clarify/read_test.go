package clarify

import (
	"net/http"
	"testing"
	"time"

	"github.com/amp-labs/connectors"
	"github.com/amp-labs/connectors/common"
	"github.com/amp-labs/connectors/test/utils/mockutils/mockcond"
	"github.com/amp-labs/connectors/test/utils/mockutils/mockserver"
	"github.com/amp-labs/connectors/test/utils/testconn"
	"github.com/amp-labs/connectors/test/utils/testutils"
)

func TestRead(t *testing.T) { //nolint:funlen,maintidx
	t.Parallel()

	responseSchemas := testutils.DataFromFile(t, "schemas.json")
	responsePersonIncluded := testutils.DataFromFile(t, "person-included.json")
	responseTaskFirstPage := testutils.DataFromFile(t, "task-first-page.json")
	responseTaskEmpty := testutils.DataFromFile(t, "task-empty.json")
	responseUsers := testutils.DataFromFile(t, "users.json")
	responseNotFound := testutils.DataFromFile(t, "error-workspace-not-found.json")

	schemasCase := mockserver.Case{
		If:   mockcond.And{mockcond.MethodGET(), mockcond.Path("/v1/workspaces/acme/schemas")},
		Then: mockserver.Response(http.StatusOK, responseSchemas),
	}

	tests := []testconn.TestCaseRead{
		{
			Name:         "Read object must be included",
			Input:        common.ReadParams{Fields: connectors.Fields("id")},
			Server:       mockserver.Dummy(),
			ExpectedErrs: []error{common.ErrMissingObjects},
		},
		{
			Name:         "At least one field is requested",
			Input:        common.ReadParams{ObjectName: "person"},
			Server:       mockserver.Dummy(),
			ExpectedErrs: []error{common.ErrMissingFields},
		},
		{
			Name:         "Unknown object is not supported",
			Input:        common.ReadParams{ObjectName: "comments", Fields: connectors.Fields("id")},
			Server:       mockserver.Dummy(),
			ExpectedErrs: []error{common.ErrOperationNotSupportedForObject},
		},
		{
			Name:         "Bare custom object prefix is not an object",
			Input:        common.ReadParams{ObjectName: "c_", Fields: connectors.Fields("id")},
			Server:       mockserver.Dummy(),
			ExpectedErrs: []error{common.ErrOperationNotSupportedForObject},
		},
		{
			Name: "Users have no deleted records endpoint",
			Input: common.ReadParams{
				ObjectName: "users",
				Fields:     connectors.Fields("id"),
				Deleted:    true,
			},
			Server:       mockserver.Dummy(),
			ExpectedErrs: []error{common.ErrOperationNotSupportedForObject},
		},
		{
			Name:  "JSON:API error response is parsed",
			Input: common.ReadParams{ObjectName: "users", Fields: connectors.Fields("id")},
			Server: mockserver.Fixed{
				Setup:  mockserver.ContentJSON(),
				Always: mockserver.Response(http.StatusNotFound, responseNotFound),
			}.Server(),
			ExpectedErrs: []error{
				common.ErrNotFound,
				testutils.StringError("Workspace not found"),
			},
		},
		{
			Name: "Person attributes are flattened and requested to-many relationships are included",
			Input: common.ReadParams{
				ObjectName: "person",
				Fields:     connectors.Fields("name", "job_title", "company_id", "deals", "_updated_at"),
				PageSize:   2,
			},
			Server: mockserver.Switch{
				Setup: mockserver.ContentJSON(),
				Cases: []mockserver.Case{
					schemasCase,
					{
						If: mockcond.And{
							mockcond.MethodGET(),
							mockcond.Path("/v1/workspaces/acme/objects/person/resources"),
							mockcond.QueryParam("page[limit]", "2"),
							mockcond.QueryParam("include", "deals"),
						},
						Then: mockserver.Response(http.StatusOK, responsePersonIncluded),
					},
				},
			}.Server(),
			Comparator: testconn.ComparatorSubsetRead,
			Expected: &common.ReadResult{
				Rows: 1,
				Data: []common.ReadResultRow{{
					Id: "468acaf0-d6dd-4ee3-a5ae-8c9b3c243781",
					Fields: map[string]any{
						"name": map[string]any{
							"full_name":  "Rajat Pawar",
							"first_name": "Rajat",
							"last_name":  "Pawar",
						},
						"job_title":   "Founding Engineer",
						"company_id":  "0adf6280-c5d7-4b64-9de2-db7bc81ba0e5",
						"deals":       []any{},
						"_updated_at": "2026-10-21T06:41:44.735Z",
					},
					Raw: map[string]any{
						"id":   "468acaf0-d6dd-4ee3-a5ae-8c9b3c243781",
						"type": "person",
					},
				}},
				NextPage: "",
				Done:     true,
			},
		},
		{
			Name: "First page of tasks links to the next offset on the request's own URL",
			Input: common.ReadParams{
				ObjectName: "task",
				Fields:     connectors.Fields("title", "status"),
				PageSize:   2,
			},
			Server: mockserver.Switch{
				Setup: mockserver.ContentJSON(),
				Cases: []mockserver.Case{
					schemasCase,
					{
						If: mockcond.And{
							mockcond.Path("/v1/workspaces/acme/objects/task/resources"),
							mockcond.QueryParam("page[limit]", "2"),
							mockcond.QueryParamsMissing("include", "page[offset]"),
						},
						Then: mockserver.Response(http.StatusOK, responseTaskFirstPage),
					},
				},
			}.Server(),
			Comparator: testconn.ComparatorSubsetRead,
			Expected: &common.ReadResult{
				Rows: 2,
				Data: []common.ReadResultRow{{
					Id: "1ce3d348-0186-4584-b4a8-cd15d807e8fb",
					Fields: map[string]any{
						"title":  "Download the desktop app",
						"status": "To Do",
					},
					Raw: map[string]any{"type": "task"},
				}, {
					Id: "4f7ab923-fbc4-4e84-b8d0-1b1f3b34a1ec",
					Fields: map[string]any{
						"title":  "Use Ask for Meetings and Deals",
						"status": "To Do",
					},
					Raw: map[string]any{"type": "task"},
				}},
				NextPage: testconn.URLTestServer + "/v1/workspaces/acme/objects/task/resources?page%5Blimit%5D=2&page%5Boffset%5D=2",
				Done:     false,
			},
		},
		{
			Name: "Incremental read sends a whole-second window and trims it to (Since, Until]",
			Input: common.ReadParams{
				ObjectName: "task",
				Fields:     connectors.Fields("title"),
				PageSize:   2,
				// Both tasks were updated at 06:42:47.281Z, which Since excludes.
				Since: time.Date(2026, 10, 21, 6, 42, 47, 281000000, time.UTC),
				Until: time.Date(2026, 10, 21, 6, 42, 47, 500000000, time.UTC),
			},
			Server: mockserver.Switch{
				Setup: mockserver.ContentJSON(),
				Cases: []mockserver.Case{
					schemasCase,
					{
						If: mockcond.And{
							mockcond.Path("/v1/workspaces/acme/objects/task/resources"),
							mockcond.QueryParam("filter[_updated_at][Is on or after]", "2026-10-21T06:42:47Z"),
							mockcond.QueryParam("filter[_updated_at][Is before]", "2026-10-21T06:42:48Z"),
						},
						Then: mockserver.Response(http.StatusOK, responseTaskFirstPage),
					},
				},
			}.Server(),
			Comparator: testconn.ComparatorPagination,
			Expected: &common.ReadResult{
				Rows:     0,
				NextPage: testconn.URLTestServer + "/v1/workspaces/acme/objects/task/resources?filter%5B_updated_at%5D%5BIs%20before%5D=2026-10-21T06%3A42%3A48Z&filter%5B_updated_at%5D%5BIs%20on%20or%20after%5D=2026-10-21T06%3A42%3A47Z&page%5Blimit%5D=2&page%5Boffset%5D=2", // nolint:lll
				Done:     false,
			},
		},
		{
			Name: "Until is inclusive at the record's own millisecond",
			Input: common.ReadParams{
				ObjectName: "task",
				Fields:     connectors.Fields("title"),
				PageSize:   2,
				Until:      time.Date(2026, 10, 21, 6, 42, 47, 281000000, time.UTC),
			},
			Server: mockserver.Switch{
				Setup: mockserver.ContentJSON(),
				Cases: []mockserver.Case{
					schemasCase,
					{
						If: mockcond.And{
							mockcond.Path("/v1/workspaces/acme/objects/task/resources"),
							mockcond.QueryParam("filter[_updated_at][Is before]", "2026-10-21T06:42:48Z"),
							mockcond.QueryParamsMissing("filter[_updated_at][Is on or after]"),
						},
						Then: mockserver.Response(http.StatusOK, responseTaskFirstPage),
					},
				},
			}.Server(),
			Comparator: testconn.ComparatorSubsetRead,
			Expected: &common.ReadResult{
				Rows: 2,
				Data: []common.ReadResultRow{{
					Id:     "1ce3d348-0186-4584-b4a8-cd15d807e8fb",
					Fields: map[string]any{"title": "Download the desktop app"},
					Raw:    map[string]any{"id": "1ce3d348-0186-4584-b4a8-cd15d807e8fb"},
				}, {
					Id:     "4f7ab923-fbc4-4e84-b8d0-1b1f3b34a1ec",
					Fields: map[string]any{"title": "Use Ask for Meetings and Deals"},
					Raw:    map[string]any{"id": "4f7ab923-fbc4-4e84-b8d0-1b1f3b34a1ec"},
				}},
				NextPage: testconn.URLTestServer + "/v1/workspaces/acme/objects/task/resources?filter%5B_updated_at%5D%5BIs%20before%5D=2026-10-21T06%3A42%3A48Z&page%5Blimit%5D=2&page%5Boffset%5D=2", // nolint:lll
				Done:     false,
			},
		},
		{
			Name: "Users are filtered by the connector, which sends no time filter",
			Input: common.ReadParams{
				ObjectName: "users",
				Fields:     connectors.Fields("email"),
				Since:      time.Date(2026, 10, 21, 6, 41, 44, 421000000, time.UTC),
			},
			Server: mockserver.Conditional{
				Setup: mockserver.ContentJSON(),
				If: mockcond.And{
					mockcond.Path("/v1/workspaces/acme/users"),
					mockcond.QueryParam("page[limit]", "500"),
					mockcond.QueryParamsMissing("filter[_updated_at][Is on or after]"),
				},
				Then: mockserver.Response(http.StatusOK, responseUsers),
			}.Server(),
			Comparator: testconn.ComparatorPagination,
			Expected: &common.ReadResult{
				Rows:     0,
				NextPage: "",
				Done:     true,
			},
		},
		{
			Name: "Users are read with their email and id",
			Input: common.ReadParams{
				ObjectName: "users",
				Fields:     connectors.Fields("email", "last_active_at"),
			},
			Server: mockserver.Conditional{
				Setup: mockserver.ContentJSON(),
				If:    mockcond.Path("/v1/workspaces/acme/users"),
				Then:  mockserver.Response(http.StatusOK, responseUsers),
			}.Server(),
			Comparator: testconn.ComparatorSubsetRead,
			Expected: &common.ReadResult{
				Rows: 1,
				Data: []common.ReadResultRow{{
					Id: "user_01K0CTE7EWH9CRGQDF1SV7AXX6",
					Fields: map[string]any{
						"email":          "rajat@withampersand.com",
						"last_active_at": "2026-10-21T06:41:46.022Z",
					},
					Raw: map[string]any{"type": "user"},
				}},
				Done: true,
			},
		},
		{
			Name: "Deleted records come from deleted-resources, windowed in epoch seconds",
			Input: common.ReadParams{
				ObjectName: "task",
				Fields:     connectors.Fields("title"),
				Deleted:    true,
				Since:      time.Date(2026, 10, 21, 6, 42, 47, 281000000, time.UTC),
				Until:      time.Date(2026, 10, 21, 7, 0, 0, 500000000, time.UTC),
			},
			Server: mockserver.Conditional{
				Setup: mockserver.ContentJSON(),
				If: mockcond.And{
					mockcond.Path("/v1/workspaces/acme/objects/task/deleted-resources"),
					mockcond.QueryParam("filter[_deleted_at][Greater than]", "1792564967"),
					mockcond.QueryParam("filter[_deleted_at][Less than or equal]", "1792566001"),
				},
				Then: mockserver.Response(http.StatusOK, responseTaskEmpty),
			}.Server(),
			Comparator: testconn.ComparatorPagination,
			Expected: &common.ReadResult{
				Rows:     0,
				NextPage: "",
				Done:     true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			t.Parallel()

			tt.Run(t, func() (testconn.TestableReader, error) {
				return constructTestConnector(tt.Server.URL)
			})
		})
	}
}
