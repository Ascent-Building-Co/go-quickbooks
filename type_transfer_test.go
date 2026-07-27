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

func TestTransferServiceGet(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/company/123/transfer/170", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_, err := io.WriteString(w, `{"Transfer":{"Id":"170","SyncToken":"0","domain":"QBO","status":"Active","FromAccountRef":{"value":"35","name":"Checking"},"ToAccountRef":{"value":"36","name":"Savings"},"Amount":120.0}}`)
		require.NoError(t, err)
	}))
	defer server.Close()

	client, err := NewClient(WithHTTPClient(server.Client()), WithBaseURL(server.URL+"/v3/company"))
	require.NoError(t, err)

	tr, err := client.Company("123", oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test-token"})).Transfers().Get(context.Background(), "170")
	require.NoError(t, err)
	assert.Equal(t, "170", tr.ID())
	assert.Equal(t, EntityTransfer, tr.EntityName())
	assert.Equal(t, "Active", tr.CDCStatus())
	assert.Equal(t, "QBO", tr.CDCDomain())
	require.NotNil(t, tr.FromAccountRef)
	assert.Equal(t, "35", tr.FromAccountRef.Value)
	require.NotNil(t, tr.ToAccountRef)
	assert.Equal(t, "36", tr.ToAccountRef.Value)
	assert.Equal(t, "120.0", tr.Amount.String())
}

func TestTransferServiceQueryUsesQueryOptions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/company/123/query", r.URL.Path)
		assert.Equal(t, "SELECT * FROM Transfer STARTPOSITION 11 MAXRESULTS 10", r.URL.Query().Get("query"))
		w.Header().Set("Content-Type", "application/json")
		_, err := io.WriteString(w, `{"QueryResponse":{"Transfer":[{"Id":"170","SyncToken":"0"},{"Id":"171","SyncToken":"1"}],"startPosition":6,"maxResults":10}}`)
		require.NoError(t, err)
	}))
	defer server.Close()

	client, err := NewClient(WithHTTPClient(server.Client()), WithBaseURL(server.URL+"/v3/company"))
	require.NoError(t, err)

	trs, err := client.Company("123", oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test-token"})).Transfers().ListPage(context.Background(), 2, 10)
	require.NoError(t, err)
	require.Len(t, trs, 2)
	assert.Equal(t, "170", trs[0].ID())
	assert.Equal(t, "171", trs[1].ID())
}

func TestTransferJSONDecodesV75Fields(t *testing.T) {
	const fixture = `{
		"Transfer": {
			"Id": "170",
			"SyncToken": "0",
			"domain": "QBO",
			"status": "Active",
			"MetaData": {
				"CreateTime": "2015-02-06T11:06:12-08:00",
				"LastUpdatedTime": "2015-02-06T11:06:12-08:00"
			},
			"FromAccountRef": {"value": "35", "name": "Checking"},
			"ToAccountRef": {"value": "36", "name": "Savings"},
			"Amount": 120.0,
			"TxnDate": "2015-02-06",
			"PrivateNote": "Monthly transfer",
			"TransactionLocationType": "WithinFrance"
		}
	}`

	var envelope struct {
		Transfer Transfer `json:"Transfer"`
	}
	require.NoError(t, json.Unmarshal([]byte(fixture), &envelope))

	tr := envelope.Transfer
	assert.Equal(t, "170", tr.ID())
	assert.Equal(t, "0", tr.SyncToken())
	assert.Equal(t, "QBO", tr.CDCDomain())
	assert.Equal(t, "Active", tr.CDCStatus())

	require.NotNil(t, tr.MetaData)
	require.NotNil(t, tr.MetaData.CreateTime)

	require.NotNil(t, tr.FromAccountRef)
	assert.Equal(t, "35", tr.FromAccountRef.Value)
	assert.Equal(t, "Checking", tr.FromAccountRef.Name)

	require.NotNil(t, tr.ToAccountRef)
	assert.Equal(t, "36", tr.ToAccountRef.Value)
	assert.Equal(t, "Savings", tr.ToAccountRef.Name)

	assert.Equal(t, "120.0", tr.Amount.String())

	require.NotNil(t, tr.TxnDate)
	assert.Equal(t, "2015-02-06", tr.TxnDate.String())

	assert.Equal(t, "Monthly transfer", tr.PrivateNote)
	assert.Equal(t, "WithinFrance", tr.TransactionLocationType)
}

func TestTransferRequestsMarshalWritablePayloads(t *testing.T) {
	create := Transfer{
		FromAccountRef: &ReferenceType{Value: "35"},
		ToAccountRef:   &ReferenceType{Value: "36"},
		Amount:         json.Number("120.00"),
	}

	body, err := json.Marshal(create)
	require.NoError(t, err)
	assert.Contains(t, string(body), `"FromAccountRef"`)
	assert.Contains(t, string(body), `"ToAccountRef"`)
	assert.Contains(t, string(body), `"Amount":120.00`)
	assert.NotContains(t, string(body), `"sparse"`)
	assert.NotContains(t, string(body), `"RecurDataRef"`)
}

func TestTransferServiceUpdateSparseInjectsSparseFlag(t *testing.T) {
	var capturedBody []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/company/123/transfer", r.URL.Path)
		var err error
		capturedBody, err = io.ReadAll(r.Body)
		require.NoError(t, err)
		w.Header().Set("Content-Type", "application/json")
		_, err = io.WriteString(w, `{"Transfer":{"Id":"170","SyncToken":"2"}}`)
		require.NoError(t, err)
	}))
	defer server.Close()

	client, err := NewClient(WithHTTPClient(server.Client()), WithBaseURL(server.URL+"/v3/company"))
	require.NoError(t, err)

	transfer := Transfer{
		IDValue:        IDValue{Id: "170"},
		SyncTokenValue: SyncTokenValue{SyncTokenValue: "1"},
		FromAccountRef: &ReferenceType{Value: "35"},
		ToAccountRef:   &ReferenceType{Value: "36"},
		Amount:         json.Number("660.0"),
	}

	_, err = client.Company("123", oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test-token"})).Transfers().UpdateSparse(context.Background(), transfer)
	require.NoError(t, err)

	assert.Contains(t, string(capturedBody), `"Id":"170"`)
	assert.Contains(t, string(capturedBody), `"SyncToken":"1"`)
	assert.Contains(t, string(capturedBody), `"sparse":true`)
	assert.Contains(t, string(capturedBody), `"Amount":660.0`)
}

func TestTransferCDCExtraction(t *testing.T) {
	changedSince := time.Date(2026, time.March, 10, 12, 0, 0, 0, time.UTC)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/company/123/cdc", r.URL.Path)
		assert.Equal(t, string(EntityTransfer), r.URL.Query().Get("entities"))
		assert.Equal(t, changedSince.Format(time.RFC3339), r.URL.Query().Get("changedSince"))
		w.Header().Set("Content-Type", "application/json")
		_, err := io.WriteString(w, `{"CDCResponse":[{"QueryResponse":[{"Transfer":[{"Id":"170","SyncToken":"1","domain":"QBO","status":"Deleted"}],"startPosition":1,"maxResults":1}]}],"time":"2026-03-10T12:00:00Z"}`)
		require.NoError(t, err)
	}))
	defer server.Close()

	client, err := NewClient(WithHTTPClient(server.Client()), WithBaseURL(server.URL+"/v3/company"))
	require.NoError(t, err)

	trs, err := client.Company("123", oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test-token"})).Transfers().ChangeDataCapture(context.Background(), changedSince)
	require.NoError(t, err)
	require.Len(t, trs, 1)
	assert.Equal(t, "170", trs[0].ID())
	assert.Equal(t, "Deleted", trs[0].CDCStatus())
}

// Compile-time interface compliance checks for Transfer and TransferService.
var (
	// Transfer entity type compliance.
	_ Entity    = Transfer{}
	_ CDCEntity = Transfer{}

	// TransferService operation compliance.
	_ Getter[Transfer]          = (*TransferService)(nil)
	_ Querier[Transfer]         = (*TransferService)(nil)
	_ Lister[Transfer]          = (*TransferService)(nil)
	_ Pager[Transfer]           = (*TransferService)(nil)
	_ Creator[Transfer]         = (*TransferService)(nil)
	_ Updater[Transfer]         = (*TransferService)(nil)
	_ SparseUpdater[Transfer]   = (*TransferService)(nil)
	_ Deleter[Transfer]         = (*TransferService)(nil)
	_ EntityCDCReader[Transfer] = (*TransferService)(nil)
)
