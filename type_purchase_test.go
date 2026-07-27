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

func TestPurchaseServiceGet(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/company/123/purchase/252", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_, err := io.WriteString(w, `{"Purchase":{"Id":"252","SyncToken":"0","domain":"QBO","status":"Active","AccountRef":{"value":"35","name":"Checking"},"PaymentType":"Cash","TotalAmt":10.0}}`)
		require.NoError(t, err)
	}))
	defer server.Close()

	client, err := NewClient(WithHTTPClient(server.Client()), WithBaseURL(server.URL+"/v3/company"))
	require.NoError(t, err)

	pur, err := client.Company("123", oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test-token"})).Purchases().Get(context.Background(), "252")
	require.NoError(t, err)
	assert.Equal(t, "252", pur.ID())
	assert.Equal(t, EntityPurchase, pur.EntityName())
	assert.Equal(t, "Active", pur.CDCStatus())
	assert.Equal(t, "QBO", pur.CDCDomain())
	require.NotNil(t, pur.AccountRef)
	assert.Equal(t, "35", pur.AccountRef.Value)
	assert.Equal(t, CashPurchasePaymentType, pur.PaymentType)
}

func TestPurchaseServiceQueryUsesQueryOptions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/company/123/query", r.URL.Path)
		assert.Equal(t, "SELECT * FROM Purchase STARTPOSITION 11 MAXRESULTS 10", r.URL.Query().Get("query"))
		w.Header().Set("Content-Type", "application/json")
		_, err := io.WriteString(w, `{"QueryResponse":{"Purchase":[{"Id":"252","SyncToken":"0"},{"Id":"253","SyncToken":"1"}],"startPosition":6,"maxResults":10}}`)
		require.NoError(t, err)
	}))
	defer server.Close()

	client, err := NewClient(WithHTTPClient(server.Client()), WithBaseURL(server.URL+"/v3/company"))
	require.NoError(t, err)

	purs, err := client.Company("123", oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test-token"})).Purchases().ListPage(context.Background(), 2, 10)
	require.NoError(t, err)
	require.Len(t, purs, 2)
	assert.Equal(t, "252", purs[0].ID())
	assert.Equal(t, "253", purs[1].ID())
}

func TestPurchaseJSONDecodesAccountBasedExpenseLine(t *testing.T) {
	const fixture = `{
		"Purchase": {
			"Id": "252",
			"SyncToken": "0",
			"domain": "QBO",
			"status": "Active",
			"MetaData": {
				"CreateTime": "2015-07-27T10:37:26-07:00",
				"LastUpdatedTime": "2015-07-27T10:37:26-07:00"
			},
			"AccountRef": {"value": "35", "name": "Checking"},
			"PaymentType": "Cash",
			"TxnDate": "2015-07-27",
			"DocNumber": "PUR-252",
			"PrivateNote": "internal note",
			"PrintStatus": "NotSet",
			"DepartmentRef": {"value": "1"},
			"CurrencyRef": {"value": "USD", "name": "United States Dollar"},
			"EntityRef": {"value": "33", "name": "Chin's Gas and Oil", "type": "Vendor"},
			"TotalAmt": 10.0,
			"LinkedTxn": [
				{"TxnId": "55", "TxnType": "BillPaymentCheck"}
			],
			"Line": [
				{
					"Id": "1",
					"LineNum": 1,
					"Description": "Meals and Entertainment",
					"Amount": 10.0,
					"DetailType": "AccountBasedExpenseLineDetail",
					"ProjectRef": {"value": "39298034"},
					"AccountBasedExpenseLineDetail": {
						"AccountRef": {"value": "13", "name": "Meals and Entertainment"},
						"TaxCodeRef": {"value": "NON"},
						"BillableStatus": "NotBillable",
						"CustomerRef": {"value": "5"}
					}
				}
			],
			"TxnTaxDetail": {
				"TotalTax": 0
			}
		}
	}`

	var envelope struct {
		Purchase Purchase `json:"Purchase"`
	}
	require.NoError(t, json.Unmarshal([]byte(fixture), &envelope))

	pur := envelope.Purchase
	assert.Equal(t, "252", pur.ID())
	assert.Equal(t, "0", pur.SyncToken())
	assert.Equal(t, "QBO", pur.CDCDomain())
	assert.Equal(t, "Active", pur.CDCStatus())

	require.NotNil(t, pur.MetaData)
	require.NotNil(t, pur.MetaData.CreateTime)

	require.NotNil(t, pur.AccountRef)
	assert.Equal(t, "35", pur.AccountRef.Value)
	assert.Equal(t, CashPurchasePaymentType, pur.PaymentType)

	require.NotNil(t, pur.TxnDate)
	assert.Equal(t, "2015-07-27", pur.TxnDate.String())

	assert.Equal(t, "PUR-252", pur.DocNumber)
	assert.Equal(t, "internal note", pur.PrivateNote)
	assert.Equal(t, NotSetStatus, pur.PrintStatus)

	require.NotNil(t, pur.DepartmentRef)
	assert.Equal(t, "1", pur.DepartmentRef.Value)

	require.NotNil(t, pur.CurrencyRef)
	assert.Equal(t, "USD", pur.CurrencyRef.Value)

	require.NotNil(t, pur.EntityRef)
	assert.Equal(t, "33", pur.EntityRef.Value)
	assert.Equal(t, "Vendor", pur.EntityRef.Type)

	assert.Equal(t, "10.0", pur.TotalAmt.String())

	require.Len(t, pur.LinkedTxn, 1)
	assert.Equal(t, "55", pur.LinkedTxn[0].TxnID)

	require.Len(t, pur.Line, 1)
	line0 := pur.Line[0]
	assert.Equal(t, AccountBasedExpenseLineDetailType, line0.DetailType)
	assert.Equal(t, "Meals and Entertainment", line0.Description)
	assert.Equal(t, "10.0", line0.Amount.String())
	require.NotNil(t, line0.ProjectRef)
	assert.Equal(t, "39298034", line0.ProjectRef.Value)
	require.NotNil(t, line0.AccountBasedExpenseLineDetail)
	assert.Equal(t, "13", line0.AccountBasedExpenseLineDetail.AccountRef.Value)
	assert.Equal(t, NotBillableBillableStatus, line0.AccountBasedExpenseLineDetail.BillableStatus)
	assert.Equal(t, "5", line0.AccountBasedExpenseLineDetail.CustomerRef.Value)

	require.NotNil(t, pur.TxnTaxDetail)
	assert.Equal(t, "0", pur.TxnTaxDetail.TotalTax.String())
}

func TestPurchaseJSONDecodesItemBasedExpenseLine(t *testing.T) {
	const fixture = `{
		"Purchase": {
			"Id": "300",
			"SyncToken": "0",
			"domain": "QBO",
			"status": "Active",
			"AccountRef": {"value": "42", "name": "Visa"},
			"PaymentType": "CreditCard",
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
						"TaxCodeRef": {"value": "TAX"},
						"BillableStatus": "Billable",
						"CustomerRef": {"value": "3"}
					}
				}
			]
		}
	}`

	var envelope struct {
		Purchase Purchase `json:"Purchase"`
	}
	require.NoError(t, json.Unmarshal([]byte(fixture), &envelope))

	pur := envelope.Purchase
	assert.Equal(t, "300", pur.ID())
	assert.Equal(t, CreditCardPurchasePaymentType, pur.PaymentType)

	require.Len(t, pur.Line, 1)
	line0 := pur.Line[0]
	assert.Equal(t, ItemBasedExpenseLineDetailType, line0.DetailType)
	require.NotNil(t, line0.ItemBasedExpenseLineDetail)
	assert.Equal(t, "5", line0.ItemBasedExpenseLineDetail.ItemRef.Value)
	assert.Equal(t, "2", line0.ItemBasedExpenseLineDetail.Qty.String())
	assert.Equal(t, "25", line0.ItemBasedExpenseLineDetail.UnitPrice.String())
	assert.Equal(t, BillableBillableStatus, line0.ItemBasedExpenseLineDetail.BillableStatus)
	assert.Equal(t, "3", line0.ItemBasedExpenseLineDetail.CustomerRef.Value)
}

func TestPurchaseRequestsMarshalWritablePayloads(t *testing.T) {
	create := Purchase{
		AccountRef:  &ReferenceType{Value: "35"},
		PaymentType: CheckPurchasePaymentType,
		Line: []PurchaseLine{{
			Amount:     json.Number("10"),
			DetailType: AccountBasedExpenseLineDetailType,
			AccountBasedExpenseLineDetail: &AccountBasedExpenseLineDetail{
				AccountRef: &ReferenceType{Value: "13"},
			},
		}},
	}

	body, err := json.Marshal(create)
	require.NoError(t, err)
	assert.Contains(t, string(body), `"AccountRef"`)
	assert.Contains(t, string(body), `"PaymentType":"Check"`)
	assert.Contains(t, string(body), `"AccountBasedExpenseLineDetail"`)
	assert.NotContains(t, string(body), `"sparse"`)
	assert.NotContains(t, string(body), `"TotalAmt"`)
}

func TestPurchaseCDCExtraction(t *testing.T) {
	changedSince := time.Date(2026, time.March, 10, 12, 0, 0, 0, time.UTC)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/company/123/cdc", r.URL.Path)
		assert.Equal(t, string(EntityPurchase), r.URL.Query().Get("entities"))
		assert.Equal(t, changedSince.Format(time.RFC3339), r.URL.Query().Get("changedSince"))
		w.Header().Set("Content-Type", "application/json")
		_, err := io.WriteString(w, `{"CDCResponse":[{"QueryResponse":[{"Purchase":[{"Id":"252","SyncToken":"1","domain":"QBO","status":"Deleted"}],"startPosition":1,"maxResults":1}]}],"time":"2026-03-10T12:00:00Z"}`)
		require.NoError(t, err)
	}))
	defer server.Close()

	client, err := NewClient(WithHTTPClient(server.Client()), WithBaseURL(server.URL+"/v3/company"))
	require.NoError(t, err)

	purs, err := client.Company("123", oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test-token"})).Purchases().ChangeDataCapture(context.Background(), changedSince)
	require.NoError(t, err)
	require.Len(t, purs, 1)
	assert.Equal(t, "252", purs[0].ID())
	assert.Equal(t, "Deleted", purs[0].CDCStatus())
}

// Compile-time interface compliance checks for Purchase and PurchaseService.
var (
	// Purchase entity type compliance.
	_ Entity    = Purchase{}
	_ CDCEntity = Purchase{}

	// PurchaseService operation compliance.
	_ Getter[Purchase]          = (*PurchaseService)(nil)
	_ Querier[Purchase]         = (*PurchaseService)(nil)
	_ Lister[Purchase]          = (*PurchaseService)(nil)
	_ Pager[Purchase]           = (*PurchaseService)(nil)
	_ Creator[Purchase]         = (*PurchaseService)(nil)
	_ Updater[Purchase]         = (*PurchaseService)(nil)
	_ SparseUpdater[Purchase]   = (*PurchaseService)(nil)
	_ Deleter[Purchase]         = (*PurchaseService)(nil)
	_ EntityCDCReader[Purchase] = (*PurchaseService)(nil)
)

func TestPurchaseServiceUpdateSparseInjectsSparseFlag(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/company/123/purchase", r.URL.Path)
		var payload map[string]interface{}
		require.NoError(t, json.NewDecoder(r.Body).Decode(&payload))
		assert.Equal(t, true, payload["sparse"])
		w.Header().Set("Content-Type", "application/json")
		_, err := io.WriteString(w, `{"Purchase":{"Id":"1","SyncToken":"2"}}`)
		require.NoError(t, err)
	}))
	defer server.Close()

	client, err := NewClient(WithHTTPClient(server.Client()), WithBaseURL(server.URL+"/v3/company"))
	require.NoError(t, err)

	purchase := Purchase{}
	purchase.Id = "1"
	purchase.SyncTokenValue = SyncTokenValue{"1"}

	_, err = client.Company("123", oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test-token"})).Purchases().UpdateSparse(context.Background(), purchase)
	require.NoError(t, err)
}
