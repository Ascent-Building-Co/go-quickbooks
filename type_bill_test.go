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

func TestBillServiceGet(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/company/123/bill/25", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_, err := io.WriteString(w, `{"Bill":{"Id":"25","SyncToken":"2","domain":"QBO","status":"Active","VendorRef":{"value":"46","name":"Norton Lumber and Building Materials"}}}`)
		require.NoError(t, err)
	}))
	defer server.Close()

	client, err := NewClient(WithHTTPClient(server.Client()), WithBaseURL(server.URL+"/v3/company"))
	require.NoError(t, err)

	bill, err := client.Company("123", oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test-token"})).Bills().Get(context.Background(), "25")
	require.NoError(t, err)
	assert.Equal(t, "25", bill.ID())
	assert.Equal(t, EntityBill, bill.EntityName())
	assert.Equal(t, "Active", bill.CDCStatus())
	assert.Equal(t, "QBO", bill.CDCDomain())
	require.NotNil(t, bill.VendorRef)
	assert.Equal(t, "46", bill.VendorRef.Value)
}

func TestBillServiceQueryUsesQueryOptions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/company/123/query", r.URL.Path)
		assert.Equal(t, "SELECT * FROM Bill STARTPOSITION 11 MAXRESULTS 10", r.URL.Query().Get("query"))
		w.Header().Set("Content-Type", "application/json")
		_, err := io.WriteString(w, `{"QueryResponse":{"Bill":[{"Id":"25","SyncToken":"2"},{"Id":"26","SyncToken":"0"}],"startPosition":6,"maxResults":10}}`)
		require.NoError(t, err)
	}))
	defer server.Close()

	client, err := NewClient(WithHTTPClient(server.Client()), WithBaseURL(server.URL+"/v3/company"))
	require.NoError(t, err)

	bills, err := client.Company("123", oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test-token"})).Bills().ListPage(context.Background(), 2, 10)
	require.NoError(t, err)
	require.Len(t, bills, 2)
	assert.Equal(t, "25", bills[0].ID())
	assert.Equal(t, "26", bills[1].ID())
}

func TestBillJSONDecodesV75Fields(t *testing.T) {
	const fixture = `{
		"Bill": {
			"Id": "25",
			"SyncToken": "2",
			"domain": "QBO",
			"status": "Active",
			"MetaData": {
				"CreateTime": "2014-11-06T15:37:25-08:00",
				"LastUpdatedTime": "2015-02-09T10:11:11-08:00"
			},
			"VendorRef": {"value": "46", "name": "Norton Lumber and Building Materials"},
			"APAccountRef": {"value": "33", "name": "Accounts Payable (A/P)"},
			"CurrencyRef": {"value": "USD", "name": "United States Dollar"},
			"TxnDate": "2014-11-06",
			"DueDate": "2014-12-06",
			"DocNumber": "BILL-25",
			"PrivateNote": "internal note",
			"SalesTermRef": {"value": "3"},
			"DepartmentRef": {"value": "1", "name": "Garden Services"},
			"TotalAmt": 103.55,
			"Balance": 0,
			"LinkedTxn": [
				{"TxnId": "118", "TxnType": "BillPaymentCheck"}
			],
			"Line": [
				{
					"Id": "1",
					"LineNum": 1,
					"Description": "Lumber",
					"Amount": 103.55,
					"DetailType": "AccountBasedExpenseLineDetail",
					"ProjectRef": {"value": "39298034"},
					"AccountBasedExpenseLineDetail": {
						"AccountRef": {"value": "64", "name": "Job Expenses:Job Materials:Decks and Patios"},
						"BillableStatus": "Billable",
						"CustomerRef": {"value": "26", "name": "Travis Waldron"},
						"TaxCodeRef": {"value": "TAX"}
					}
				},
				{
					"Id": "2",
					"LineNum": 2,
					"Description": "Hardware",
					"Amount": 50.00,
					"DetailType": "ItemBasedExpenseLineDetail",
					"ItemBasedExpenseLineDetail": {
						"ItemRef": {"value": "5", "name": "Lumber"},
						"Qty": 2,
						"UnitPrice": 25,
						"BillableStatus": "NotBillable",
						"TaxCodeRef": {"value": "TAX"},
						"ClassRef": {"value": "3"}
					}
				}
			],
			"TxnTaxDetail": {
				"TxnTaxCodeRef": {"value": "8"},
				"TotalTax": 5.00,
				"TaxLine": [
					{
						"Amount": 5.00,
						"DetailType": "TaxLineDetail",
						"TaxLineDetail": {
							"TaxRateRef": {"value": "1"},
							"NetAmountTaxable": 100.00
						}
					}
				]
			}
		}
	}`

	var envelope struct {
		Bill Bill `json:"Bill"`
	}
	require.NoError(t, json.Unmarshal([]byte(fixture), &envelope))

	b := envelope.Bill
	assert.Equal(t, "25", b.ID())
	assert.Equal(t, "2", b.SyncToken())
	assert.Equal(t, "QBO", b.CDCDomain())
	assert.Equal(t, "Active", b.CDCStatus())

	require.NotNil(t, b.MetaData)
	require.NotNil(t, b.MetaData.CreateTime)

	require.NotNil(t, b.VendorRef)
	assert.Equal(t, "46", b.VendorRef.Value)

	require.NotNil(t, b.APAccountRef)
	assert.Equal(t, "33", b.APAccountRef.Value)

	require.NotNil(t, b.CurrencyRef)
	assert.Equal(t, "USD", b.CurrencyRef.Value)

	require.NotNil(t, b.TxnDate)
	assert.Equal(t, "2014-11-06", b.TxnDate.String())

	require.NotNil(t, b.DueDate)
	assert.Equal(t, "2014-12-06", b.DueDate.String())

	assert.Equal(t, "BILL-25", b.DocNumber)
	assert.Equal(t, "internal note", b.PrivateNote)

	require.NotNil(t, b.SalesTermRef)
	assert.Equal(t, "3", b.SalesTermRef.Value)

	require.NotNil(t, b.DepartmentRef)
	assert.Equal(t, "1", b.DepartmentRef.Value)

	assert.Equal(t, "103.55", b.TotalAmt.String())
	assert.Equal(t, "0", b.Balance.String())

	require.Len(t, b.LinkedTxn, 1)
	assert.Equal(t, "118", b.LinkedTxn[0].TxnID)
	assert.Equal(t, "BillPaymentCheck", b.LinkedTxn[0].TxnType)

	require.Len(t, b.Line, 2)

	// Account-based line
	line0 := b.Line[0]
	assert.Equal(t, AccountBasedExpenseLineDetailType, line0.DetailType)
	assert.Equal(t, "Lumber", line0.Description)
	assert.Equal(t, "103.55", line0.Amount.String())
	require.NotNil(t, line0.ProjectRef)
	assert.Equal(t, "39298034", line0.ProjectRef.Value)
	require.NotNil(t, line0.AccountBasedExpenseLineDetail)
	assert.Equal(t, "64", line0.AccountBasedExpenseLineDetail.AccountRef.Value)
	assert.Equal(t, BillableBillableStatus, line0.AccountBasedExpenseLineDetail.BillableStatus)
	assert.Equal(t, "26", line0.AccountBasedExpenseLineDetail.CustomerRef.Value)

	// Item-based line
	line1 := b.Line[1]
	assert.Equal(t, ItemBasedExpenseLineDetailType, line1.DetailType)
	require.NotNil(t, line1.ItemBasedExpenseLineDetail)
	assert.Equal(t, "5", line1.ItemBasedExpenseLineDetail.ItemRef.Value)
	assert.Equal(t, "2", line1.ItemBasedExpenseLineDetail.Qty.String())
	assert.Equal(t, "25", line1.ItemBasedExpenseLineDetail.UnitPrice.String())
	assert.Equal(t, NotBillableBillableStatus, line1.ItemBasedExpenseLineDetail.BillableStatus)
	assert.Equal(t, "3", line1.ItemBasedExpenseLineDetail.ClassRef.Value)

	// Tax detail
	require.NotNil(t, b.TxnTaxDetail)
	assert.Equal(t, "8", b.TxnTaxDetail.TxnTaxCodeRef.Value)
	assert.Equal(t, "5.00", b.TxnTaxDetail.TotalTax.String())
	require.Len(t, b.TxnTaxDetail.TaxLine, 1)
	require.NotNil(t, b.TxnTaxDetail.TaxLine[0].TaxLineDetail)
	assert.Equal(t, "100.00", b.TxnTaxDetail.TaxLine[0].TaxLineDetail.NetAmountTaxable.String())
}

func TestBillRequestsMarshalWritablePayloads(t *testing.T) {
	create := Bill{
		VendorRef: &ReferenceType{Value: "46"},
		Line: []BillLine{{
			Amount:     json.Number("200"),
			DetailType: AccountBasedExpenseLineDetailType,
			AccountBasedExpenseLineDetail: &AccountBasedExpenseLineDetail{
				AccountRef: &ReferenceType{Value: "7"},
			},
		}},
	}

	body, err := json.Marshal(create)
	require.NoError(t, err)
	assert.Contains(t, string(body), `"VendorRef"`)
	assert.Contains(t, string(body), `"AccountBasedExpenseLineDetail"`)
	assert.NotContains(t, string(body), `"sparse"`)
	assert.NotContains(t, string(body), `"Balance"`)
}

func TestBillCDCExtraction(t *testing.T) {
	changedSince := time.Date(2026, time.March, 10, 12, 0, 0, 0, time.UTC)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/company/123/cdc", r.URL.Path)
		assert.Equal(t, string(EntityBill), r.URL.Query().Get("entities"))
		assert.Equal(t, changedSince.Format(time.RFC3339), r.URL.Query().Get("changedSince"))
		w.Header().Set("Content-Type", "application/json")
		_, err := io.WriteString(w, `{"CDCResponse":[{"QueryResponse":[{"Bill":[{"Id":"25","SyncToken":"3","domain":"QBO","status":"Deleted"}],"startPosition":1,"maxResults":1}]}],"time":"2026-03-10T12:00:00Z"}`)
		require.NoError(t, err)
	}))
	defer server.Close()

	client, err := NewClient(WithHTTPClient(server.Client()), WithBaseURL(server.URL+"/v3/company"))
	require.NoError(t, err)

	bills, err := client.Company("123", oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test-token"})).Bills().ChangeDataCapture(context.Background(), changedSince)
	require.NoError(t, err)
	require.Len(t, bills, 1)
	assert.Equal(t, "25", bills[0].ID())
	assert.Equal(t, "Deleted", bills[0].CDCStatus())
}

// Compile-time interface compliance checks for Bill and BillService.
var (
	// Bill entity type compliance.
	_ Entity    = Bill{}
	_ CDCEntity = Bill{}

	// BillService operation compliance.
	_ Getter[Bill]          = (*BillService)(nil)
	_ Querier[Bill]         = (*BillService)(nil)
	_ Lister[Bill]          = (*BillService)(nil)
	_ Pager[Bill]           = (*BillService)(nil)
	_ Creator[Bill]         = (*BillService)(nil)
	_ Updater[Bill]         = (*BillService)(nil)
	_ SparseUpdater[Bill]   = (*BillService)(nil)
	_ Deleter[Bill]         = (*BillService)(nil)
	_ EntityCDCReader[Bill] = (*BillService)(nil)
)

func TestBillServiceUpdateSparseInjectsSparseFlag(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/company/123/bill", r.URL.Path)
		var payload map[string]interface{}
		require.NoError(t, json.NewDecoder(r.Body).Decode(&payload))
		assert.Equal(t, true, payload["sparse"])
		w.Header().Set("Content-Type", "application/json")
		_, err := io.WriteString(w, `{"Bill":{"Id":"1","SyncToken":"2"}}`)
		require.NoError(t, err)
	}))
	defer server.Close()

	client, err := NewClient(WithHTTPClient(server.Client()), WithBaseURL(server.URL+"/v3/company"))
	require.NoError(t, err)

	bill := Bill{}
	bill.Id = "1"
	bill.SyncTokenValue = SyncTokenValue{"1"}

	_, err = client.Company("123", oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test-token"})).Bills().UpdateSparse(context.Background(), bill)
	require.NoError(t, err)
}
