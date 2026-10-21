package clarify

import (
	"net/http"
	"sync/atomic"
	"testing"

	"github.com/amp-labs/connectors"
	"github.com/amp-labs/connectors/common"
	"github.com/amp-labs/connectors/common/urlbuilder"
	"github.com/amp-labs/connectors/test/utils/mockutils/mockserver"
	"github.com/amp-labs/connectors/test/utils/testutils"
)

func TestSchemasAreReadOncePerConnector(t *testing.T) {
	t.Parallel()

	responseSchemas := testutils.DataFromFile(t, "schemas.json")
	responseTask := testutils.DataFromFile(t, "task-first-page.json")

	var schemaRequests atomic.Int32

	server := mockserver.NewServer(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch r.URL.Path {
		case "/v1/workspaces/acme/schemas":
			schemaRequests.Add(1)

			_, _ = w.Write(responseSchemas)
		case "/v1/workspaces/acme/objects/task/resources":
			_, _ = w.Write(responseTask)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})
	defer server.Close()

	conn, err := constructTestConnector(server.URL)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := conn.ListObjectMetadata(t.Context(), []string{"task"}); err != nil {
		t.Fatal(err)
	}

	for range 2 {
		if _, err := conn.Read(t.Context(), common.ReadParams{
			ObjectName: "task",
			Fields:     connectors.Fields("title"),
		}); err != nil {
			t.Fatal(err)
		}
	}

	if got := schemaRequests.Load(); got != 1 {
		t.Fatalf("GET /schemas was sent %d times, want 1", got)
	}
}

func TestOnOriginMovesLinksToTheConnectorsOrigin(t *testing.T) {
	t.Parallel()

	origin, err := urlbuilder.New("https://api.clarify.ai/v1/workspaces/acme/schemas")
	if err != nil {
		t.Fatal(err)
	}

	got, err := onOrigin("http://api.clarify.ai/v1/workspaces/acme/schemas?page%5Bcursor%5D=abc", origin)
	if err != nil {
		t.Fatal(err)
	}

	if want := "https://api.clarify.ai/v1/workspaces/acme/schemas?page%5Bcursor%5D=abc"; got != want {
		t.Fatalf("got %s, want %s", got, want)
	}
}
