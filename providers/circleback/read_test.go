package circleback

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

func TestRead(t *testing.T) { //nolint:funlen
	t.Parallel()

	// Saved from the live read; the test account has no meetings yet.
	responseEmpty := testutils.DataFromFile(t, "meetings-empty.json")
	// The documented 429 body (https://circleback.ai/docs/api#rate-limits).
	responseRateLimited := testutils.DataFromFile(t, "error-rate-limited.json")

	tests := []testconn.TestCaseRead{
		{
			Name:         "Read object must be included",
			Input:        common.ReadParams{Fields: connectors.Fields("id")},
			Server:       mockserver.Dummy(),
			ExpectedErrs: []error{common.ErrMissingObjects},
		},
		{
			Name:         "At least one field is requested",
			Input:        common.ReadParams{ObjectName: "meetings"},
			Server:       mockserver.Dummy(),
			ExpectedErrs: []error{common.ErrMissingFields},
		},
		{
			Name:         "Unknown object is not supported",
			Input:        common.ReadParams{ObjectName: "calendar/events", Fields: connectors.Fields("id")},
			Server:       mockserver.Dummy(),
			ExpectedErrs: []error{common.ErrOperationNotSupportedForObject},
		},
		{
			Name:  "Error envelope is parsed",
			Input: common.ReadParams{ObjectName: "tag", Fields: connectors.Fields("id")},
			Server: mockserver.Fixed{
				Setup:  mockserver.ContentJSON(),
				Always: mockserver.Response(http.StatusTooManyRequests, responseRateLimited),
			}.Server(),
			ExpectedErrs: []error{
				common.ErrLimitExceeded,
				testutils.StringError("Ratelimit exceeded for the Free plan. (RATE_LIMITED)"),
			},
		},
		{
			Name:  "Meetings list every accessible meeting and follow the Link header",
			Input: common.ReadParams{ObjectName: "meetings", Fields: connectors.Fields("id")},
			Server: mockserver.Conditional{
				Setup: mockserver.ContentJSON(),
				If: mockcond.And{
					mockcond.MethodGET(),
					mockcond.Path("/api/meetings"),
					mockcond.QueryParam("ownership", "All"),
				},
				Then: mockserver.ResponseChainedFuncs(
					mockserver.Header("Link", `</api/meetings?ownership=All&cursor=eyJwYWdlIjoxfQ>; rel="next"`),
					mockserver.Response(http.StatusOK, responseEmpty),
				),
			}.Server(),
			Comparator: testconn.ComparatorPagination,
			Expected: &common.ReadResult{
				Rows:     0,
				Data:     []common.ReadResultRow{},
				NextPage: testconn.URLTestServer + "/api/meetings?ownership=All&cursor=eyJwYWdlIjoxfQ",
				Done:     false,
			},
		},
		{
			Name: "Next page URL is requested as given",
			Input: common.ReadParams{
				ObjectName: "meetings",
				Fields:     connectors.Fields("id"),
				NextPage:   testconn.URLTestServer + "/api/meetings?ownership=All&cursor=eyJwYWdlIjoxfQ",
			},
			Server: mockserver.Conditional{
				Setup: mockserver.ContentJSON(),
				If: mockcond.And{
					mockcond.Path("/api/meetings"),
					mockcond.QueryParam("ownership", "All"),
					mockcond.QueryParam("cursor", "eyJwYWdlIjoxfQ"),
				},
				Then: mockserver.Response(http.StatusOK, responseEmpty),
			}.Server(),
			Comparator: testconn.ComparatorPagination,
			Expected:   &common.ReadResult{Rows: 0, Data: []common.ReadResultRow{}, Done: true},
		},
		{
			Name: "Meetings in a time window still read every page",
			Input: common.ReadParams{
				ObjectName: "meetings",
				Fields:     connectors.Fields("id"),
				Since:      time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC),
				Until:      time.Date(2026, 10, 2, 0, 0, 0, 0, time.UTC),
			},
			Server: mockserver.Conditional{
				Setup: mockserver.ContentJSON(),
				If:    mockcond.Path("/api/meetings"),
				Then: mockserver.ResponseChainedFuncs(
					mockserver.Header("Link", `</api/meetings?ownership=All&cursor=eyJwYWdlIjoxfQ>; rel="next"`),
					mockserver.Response(http.StatusOK, responseEmpty),
				),
			}.Server(),
			Comparator: testconn.ComparatorPagination,
			Expected: &common.ReadResult{
				Rows:     0,
				Data:     []common.ReadResultRow{},
				NextPage: testconn.URLTestServer + "/api/meetings?ownership=All&cursor=eyJwYWdlIjoxfQ",
				Done:     false,
			},
		},
		{
			Name:  "Action items are read for every assignee",
			Input: common.ReadParams{ObjectName: "action-items", Fields: connectors.Fields("id")},
			Server: mockserver.Conditional{
				Setup: mockserver.ContentJSON(),
				If: mockcond.And{
					mockcond.Path("/api/action-items"),
					mockcond.QueryParam("assigneeType", "Anyone"),
				},
				Then: mockserver.Response(http.StatusOK, responseEmpty),
			}.Server(),
			Comparator: testconn.ComparatorPagination,
			Expected:   &common.ReadResult{Rows: 0, Data: []common.ReadResultRow{}, Done: true},
		},
		{
			Name:  "People honour the page size",
			Input: common.ReadParams{ObjectName: "people", Fields: connectors.Fields("id"), PageSize: 2},
			Server: mockserver.Conditional{
				Setup: mockserver.ContentJSON(),
				If: mockcond.And{
					mockcond.Path("/api/people"),
					mockcond.QueryParam("limit", "2"),
				},
				Then: mockserver.Response(http.StatusOK, responseEmpty),
			}.Server(),
			Comparator: testconn.ComparatorPagination,
			Expected:   &common.ReadResult{Rows: 0, Data: []common.ReadResultRow{}, Done: true},
		},
		{
			Name:  "Companies list without parameters",
			Input: common.ReadParams{ObjectName: "companies", Fields: connectors.Fields("domain"), PageSize: 2},
			Server: mockserver.Conditional{
				Setup: mockserver.ContentJSON(),
				If: mockcond.And{
					mockcond.Path("/api/companies"),
					mockcond.QueryParamsMissing("limit"),
				},
				Then: mockserver.Response(http.StatusOK, responseEmpty),
			}.Server(),
			Comparator: testconn.ComparatorPagination,
			Expected:   &common.ReadResult{Rows: 0, Data: []common.ReadResultRow{}, Done: true},
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
