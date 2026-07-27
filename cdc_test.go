package quickbooks

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
)

// cdcResponseJSON is a representative CDC API response. The QueryResponse
// field is a JSON array — the shape that caused the original unmarshal failure.
const cdcResponseJSON = `{
	"CDCResponse": [
		{
			"QueryResponse": [
				{
					"Invoice": [
						{"Id":"1","SyncToken":"0","domain":"QBO","status":"Deleted","DocNumber":"INV-1"},
						{"Id":"2","SyncToken":"1","domain":"QBO","status":"Emailed","DocNumber":"INV-2"}
					],
					"startPosition": 1,
					"maxResults": 2
				}
			]
		}
	],
	"time": "2026-03-15T10:00:00-07:00"
}`

func TestCDCResponseUnmarshalsQueryResponseArray(t *testing.T) {
	var resp CDCResponse
	err := json.Unmarshal([]byte(cdcResponseJSON), &resp)
	require.NoError(t, err)

	require.Len(t, resp.CDCResponse, 1)
	require.Len(t, resp.CDCResponse[0].QueryResponse, 1)

	qr := resp.CDCResponse[0].QueryResponse[0]
	assert.Equal(t, 1, qr.StartPosition)
	assert.Equal(t, 2, qr.MaxResults)

	raw, err := qr.Entities(EntityInvoice)
	require.NoError(t, err)
	assert.NotEmpty(t, raw)
}

func TestExtractCDCEntitiesDecodesInvoices(t *testing.T) {
	var resp CDCResponse
	require.NoError(t, json.Unmarshal([]byte(cdcResponseJSON), &resp))

	invoices, err := ExtractCDCEntities[Invoice](&resp, EntityInvoice)
	require.NoError(t, err)
	require.Len(t, invoices, 2)

	assert.Equal(t, "1", invoices[0].ID())
	assert.Equal(t, "Deleted", invoices[0].CDCStatus())
	assert.Equal(t, "QBO", invoices[0].CDCDomain())
	assert.Equal(t, "INV-1", invoices[0].DocNumber)

	assert.Equal(t, "2", invoices[1].ID())
	assert.Equal(t, "Emailed", invoices[1].CDCStatus())
	assert.Equal(t, "INV-2", invoices[1].DocNumber)
}

func TestExtractCDCEntitiesReturnsNilForNilResponse(t *testing.T) {
	invoices, err := ExtractCDCEntities[Invoice](nil, EntityInvoice)
	require.NoError(t, err)
	assert.Nil(t, invoices)
}

func TestExtractCDCEntitiesHandlesMultipleCDCItems(t *testing.T) {
	// QBO may return multiple CDCResponse items when multiple entity types
	// are requested. Each contains its own QueryResponse array.
	multiJSON := `{
		"CDCResponse": [
			{
				"QueryResponse": [
					{
						"Invoice": [{"Id":"10","SyncToken":"0","domain":"QBO","status":"Deleted"}],
						"startPosition": 1,
						"maxResults": 1
					}
				]
			},
			{
				"QueryResponse": [
					{
						"Invoice": [{"Id":"20","SyncToken":"0","domain":"QBO","status":"Emailed"}],
						"startPosition": 1,
						"maxResults": 1
					}
				]
			}
		],
		"time": "2026-03-15T10:00:00-07:00"
	}`

	var resp CDCResponse
	require.NoError(t, json.Unmarshal([]byte(multiJSON), &resp))

	invoices, err := ExtractCDCEntities[Invoice](&resp, EntityInvoice)
	require.NoError(t, err)
	require.Len(t, invoices, 2)
	assert.Equal(t, "10", invoices[0].ID())
	assert.Equal(t, "20", invoices[1].ID())
}

func TestInvoiceServiceChangeDataCapture(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/company/123/cdc", r.URL.Path)
		assert.Equal(t, "Invoice", r.URL.Query().Get("entities"))
		assert.NotEmpty(t, r.URL.Query().Get("changedSince"))

		w.Header().Set("Content-Type", "application/json")
		_, err := io.WriteString(w, cdcResponseJSON)
		require.NoError(t, err)
	}))
	defer server.Close()

	client, err := NewClient(WithHTTPClient(server.Client()), WithBaseURL(server.URL+"/v3/company"))
	require.NoError(t, err)

	changedSince := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	invoices, err := client.Company("123", oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test-token"})).
		Invoices().ChangeDataCapture(context.Background(), changedSince)
	require.NoError(t, err)
	require.Len(t, invoices, 2)
	assert.Equal(t, "1", invoices[0].ID())
	assert.Equal(t, "2", invoices[1].ID())
}
