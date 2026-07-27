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

func TestPaymentMethodServiceGet(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/company/123/paymentmethod/5000000000000007325", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_, err := io.WriteString(w, `{"PaymentMethod":{"Id":"5000000000000007325","SyncToken":"0","domain":"QBO","status":"Active","Name":"Visa","Type":"CREDIT_CARD","Active":true}}`)
		require.NoError(t, err)
	}))
	defer server.Close()

	client, err := NewClient(WithHTTPClient(server.Client()), WithBaseURL(server.URL+"/v3/company"))
	require.NoError(t, err)

	pm, err := client.Company("123", oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test-token"})).PaymentMethods().Get(context.Background(), "5000000000000007325")
	require.NoError(t, err)
	assert.Equal(t, "5000000000000007325", pm.ID())
	assert.Equal(t, EntityPaymentMethod, pm.EntityName())
	assert.Equal(t, "Active", pm.CDCStatus())
	assert.Equal(t, "QBO", pm.CDCDomain())
	assert.Equal(t, "Visa", pm.Name)
	assert.Equal(t, "CREDIT_CARD", pm.Type)
	require.NotNil(t, pm.Active)
	assert.True(t, *pm.Active)
}

func TestPaymentMethodServiceListPage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/company/123/query", r.URL.Path)
		assert.Equal(t, "SELECT * FROM PaymentMethod STARTPOSITION 1 MAXRESULTS 5", r.URL.Query().Get("query"))
		w.Header().Set("Content-Type", "application/json")
		_, err := io.WriteString(w, `{"QueryResponse":{"PaymentMethod":[{"Id":"1","SyncToken":"0","Name":"Cash"},{"Id":"2","SyncToken":"0","Name":"Check"}],"startPosition":1,"maxResults":2}}`)
		require.NoError(t, err)
	}))
	defer server.Close()

	client, err := NewClient(WithHTTPClient(server.Client()), WithBaseURL(server.URL+"/v3/company"))
	require.NoError(t, err)

	pms, err := client.Company("123", oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test-token"})).PaymentMethods().ListPage(context.Background(), 1, 5)
	require.NoError(t, err)
	require.Len(t, pms, 2)
	assert.Equal(t, "1", pms[0].ID())
	assert.Equal(t, "Cash", pms[0].Name)
	assert.Equal(t, "2", pms[1].ID())
	assert.Equal(t, "Check", pms[1].Name)
}

func TestPaymentMethodJSONDecodesV75Fields(t *testing.T) {
	const fixture = `{
		"PaymentMethod": {
			"Id": "5000000000000007325",
			"SyncToken": "1",
			"domain": "QBO",
			"status": "Active",
			"MetaData": {
				"CreateTime": "2015-07-22T13:57:27-07:00",
				"LastUpdatedTime": "2015-07-22T13:57:27-07:00"
			},
			"Name": "Visa",
			"Type": "CREDIT_CARD",
			"Active": true
		}
	}`

	var envelope struct {
		PaymentMethod PaymentMethod `json:"PaymentMethod"`
	}
	require.NoError(t, json.Unmarshal([]byte(fixture), &envelope))

	pm := envelope.PaymentMethod
	assert.Equal(t, "5000000000000007325", pm.ID())
	assert.Equal(t, "1", pm.SyncToken())
	assert.Equal(t, "QBO", pm.CDCDomain())
	assert.Equal(t, "Active", pm.CDCStatus())

	require.NotNil(t, pm.MetaData)
	require.NotNil(t, pm.MetaData.CreateTime)

	assert.Equal(t, "Visa", pm.Name)
	assert.Equal(t, "CREDIT_CARD", pm.Type)
	require.NotNil(t, pm.Active)
	assert.True(t, *pm.Active)
}

func TestPaymentMethodRequestsMarshalWritablePayloads(t *testing.T) {
	create := PaymentMethod{
		Name: "Visa",
		Type: "CREDIT_CARD",
	}

	body, err := json.Marshal(create)
	require.NoError(t, err)
	assert.Contains(t, string(body), `"Name":"Visa"`)
	assert.Contains(t, string(body), `"Type":"CREDIT_CARD"`)
	assert.NotContains(t, string(body), `"sparse"`)
}

func TestPaymentMethodCDCExtraction(t *testing.T) {
	changedSince := time.Date(2026, time.March, 10, 12, 0, 0, 0, time.UTC)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/company/123/cdc", r.URL.Path)
		assert.Equal(t, string(EntityPaymentMethod), r.URL.Query().Get("entities"))
		assert.Equal(t, changedSince.Format(time.RFC3339), r.URL.Query().Get("changedSince"))
		w.Header().Set("Content-Type", "application/json")
		_, err := io.WriteString(w, `{"CDCResponse":[{"QueryResponse":[{"PaymentMethod":[{"Id":"1","SyncToken":"2","domain":"QBO","status":"Deleted"}],"startPosition":1,"maxResults":1}]}],"time":"2026-03-10T12:00:00Z"}`)
		require.NoError(t, err)
	}))
	defer server.Close()

	client, err := NewClient(WithHTTPClient(server.Client()), WithBaseURL(server.URL+"/v3/company"))
	require.NoError(t, err)

	pms, err := client.Company("123", oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test-token"})).PaymentMethods().ChangeDataCapture(context.Background(), changedSince)
	require.NoError(t, err)
	require.Len(t, pms, 1)
	assert.Equal(t, "1", pms[0].ID())
	assert.Equal(t, "Deleted", pms[0].CDCStatus())
}

// Compile-time interface compliance checks for PaymentMethod and PaymentMethodService.
var (
	_ Entity    = PaymentMethod{}
	_ CDCEntity = PaymentMethod{}

	_ Getter[PaymentMethod]          = (*PaymentMethodService)(nil)
	_ Querier[PaymentMethod]         = (*PaymentMethodService)(nil)
	_ Lister[PaymentMethod]          = (*PaymentMethodService)(nil)
	_ Pager[PaymentMethod]           = (*PaymentMethodService)(nil)
	_ Creator[PaymentMethod]         = (*PaymentMethodService)(nil)
	_ Updater[PaymentMethod]         = (*PaymentMethodService)(nil)
	_ SparseUpdater[PaymentMethod]   = (*PaymentMethodService)(nil)
	_ EntityCDCReader[PaymentMethod] = (*PaymentMethodService)(nil)
)

func TestPaymentMethodServiceUpdateSparseInjectsSparseFlag(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/company/123/paymentmethod", r.URL.Path)
		var payload map[string]interface{}
		require.NoError(t, json.NewDecoder(r.Body).Decode(&payload))
		assert.Equal(t, true, payload["sparse"])
		w.Header().Set("Content-Type", "application/json")
		_, err := io.WriteString(w, `{"PaymentMethod":{"Id":"1","SyncToken":"2"}}`)
		require.NoError(t, err)
	}))
	defer server.Close()

	client, err := NewClient(WithHTTPClient(server.Client()), WithBaseURL(server.URL+"/v3/company"))
	require.NoError(t, err)

	pm := PaymentMethod{}
	pm.Id = "1"
	pm.SyncTokenValue = SyncTokenValue{"1"}

	_, err = client.Company("123", oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test-token"})).PaymentMethods().UpdateSparse(context.Background(), pm)
	require.NoError(t, err)
}
