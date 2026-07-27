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

func TestVendorCreditServiceGet(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/company/123/vendorcredit/255", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_, err := io.WriteString(w, `{"VendorCredit":{"Id":"255","SyncToken":"0","domain":"QBO","status":"Active","VendorRef":{"value":"30","name":"Books by Bessie"},"TotalAmt":90.0}}`)
		require.NoError(t, err)
	}))
	defer server.Close()

	client, err := NewClient(WithHTTPClient(server.Client()), WithBaseURL(server.URL+"/v3/company"))
	require.NoError(t, err)

	vc, err := client.Company("123", oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test-token"})).VendorCredits().Get(context.Background(), "255")
	require.NoError(t, err)
	assert.Equal(t, "255", vc.ID())
	assert.Equal(t, EntityVendorCredit, vc.EntityName())
	assert.Equal(t, "Active", vc.CDCStatus())
	assert.Equal(t, "QBO", vc.CDCDomain())
	require.NotNil(t, vc.VendorRef)
	assert.Equal(t, "30", vc.VendorRef.Value)
}

func TestVendorCreditServiceQueryUsesQueryOptions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/company/123/query", r.URL.Path)
		assert.Equal(t, "SELECT * FROM VendorCredit STARTPOSITION 11 MAXRESULTS 10", r.URL.Query().Get("query"))
		w.Header().Set("Content-Type", "application/json")
		_, err := io.WriteString(w, `{"QueryResponse":{"VendorCredit":[{"Id":"255","SyncToken":"0"},{"Id":"256","SyncToken":"1"}],"startPosition":6,"maxResults":10}}`)
		require.NoError(t, err)
	}))
	defer server.Close()

	client, err := NewClient(WithHTTPClient(server.Client()), WithBaseURL(server.URL+"/v3/company"))
	require.NoError(t, err)

	vcs, err := client.Company("123", oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test-token"})).VendorCredits().ListPage(context.Background(), 2, 10)
	require.NoError(t, err)
	require.Len(t, vcs, 2)
	assert.Equal(t, "255", vcs[0].ID())
	assert.Equal(t, "256", vcs[1].ID())
}

func TestVendorCreditJSONDecodesAccountBasedLine(t *testing.T) {
	const fixture = `{
		"VendorCredit": {
			"Id": "255",
			"SyncToken": "0",
			"domain": "QBO",
			"status": "Active",
			"MetaData": {
				"CreateTime": "2015-07-28T14:13:30-07:00",
				"LastUpdatedTime": "2015-07-28T14:13:30-07:00"
			},
			"VendorRef": {"value": "30", "name": "Books by Bessie"},
			"APAccountRef": {"value": "33", "name": "Accounts Payable (A/P)"},
			"CurrencyRef": {"value": "USD", "name": "United States Dollar"},
			"TxnDate": "2014-12-23",
			"DocNumber": "VC-255",
			"PrivateNote": "internal note",
			"DepartmentRef": {"value": "1"},
			"TotalAmt": 90.0,
			"Balance": 90.0,
			"LinkedTxn": [
				{"TxnId": "100", "TxnType": "Bill"}
			],
			"Line": [
				{
					"Id": "1",
					"LineNum": 1,
					"Description": "Bank Charges refund",
					"Amount": 90.0,
					"DetailType": "AccountBasedExpenseLineDetail",
					"ProjectRef": {"value": "39298045"},
					"AccountBasedExpenseLineDetail": {
						"AccountRef": {"value": "8", "name": "Bank Charges"},
						"TaxCodeRef": {"value": "TAX"},
						"BillableStatus": "Billable",
						"CustomerRef": {"value": "1", "name": "Amy's Bird Sanctuary"}
					}
				}
			]
		}
	}`

	var envelope struct {
		VendorCredit VendorCredit `json:"VendorCredit"`
	}
	require.NoError(t, json.Unmarshal([]byte(fixture), &envelope))

	vc := envelope.VendorCredit
	assert.Equal(t, "255", vc.ID())
	assert.Equal(t, "0", vc.SyncToken())
	assert.Equal(t, "QBO", vc.CDCDomain())
	assert.Equal(t, "Active", vc.CDCStatus())

	require.NotNil(t, vc.MetaData)
	require.NotNil(t, vc.MetaData.CreateTime)

	require.NotNil(t, vc.VendorRef)
	assert.Equal(t, "30", vc.VendorRef.Value)

	require.NotNil(t, vc.APAccountRef)
	assert.Equal(t, "33", vc.APAccountRef.Value)

	require.NotNil(t, vc.CurrencyRef)
	assert.Equal(t, "USD", vc.CurrencyRef.Value)

	require.NotNil(t, vc.TxnDate)
	assert.Equal(t, "2014-12-23", vc.TxnDate.String())

	assert.Equal(t, "VC-255", vc.DocNumber)
	assert.Equal(t, "internal note", vc.PrivateNote)

	require.NotNil(t, vc.DepartmentRef)
	assert.Equal(t, "1", vc.DepartmentRef.Value)

	assert.Equal(t, "90.0", vc.TotalAmt.String())
	assert.Equal(t, "90.0", vc.Balance.String())

	require.Len(t, vc.LinkedTxn, 1)
	assert.Equal(t, "100", vc.LinkedTxn[0].TxnID)
	assert.Equal(t, "Bill", vc.LinkedTxn[0].TxnType)

	require.Len(t, vc.Line, 1)
	line0 := vc.Line[0]
	assert.Equal(t, AccountBasedExpenseLineDetailType, line0.DetailType)
	assert.Equal(t, "Bank Charges refund", line0.Description)
	assert.Equal(t, "90.0", line0.Amount.String())
	require.NotNil(t, line0.ProjectRef)
	assert.Equal(t, "39298045", line0.ProjectRef.Value)
	require.NotNil(t, line0.AccountBasedExpenseLineDetail)
	assert.Equal(t, "8", line0.AccountBasedExpenseLineDetail.AccountRef.Value)
	assert.Equal(t, BillableBillableStatus, line0.AccountBasedExpenseLineDetail.BillableStatus)
	assert.Equal(t, "1", line0.AccountBasedExpenseLineDetail.CustomerRef.Value)
}

func TestVendorCreditJSONDecodesItemBasedLine(t *testing.T) {
	const fixture = `{
		"VendorCredit": {
			"Id": "256",
			"SyncToken": "0",
			"domain": "QBO",
			"status": "Active",
			"VendorRef": {"value": "30"},
			"TotalAmt": 50.0,
			"Line": [
				{
					"Id": "1",
					"Amount": 50.0,
					"DetailType": "ItemBasedExpenseLineDetail",
					"ItemBasedExpenseLineDetail": {
						"ItemRef": {"value": "5", "name": "Concrete"},
						"Qty": 2,
						"UnitPrice": 25,
						"BillableStatus": "NotBillable"
					}
				}
			]
		}
	}`

	var envelope struct {
		VendorCredit VendorCredit `json:"VendorCredit"`
	}
	require.NoError(t, json.Unmarshal([]byte(fixture), &envelope))

	vc := envelope.VendorCredit
	require.Len(t, vc.Line, 1)
	line0 := vc.Line[0]
	assert.Equal(t, ItemBasedExpenseLineDetailType, line0.DetailType)
	require.NotNil(t, line0.ItemBasedExpenseLineDetail)
	assert.Equal(t, "5", line0.ItemBasedExpenseLineDetail.ItemRef.Value)
	assert.Equal(t, "2", line0.ItemBasedExpenseLineDetail.Qty.String())
	assert.Equal(t, "25", line0.ItemBasedExpenseLineDetail.UnitPrice.String())
	assert.Equal(t, NotBillableBillableStatus, line0.ItemBasedExpenseLineDetail.BillableStatus)
}

func TestVendorCreditRequestsMarshalWritablePayloads(t *testing.T) {
	create := VendorCredit{
		VendorRef: &ReferenceType{Value: "30"},
		Line: []VendorCreditLine{{
			Amount:     json.Number("90"),
			DetailType: AccountBasedExpenseLineDetailType,
			AccountBasedExpenseLineDetail: &AccountBasedExpenseLineDetail{
				AccountRef: &ReferenceType{Value: "8"},
			},
		}},
	}

	body, err := json.Marshal(create)
	require.NoError(t, err)
	assert.Contains(t, string(body), `"VendorRef"`)
	assert.Contains(t, string(body), `"AccountBasedExpenseLineDetail"`)
	assert.NotContains(t, string(body), `"sparse"`)
	assert.NotContains(t, string(body), `"TotalAmt"`)
	assert.NotContains(t, string(body), `"RecurDataRef"`)
}

func TestVendorCreditCDCExtraction(t *testing.T) {
	changedSince := time.Date(2026, time.March, 10, 12, 0, 0, 0, time.UTC)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/company/123/cdc", r.URL.Path)
		assert.Equal(t, string(EntityVendorCredit), r.URL.Query().Get("entities"))
		assert.Equal(t, changedSince.Format(time.RFC3339), r.URL.Query().Get("changedSince"))
		w.Header().Set("Content-Type", "application/json")
		_, err := io.WriteString(w, `{"CDCResponse":[{"QueryResponse":[{"VendorCredit":[{"Id":"255","SyncToken":"1","domain":"QBO","status":"Deleted"}],"startPosition":1,"maxResults":1}]}],"time":"2026-03-10T12:00:00Z"}`)
		require.NoError(t, err)
	}))
	defer server.Close()

	client, err := NewClient(WithHTTPClient(server.Client()), WithBaseURL(server.URL+"/v3/company"))
	require.NoError(t, err)

	vcs, err := client.Company("123", oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test-token"})).VendorCredits().ChangeDataCapture(context.Background(), changedSince)
	require.NoError(t, err)
	require.Len(t, vcs, 1)
	assert.Equal(t, "255", vcs[0].ID())
	assert.Equal(t, "Deleted", vcs[0].CDCStatus())
}

// Compile-time interface compliance checks for VendorCredit and VendorCreditService.
var (
	_ Entity    = VendorCredit{}
	_ CDCEntity = VendorCredit{}

	_ Getter[VendorCredit]          = (*VendorCreditService)(nil)
	_ Querier[VendorCredit]         = (*VendorCreditService)(nil)
	_ Lister[VendorCredit]          = (*VendorCreditService)(nil)
	_ Pager[VendorCredit]           = (*VendorCreditService)(nil)
	_ Creator[VendorCredit]         = (*VendorCreditService)(nil)
	_ Updater[VendorCredit]         = (*VendorCreditService)(nil)
	_ SparseUpdater[VendorCredit]   = (*VendorCreditService)(nil)
	_ Deleter[VendorCredit]         = (*VendorCreditService)(nil)
	_ EntityCDCReader[VendorCredit] = (*VendorCreditService)(nil)
)

func TestVendorCreditServiceUpdateSparseInjectsSparseFlag(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/company/123/vendorcredit", r.URL.Path)
		var payload map[string]interface{}
		require.NoError(t, json.NewDecoder(r.Body).Decode(&payload))
		assert.Equal(t, true, payload["sparse"])
		w.Header().Set("Content-Type", "application/json")
		_, err := io.WriteString(w, `{"VendorCredit":{"Id":"1","SyncToken":"2"}}`)
		require.NoError(t, err)
	}))
	defer server.Close()

	client, err := NewClient(WithHTTPClient(server.Client()), WithBaseURL(server.URL+"/v3/company"))
	require.NoError(t, err)

	vc := VendorCredit{}
	vc.Id = "1"
	vc.SyncTokenValue = SyncTokenValue{"1"}

	_, err = client.Company("123", oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test-token"})).VendorCredits().UpdateSparse(context.Background(), vc)
	require.NoError(t, err)
}
