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

func TestInventoryAdjustmentServiceGet(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/company/123/inventoryadjustment/1455000036", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_, err := io.WriteString(w, `{"InventoryAdjustment":{"DocNumber":"3","TxnDate":"2024-05-02","PrivateNote":"","SyncToken":"0","domain":"QBO","AdjustAccountRef":{"name":"Inventory Shrinkage","value":"91"},"sparse":false,"Line":[{"ItemAdjustmentLineDetail":{"QtyDiff":-5,"ItemRef":{"value":"1010000001"}},"Id":"1"}],"Id":"1455000036"}}`)
		require.NoError(t, err)
	}))
	defer server.Close()

	client, err := NewClient(WithHTTPClient(server.Client()), WithBaseURL(server.URL+"/v3/company"))
	require.NoError(t, err)

	ia, err := client.Company("123", oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test-token"})).InventoryAdjustments().Get(context.Background(), "1455000036")
	require.NoError(t, err)
	assert.Equal(t, "1455000036", ia.ID())
	assert.Equal(t, EntityInventoryAdjustment, ia.EntityName())
	assert.Equal(t, "QBO", ia.CDCDomain())
	require.NotNil(t, ia.AdjustAccountRef)
	assert.Equal(t, "91", ia.AdjustAccountRef.Value)
	assert.Equal(t, "Inventory Shrinkage", ia.AdjustAccountRef.Name)
	require.Len(t, ia.Line, 1)
	require.NotNil(t, ia.Line[0].ItemAdjustmentLineDetail)
	assert.Equal(t, "-5", ia.Line[0].ItemAdjustmentLineDetail.QtyDiff.String())
}

func TestInventoryAdjustmentServiceQueryUsesQueryOptions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/company/123/query", r.URL.Path)
		assert.Equal(t, "SELECT * FROM InventoryAdjustment STARTPOSITION 11 MAXRESULTS 10", r.URL.Query().Get("query"))
		w.Header().Set("Content-Type", "application/json")
		_, err := io.WriteString(w, `{"QueryResponse":{"InventoryAdjustment":[{"Id":"1455000036","SyncToken":"0"},{"Id":"1455000081","SyncToken":"0"}],"startPosition":11,"maxResults":10}}`)
		require.NoError(t, err)
	}))
	defer server.Close()

	client, err := NewClient(WithHTTPClient(server.Client()), WithBaseURL(server.URL+"/v3/company"))
	require.NoError(t, err)

	ias, err := client.Company("123", oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test-token"})).InventoryAdjustments().ListPage(context.Background(), 2, 10)
	require.NoError(t, err)
	require.Len(t, ias, 2)
	assert.Equal(t, "1455000036", ias[0].ID())
	assert.Equal(t, "1455000081", ias[1].ID())
}

func TestInventoryAdjustmentJSONDecodesV75Fields(t *testing.T) {
	const fixture = `{
		"InventoryAdjustment": {
			"DocNumber": "27",
			"TxnDate": "2024-05-02",
			"PrivateNote": "Memo 1",
			"SyncToken": "0",
			"domain": "QBO",
			"AdjustAccountRef": {
				"name": "Inventory Shrinkage",
				"value": "91"
			},
			"sparse": false,
			"Line": [
				{
					"ItemAdjustmentLineDetail": {
						"QtyDiff": -2,
						"ItemRef": {
							"value": "1010000001"
						}
					},
					"Id": "1"
				}
			],
			"Id": "1455000081",
			"MetaData": {
				"LastUpdatedTime": "2024-05-07"
			}
		}
	}`

	var envelope struct {
		InventoryAdjustment InventoryAdjustment `json:"InventoryAdjustment"`
	}
	require.NoError(t, json.Unmarshal([]byte(fixture), &envelope))

	ia := envelope.InventoryAdjustment
	assert.Equal(t, "1455000081", ia.ID())
	assert.Equal(t, "0", ia.SyncToken())
	assert.Equal(t, "27", ia.DocNumber)
	assert.Equal(t, "Memo 1", ia.PrivateNote)

	require.NotNil(t, ia.TxnDate)
	assert.Equal(t, "2024-05-02", ia.TxnDate.String())

	require.NotNil(t, ia.AdjustAccountRef)
	assert.Equal(t, "91", ia.AdjustAccountRef.Value)
	assert.Equal(t, "Inventory Shrinkage", ia.AdjustAccountRef.Name)

	require.NotNil(t, ia.MetaData)
	require.NotNil(t, ia.MetaData.LastUpdatedTime)

	require.Len(t, ia.Line, 1)
	line := ia.Line[0]
	assert.Equal(t, "1", line.ID())
	require.NotNil(t, line.ItemAdjustmentLineDetail)
	assert.Equal(t, "-2", line.ItemAdjustmentLineDetail.QtyDiff.String())
	require.NotNil(t, line.ItemAdjustmentLineDetail.ItemRef)
	assert.Equal(t, "1010000001", line.ItemAdjustmentLineDetail.ItemRef.Value)
}

func TestInventoryAdjustmentRequestsMarshalWritablePayloads(t *testing.T) {
	create := InventoryAdjustment{
		AdjustAccountRef: &ReferenceType{Value: "91"},
		PrivateNote:      "Memo 1",
		TxnDate:          &Date{Time: time.Date(2024, 5, 2, 0, 0, 0, 0, time.UTC)},
		Line: []InventoryAdjustmentLine{
			{
				DetailType: ItemAdjustmentLineDetailType,
				ItemAdjustmentLineDetail: &ItemAdjustmentLineDetail{
					QtyDiff: json.Number("-2"),
					ItemRef: &ReferenceType{Value: "1010000001"},
				},
			},
		},
	}

	body, err := json.Marshal(create)
	require.NoError(t, err)

	assert.Contains(t, string(body), `"AdjustAccountRef":{"value":"91"}`)
	assert.Contains(t, string(body), `"TxnDate":"2024-05-02"`)
	assert.Contains(t, string(body), `"DetailType":"ItemAdjustmentLineDetail"`)
	assert.Contains(t, string(body), `"QtyDiff":-2`)
	assert.Contains(t, string(body), `"ItemRef":{"value":"1010000001"}`)
	assert.NotContains(t, string(body), `"Id"`)
	assert.NotContains(t, string(body), `"SyncToken"`)
	assert.NotContains(t, string(body), `"sparse"`)
}

func TestInventoryAdjustmentServiceCreate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/v3/company/123/inventoryadjustment", r.URL.Path)
		var payload map[string]interface{}
		require.NoError(t, json.NewDecoder(r.Body).Decode(&payload))
		assert.Equal(t, "2024-05-02", payload["TxnDate"])
		w.Header().Set("Content-Type", "application/json")
		_, err := io.WriteString(w, `{"InventoryAdjustment":{"Id":"1455000081","SyncToken":"0","DocNumber":"27","TxnDate":"2024-05-02","AdjustAccountRef":{"name":"Inventory Shrinkage","value":"91"},"Line":[{"ItemAdjustmentLineDetail":{"QtyDiff":-2,"ItemRef":{"value":"1010000001"}},"Id":"1"}]}}`)
		require.NoError(t, err)
	}))
	defer server.Close()

	client, err := NewClient(WithHTTPClient(server.Client()), WithBaseURL(server.URL+"/v3/company"))
	require.NoError(t, err)

	create := InventoryAdjustment{
		AdjustAccountRef: &ReferenceType{Value: "91"},
		TxnDate:          &Date{Time: time.Date(2024, 5, 2, 0, 0, 0, 0, time.UTC)},
		Line: []InventoryAdjustmentLine{
			{
				DetailType: ItemAdjustmentLineDetailType,
				ItemAdjustmentLineDetail: &ItemAdjustmentLineDetail{
					QtyDiff: json.Number("-2"),
					ItemRef: &ReferenceType{Value: "1010000001"},
				},
			},
		},
	}

	ia, err := client.Company("123", oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test-token"})).InventoryAdjustments().Create(context.Background(), create)
	require.NoError(t, err)
	assert.Equal(t, "1455000081", ia.ID())
	assert.Equal(t, "0", ia.SyncToken())
	require.Len(t, ia.Line, 1)
}

func TestInventoryAdjustmentServiceUpdateSparseInjectsSparseFlag(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/company/123/inventoryadjustment", r.URL.Path)
		var payload map[string]interface{}
		require.NoError(t, json.NewDecoder(r.Body).Decode(&payload))
		assert.Equal(t, true, payload["sparse"])
		assert.Equal(t, "1455000036", payload["Id"])
		assert.Equal(t, "0", payload["SyncToken"])
		w.Header().Set("Content-Type", "application/json")
		_, err := io.WriteString(w, `{"InventoryAdjustment":{"Id":"1455000036","SyncToken":"1"}}`)
		require.NoError(t, err)
	}))
	defer server.Close()

	client, err := NewClient(WithHTTPClient(server.Client()), WithBaseURL(server.URL+"/v3/company"))
	require.NoError(t, err)

	ia := InventoryAdjustment{
		IDValue:        IDValue{Id: "1455000036"},
		SyncTokenValue: SyncTokenValue{SyncTokenValue: "0"},
		PrivateNote:    "Update 1",
	}

	updated, err := client.Company("123", oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test-token"})).InventoryAdjustments().UpdateSparse(context.Background(), ia)
	require.NoError(t, err)
	assert.Equal(t, "1", updated.SyncToken())
}

func TestInventoryAdjustmentServiceDelete(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/company/123/inventoryadjustment", r.URL.Path)
		assert.Equal(t, "delete", r.URL.Query().Get("operation"))
		var payload map[string]interface{}
		require.NoError(t, json.NewDecoder(r.Body).Decode(&payload))
		assert.Equal(t, "1455000086", payload["Id"])
		assert.Equal(t, "0", payload["SyncToken"])
		w.Header().Set("Content-Type", "application/json")
		_, err := io.WriteString(w, `{"InventoryAdjustment":{"status":"Deleted","domain":"QBO","Id":"1455000086"}}`)
		require.NoError(t, err)
	}))
	defer server.Close()

	client, err := NewClient(WithHTTPClient(server.Client()), WithBaseURL(server.URL+"/v3/company"))
	require.NoError(t, err)

	req := DeleteRequest{}
	req.IDValue.Id = "1455000086"
	req.SyncTokenValue.SyncTokenValue = "0"

	ia, err := client.Company("123", oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test-token"})).InventoryAdjustments().Delete(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, "1455000086", ia.ID())
	assert.Equal(t, "Deleted", ia.CDCStatus())
}

func TestInventoryAdjustmentCDCExtraction(t *testing.T) {
	changedSince := time.Date(2026, time.March, 10, 12, 0, 0, 0, time.UTC)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/company/123/cdc", r.URL.Path)
		assert.Equal(t, string(EntityInventoryAdjustment), r.URL.Query().Get("entities"))
		assert.Equal(t, changedSince.Format(time.RFC3339), r.URL.Query().Get("changedSince"))
		w.Header().Set("Content-Type", "application/json")
		_, err := io.WriteString(w, `{"CDCResponse":[{"QueryResponse":[{"InventoryAdjustment":[{"Id":"1455000081","SyncToken":"0","domain":"QBO","Line":[{"ItemAdjustmentLineDetail":{"QtyDiff":-2.5,"ItemRef":{"value":"1010000001"}},"Id":"1"}]},{"Id":"1455000086","domain":"QBO","status":"Deleted"}],"startPosition":1,"maxResults":2}]}],"time":"2026-03-10T12:00:00Z"}`)
		require.NoError(t, err)
	}))
	defer server.Close()

	client, err := NewClient(WithHTTPClient(server.Client()), WithBaseURL(server.URL+"/v3/company"))
	require.NoError(t, err)

	ias, err := client.Company("123", oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test-token"})).InventoryAdjustments().ChangeDataCapture(context.Background(), changedSince)
	require.NoError(t, err)
	require.Len(t, ias, 2)

	assert.Equal(t, "1455000081", ias[0].ID())
	assert.Empty(t, ias[0].CDCStatus())
	require.Len(t, ias[0].Line, 1)
	require.NotNil(t, ias[0].Line[0].ItemAdjustmentLineDetail)
	assert.Equal(t, "-2.5", ias[0].Line[0].ItemAdjustmentLineDetail.QtyDiff.String())

	assert.Equal(t, "1455000086", ias[1].ID())
	assert.Equal(t, "Deleted", ias[1].CDCStatus())
}

// Compile-time interface compliance checks for InventoryAdjustment and
// InventoryAdjustmentService.
var (
	// InventoryAdjustment entity type compliance.
	_ Entity    = InventoryAdjustment{}
	_ CDCEntity = InventoryAdjustment{}

	// InventoryAdjustmentService operation compliance.
	_ Getter[InventoryAdjustment]          = (*InventoryAdjustmentService)(nil)
	_ Querier[InventoryAdjustment]         = (*InventoryAdjustmentService)(nil)
	_ Lister[InventoryAdjustment]          = (*InventoryAdjustmentService)(nil)
	_ Pager[InventoryAdjustment]           = (*InventoryAdjustmentService)(nil)
	_ Creator[InventoryAdjustment]         = (*InventoryAdjustmentService)(nil)
	_ SparseUpdater[InventoryAdjustment]   = (*InventoryAdjustmentService)(nil)
	_ Deleter[InventoryAdjustment]         = (*InventoryAdjustmentService)(nil)
	_ EntityCDCReader[InventoryAdjustment] = (*InventoryAdjustmentService)(nil)
)
