package sybill

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

	// Saved from the live read; the test account has no conversations, deals or accounts yet.
	responseConversationsEmpty := testutils.DataFromFile(t, "conversations-empty.json")
	responseDealsEmpty := testutils.DataFromFile(t, "deals-empty.json")
	responseAccountsEmpty := testutils.DataFromFile(t, "accounts-empty.json")
	// The live 404 body of GET /v1/conversations/{conversationId} for an unknown id.
	responseNotFound := testutils.DataFromFile(t, "error-not-found.json")

	tests := []testconn.TestCaseRead{
		{
			Name:         "Read object must be included",
			Input:        common.ReadParams{Fields: connectors.Fields("conversationId")},
			Server:       mockserver.Dummy(),
			ExpectedErrs: []error{common.ErrMissingObjects},
		},
		{
			Name:         "At least one field is requested",
			Input:        common.ReadParams{ObjectName: "conversations"},
			Server:       mockserver.Dummy(),
			ExpectedErrs: []error{common.ErrMissingFields},
		},
		{
			Name:         "Unknown object is not supported",
			Input:        common.ReadParams{ObjectName: "messages", Fields: connectors.Fields("messageId")},
			Server:       mockserver.Dummy(),
			ExpectedErrs: []error{common.ErrOperationNotSupportedForObject},
		},
		{
			Name:  "Provider error body is mapped to its status and detail message",
			Input: common.ReadParams{ObjectName: "conversations", Fields: connectors.Fields("conversationId")},
			Server: mockserver.Fixed{
				Setup:  mockserver.ContentJSON(),
				Always: mockserver.Response(http.StatusNotFound, responseNotFound),
			}.Server(),
			ExpectedErrs: []error{
				common.ErrNotFound,
				// Only a matched {"detail"} format renders this; the fallback dumps the raw body instead.
				testutils.StringError("not found: Conversation not found"),
			},
		},
		{
			Name: "Conversations honour the page size and stop when hasMore is false",
			Input: common.ReadParams{
				ObjectName: "conversations",
				Fields:     connectors.Fields("conversationId", "title"),
				PageSize:   2,
			},
			Server: mockserver.Conditional{
				Setup: mockserver.ContentJSON(),
				If: mockcond.And{
					mockcond.MethodGET(),
					mockcond.Path("/v1/conversations"),
					mockcond.QueryParam("limit", "2"),
					mockcond.QueryParamsMissing("startedAfter", "startedBefore", "cursor"),
				},
				Then: mockserver.Response(http.StatusOK, responseConversationsEmpty),
			}.Server(),
			Comparator: testconn.ComparatorPagination,
			Expected:   &common.ReadResult{Rows: 0, NextPage: "", Done: true},
		},
		{
			Name: "Incremental conversations move Since back by the lookback and send Until as is",
			Input: common.ReadParams{
				ObjectName: "conversations",
				Fields:     connectors.Fields("conversationId", "transcript"),
				Since:      time.Date(2026, 10, 20, 12, 0, 0, 0, time.UTC),
				Until:      time.Date(2026, 10, 21, 12, 30, 15, 0, time.UTC),
			},
			Server: mockserver.Conditional{
				Setup: mockserver.ContentJSON(),
				If: mockcond.And{
					mockcond.Path("/v1/conversations"),
					// Detail fields cost a request per record, so the default page is small.
					mockcond.QueryParam("limit", "10"),
					mockcond.QueryParam("startedAfter", "2026-10-19T12:00:00Z"),
					mockcond.QueryParam("startedBefore", "2026-10-21T12:30:15Z"),
				},
				Then: mockserver.Response(http.StatusOK, responseConversationsEmpty),
			}.Server(),
			Comparator: testconn.ComparatorPagination,
			Expected:   &common.ReadResult{Rows: 0, NextPage: "", Done: true},
		},
		{
			Name: "Next page repeats the filters with the cursor",
			Input: common.ReadParams{
				ObjectName: "conversations",
				Fields:     connectors.Fields("conversationId"),
				Since:      time.Date(2026, 10, 20, 12, 0, 0, 0, time.UTC),
				NextPage:   "eyJzZWFyY2hBZnRlciI6",
			},
			Server: mockserver.Conditional{
				Setup: mockserver.ContentJSON(),
				If: mockcond.And{
					mockcond.Path("/v1/conversations"),
					mockcond.QueryParam("limit", "50"),
					mockcond.QueryParam("startedAfter", "2026-10-19T12:00:00Z"),
					mockcond.QueryParam("cursor", "eyJzZWFyY2hBZnRlciI6"),
				},
				Then: mockserver.Response(http.StatusOK, responseConversationsEmpty),
			}.Server(),
			Comparator: testconn.ComparatorPagination,
			Expected:   &common.ReadResult{Rows: 0, NextPage: "", Done: true},
		},
		{
			Name: "Deals are read in full, ignoring Since and Until",
			Input: common.ReadParams{
				ObjectName: "deals",
				Fields:     connectors.Fields("dealId", "name"),
				Since:      time.Date(2026, 10, 20, 12, 0, 0, 0, time.UTC),
				Until:      time.Date(2026, 10, 21, 12, 0, 0, 0, time.UTC),
			},
			Server: mockserver.Conditional{
				Setup: mockserver.ContentJSON(),
				If: mockcond.And{
					mockcond.Path("/v1/deals"),
					mockcond.QueryParam("limit", "50"),
					mockcond.QueryParamsMissing("lastActivityAfter", "lastActivityBefore",
						"startedAfter", "startedBefore"),
				},
				Then: mockserver.Response(http.StatusOK, responseDealsEmpty),
			}.Server(),
			Comparator: testconn.ComparatorPagination,
			Expected:   &common.ReadResult{Rows: 0, NextPage: "", Done: true},
		},
		{
			Name: "Accounts page size is capped at 50",
			Input: common.ReadParams{
				ObjectName: "accounts",
				Fields:     connectors.Fields("accountId", "name"),
				PageSize:   200,
				Since:      time.Date(2026, 10, 20, 12, 0, 0, 0, time.UTC),
			},
			Server: mockserver.Conditional{
				Setup: mockserver.ContentJSON(),
				If: mockcond.And{
					mockcond.Path("/v1/accounts"),
					mockcond.QueryParam("limit", "50"),
					mockcond.QueryParamsMissing("createdAfter", "lastActivityAfter"),
				},
				Then: mockserver.Response(http.StatusOK, responseAccountsEmpty),
			}.Server(),
			Comparator: testconn.ComparatorPagination,
			Expected:   &common.ReadResult{Rows: 0, NextPage: "", Done: true},
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
