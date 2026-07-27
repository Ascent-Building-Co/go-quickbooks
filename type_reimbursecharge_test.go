package quickbooks

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
)

func TestReimburseChargeServiceGet(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/company/123/reimbursecharge/42", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_, err := io.WriteString(w, `{"ReimburseCharge":{"Id":"42","SyncToken":"0","CustomerRef":{"value":"20","name":"ACME Corp"},"Amount":"150.00","HasBeenInvoiced":false,"Line":[{"DetailType":"ReimburseLineDetail","ReimburseLineDetail":{"ItemRef":{"value":"5","name":"Services"}}}]}}`)
		require.NoError(t, err)
	}))
	defer server.Close()

	client, err := NewClient(WithHTTPClient(server.Client()), WithBaseURL(server.URL+"/v3/company"))
	require.NoError(t, err)

	rc, err := client.Company("123", oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test-token"})).ReimburseCharges().Get(context.Background(), "42")
	require.NoError(t, err)
	assert.Equal(t, "42", rc.ID())
	assert.Equal(t, EntityReimburseCharge, rc.EntityName())
	assert.Equal(t, "150.00", rc.Amount.String())
	require.NotNil(t, rc.CustomerRef)
	assert.Equal(t, "20", rc.CustomerRef.Value)
	assert.Equal(t, "ACME Corp", rc.CustomerRef.Name)
	require.NotNil(t, rc.HasBeenInvoiced)
	assert.False(t, *rc.HasBeenInvoiced)
	require.Len(t, rc.Line, 1)
	assert.Equal(t, ReimburseLineDetailType, rc.Line[0].DetailType)
	require.NotNil(t, rc.Line[0].ReimburseLineDetail)
	require.NotNil(t, rc.Line[0].ReimburseLineDetail.ItemRef)
	assert.Equal(t, "5", rc.Line[0].ReimburseLineDetail.ItemRef.Value)
}

func TestReimburseChargeServiceListPage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/company/123/query", r.URL.Path)
		assert.Equal(t, "SELECT * FROM ReimburseCharge STARTPOSITION 1 MAXRESULTS 5", r.URL.Query().Get("query"))
		w.Header().Set("Content-Type", "application/json")
		_, err := io.WriteString(w, `{"QueryResponse":{"ReimburseCharge":[{"Id":"42","SyncToken":"0","CustomerRef":{"value":"20"},"Amount":"150.00"},{"Id":"43","SyncToken":"0","CustomerRef":{"value":"21"},"Amount":"75.00"}],"startPosition":1,"maxResults":2}}`)
		require.NoError(t, err)
	}))
	defer server.Close()

	client, err := NewClient(WithHTTPClient(server.Client()), WithBaseURL(server.URL+"/v3/company"))
	require.NoError(t, err)

	rcs, err := client.Company("123", oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test-token"})).ReimburseCharges().ListPage(context.Background(), 1, 5)
	require.NoError(t, err)
	require.Len(t, rcs, 2)
	assert.Equal(t, "42", rcs[0].ID())
	assert.Equal(t, "150.00", rcs[0].Amount.String())
	assert.Equal(t, "43", rcs[1].ID())
	assert.Equal(t, "75.00", rcs[1].Amount.String())
}

func TestReimburseChargeJSONDecodesV75Fields(t *testing.T) {
	const fixture = `{
		"ReimburseCharge": {
			"Id": "42",
			"SyncToken": "0",
			"MetaData": {
				"CreateTime": "2026-01-15T10:00:00-07:00",
				"LastUpdatedTime": "2026-01-15T10:00:00-07:00"
			},
			"CustomerRef": {"value": "20", "name": "ACME Corp"},
			"CurrencyRef": {"value": "USD", "name": "United States Dollar"},
			"Amount": "250.00",
			"ExchangeRate": "1.0",
			"PrivateNote": "Reimbursable expense from client visit",
			"HasBeenInvoiced": false,
			"HomeTotalAmt": "250.00",
			"LinkedTxn": [{"TxnId": "99", "TxnType": "Invoice"}],
			"Line": [
				{
					"DetailType": "ReimburseLineDetail",
					"LineNum": "1",
					"Id": "1",
					"ReimburseLineDetail": {
						"ItemRef": {"value": "5", "name": "Services"},
						"ClassRef": {"value": "3", "name": "Engineering"},
						"TaxCodeRef": {"value": "NON"},
						"Qty": 2,
						"UnitPrice": 125.00
					},
					"LinkedTxn": [{"TxnId": "99", "TxnType": "Invoice"}]
				}
			]
		}
	}`

	var envelope struct {
		ReimburseCharge ReimburseCharge `json:"ReimburseCharge"`
	}
	require.NoError(t, json.Unmarshal([]byte(fixture), &envelope))

	rc := envelope.ReimburseCharge
	assert.Equal(t, "42", rc.ID())
	assert.Equal(t, "0", rc.SyncToken())
	require.NotNil(t, rc.MetaData)

	require.NotNil(t, rc.CustomerRef)
	assert.Equal(t, "20", rc.CustomerRef.Value)
	require.NotNil(t, rc.CurrencyRef)
	assert.Equal(t, "USD", rc.CurrencyRef.Value)

	assert.Equal(t, "250.00", rc.Amount.String())
	assert.Equal(t, "1.0", rc.ExchangeRate.String())
	assert.Equal(t, "Reimbursable expense from client visit", rc.PrivateNote)
	assert.Equal(t, "250.00", rc.HomeTotalAmt.String())

	require.NotNil(t, rc.HasBeenInvoiced)
	assert.False(t, *rc.HasBeenInvoiced)

	require.Len(t, rc.LinkedTxn, 1)
	assert.Equal(t, "99", rc.LinkedTxn[0].TxnID)

	require.Len(t, rc.Line, 1)
	line := rc.Line[0]
	assert.Equal(t, ReimburseLineDetailType, line.DetailType)
	assert.Equal(t, "1", line.ID())
	require.NotNil(t, line.ReimburseLineDetail)
	require.NotNil(t, line.ReimburseLineDetail.ItemRef)
	assert.Equal(t, "5", line.ReimburseLineDetail.ItemRef.Value)
	require.NotNil(t, line.ReimburseLineDetail.ClassRef)
	assert.Equal(t, "3", line.ReimburseLineDetail.ClassRef.Value)
	assert.Equal(t, "2", line.ReimburseLineDetail.Qty.String())
	assert.Equal(t, "125.00", line.ReimburseLineDetail.UnitPrice.String())
	require.Len(t, line.LinkedTxn, 1)
	assert.Equal(t, "99", line.LinkedTxn[0].TxnID)
}

// Compile-time interface compliance checks for ReimburseCharge and ReimburseChargeService.
// ReimburseCharge supports READ and QUERY only — no mutations.
var (
	_ Entity    = ReimburseCharge{}
	_ CDCEntity = ReimburseCharge{}

	_ Getter[ReimburseCharge]          = (*ReimburseChargeService)(nil)
	_ Querier[ReimburseCharge]         = (*ReimburseChargeService)(nil)
	_ Lister[ReimburseCharge]          = (*ReimburseChargeService)(nil)
	_ Pager[ReimburseCharge]           = (*ReimburseChargeService)(nil)
	_ EntityCDCReader[ReimburseCharge] = (*ReimburseChargeService)(nil)
)
