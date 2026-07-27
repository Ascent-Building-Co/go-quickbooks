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

func TestPurchaseOrderServiceGet(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/company/123/purchaseorder/259", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_, err := io.WriteString(w, `{"PurchaseOrder":{"Id":"259","SyncToken":"0","domain":"QBO","status":"Active","VendorRef":{"value":"41","name":"PO Vendor"},"APAccountRef":{"value":"33","name":"Accounts Payable"},"TotalAmt":25.00}}`)
		require.NoError(t, err)
	}))
	defer server.Close()

	client, err := NewClient(WithHTTPClient(server.Client()), WithBaseURL(server.URL+"/v3/company"))
	require.NoError(t, err)

	po, err := client.Company("123", oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test-token"})).PurchaseOrders().Get(context.Background(), "259")
	require.NoError(t, err)
	assert.Equal(t, "259", po.ID())
	assert.Equal(t, EntityPurchaseOrder, po.EntityName())
	assert.Equal(t, "Active", po.CDCStatus())
	assert.Equal(t, "QBO", po.CDCDomain())
	require.NotNil(t, po.VendorRef)
	assert.Equal(t, "41", po.VendorRef.Value)
	require.NotNil(t, po.APAccountRef)
	assert.Equal(t, "33", po.APAccountRef.Value)
	assert.Equal(t, "25.00", po.TotalAmt.String())
}

func TestPurchaseOrderServiceListPage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/company/123/query", r.URL.Path)
		assert.Equal(t, "SELECT * FROM PurchaseOrder STARTPOSITION 1 MAXRESULTS 5", r.URL.Query().Get("query"))
		w.Header().Set("Content-Type", "application/json")
		_, err := io.WriteString(w, `{"QueryResponse":{"PurchaseOrder":[{"Id":"259","SyncToken":"0","VendorRef":{"value":"41"},"TotalAmt":25.00},{"Id":"260","SyncToken":"0","VendorRef":{"value":"42"},"TotalAmt":100.00}],"startPosition":1,"maxResults":2}}`)
		require.NoError(t, err)
	}))
	defer server.Close()

	client, err := NewClient(WithHTTPClient(server.Client()), WithBaseURL(server.URL+"/v3/company"))
	require.NoError(t, err)

	pos, err := client.Company("123", oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test-token"})).PurchaseOrders().ListPage(context.Background(), 1, 5)
	require.NoError(t, err)
	require.Len(t, pos, 2)
	assert.Equal(t, "259", pos[0].ID())
	assert.Equal(t, "260", pos[1].ID())
}

func TestPurchaseOrderJSONDecodesV75Fields(t *testing.T) {
	const fixture = `{
		"PurchaseOrder": {
			"Id": "259",
			"SyncToken": "1",
			"domain": "QBO",
			"status": "Active",
			"MetaData": {
				"CreateTime": "2015-07-28T14:00:00-07:00",
				"LastUpdatedTime": "2015-07-28T14:00:00-07:00"
			},
			"CustomField": [
				{
					"DefinitionId": "1",
					"Name": "Crew #",
					"Type": "StringType",
					"StringValue": "102"
				}
			],
			"DocNumber": "1005",
			"TxnDate": "2015-07-28",
			"DueDate": "2015-08-27",
			"VendorRef": {"value": "41", "name": "PO Vendor"},
			"APAccountRef": {"value": "33", "name": "Accounts Payable"},
			"TotalAmt": 25.00,
			"POEmail": {"Address": "vendor@example.com"},
			"POStatus": "Open",
			"Memo": "Please ship ASAP",
			"ShipTo": {"value": "10", "name": "Customer HQ"},
			"ShipMethodRef": {"value": "FedEx", "name": "FedEx"},
			"ShipAddr": {"Line1": "100 Main St", "City": "Denver", "PostalCode": "80202"},
			"VendorAddr": {"Line1": "200 Oak St", "City": "Boulder"},
			"EmailStatus": "NotSet",
			"ExchangeRate": 1.0,
			"Line": [
				{
					"Id": "1",
					"LineNum": 1,
					"Description": "Widget A",
					"Amount": 25.00,
					"DetailType": "ItemBasedExpenseLineDetail",
					"ItemBasedExpenseLineDetail": {
						"ItemRef": {"value": "18", "name": "Widget A"},
						"CustomerRef": {"value": "10", "name": "Cool Cars"},
						"Qty": 5,
						"UnitPrice": 5,
						"TaxCodeRef": {"value": "NON"},
						"BillableStatus": "NotBillable"
					},
					"ProjectRef": {"value": "39298034"}
				}
			],
			"LinkedTxn": [
				{"TxnId": "310", "TxnType": "Bill"}
			],
			"TxnTaxDetail": {
				"TotalTax": 0
			},
			"SalesTermRef": {"value": "3", "name": "Net 30"},
			"RecurDataRef": {"value": "99"}
		}
	}`

	var envelope struct {
		PurchaseOrder PurchaseOrder `json:"PurchaseOrder"`
	}
	require.NoError(t, json.Unmarshal([]byte(fixture), &envelope))

	po := envelope.PurchaseOrder
	assert.Equal(t, "259", po.ID())
	assert.Equal(t, "1", po.SyncToken())
	assert.Equal(t, "QBO", po.CDCDomain())
	assert.Equal(t, "Active", po.CDCStatus())

	require.NotNil(t, po.MetaData)
	require.NotNil(t, po.MetaData.CreateTime)

	require.Len(t, po.CustomField, 1)
	assert.Equal(t, "1", po.CustomField[0].DefinitionID)
	assert.Equal(t, "102", po.CustomField[0].StringValue)

	assert.Equal(t, "1005", po.DocNumber)
	require.NotNil(t, po.TxnDate)
	assert.Equal(t, "2015-07-28", po.TxnDate.String())
	require.NotNil(t, po.DueDate)
	assert.Equal(t, "2015-08-27", po.DueDate.String())

	require.NotNil(t, po.VendorRef)
	assert.Equal(t, "41", po.VendorRef.Value)
	require.NotNil(t, po.APAccountRef)
	assert.Equal(t, "33", po.APAccountRef.Value)
	assert.Equal(t, "25.00", po.TotalAmt.String())

	require.NotNil(t, po.POEmail)
	assert.Equal(t, "vendor@example.com", po.POEmail.Address)
	assert.Equal(t, "Open", po.POStatus)
	assert.Equal(t, "Please ship ASAP", po.Memo)

	require.NotNil(t, po.ShipTo)
	assert.Equal(t, "10", po.ShipTo.Value)
	require.NotNil(t, po.ShipMethodRef)
	assert.Equal(t, "FedEx", po.ShipMethodRef.Value)
	require.NotNil(t, po.ShipAddr)
	assert.Equal(t, "100 Main St", po.ShipAddr.Line1)
	require.NotNil(t, po.VendorAddr)
	assert.Equal(t, "200 Oak St", po.VendorAddr.Line1)
	assert.Equal(t, EmailNotSetStatus, po.EmailStatus)
	assert.Equal(t, "1.0", po.ExchangeRate.String())

	require.Len(t, po.Line, 1)
	line := po.Line[0]
	assert.Equal(t, "1", line.ID())
	assert.Equal(t, "1", line.LineNum.String())
	assert.Equal(t, "Widget A", line.Description)
	assert.Equal(t, "25.00", line.Amount.String())
	assert.Equal(t, ItemBasedExpenseLineDetailType, line.DetailType)
	require.NotNil(t, line.ItemBasedExpenseLineDetail)
	assert.Equal(t, "18", line.ItemBasedExpenseLineDetail.ItemRef.Value)
	assert.Equal(t, "5", line.ItemBasedExpenseLineDetail.Qty.String())
	assert.Equal(t, "5", line.ItemBasedExpenseLineDetail.UnitPrice.String())
	assert.Equal(t, NotBillableBillableStatus, line.ItemBasedExpenseLineDetail.BillableStatus)
	require.NotNil(t, line.ProjectRef)
	assert.Equal(t, "39298034", line.ProjectRef.Value)

	require.Len(t, po.LinkedTxn, 1)
	assert.Equal(t, "310", po.LinkedTxn[0].TxnID)
	assert.Equal(t, "Bill", po.LinkedTxn[0].TxnType)

	require.NotNil(t, po.TxnTaxDetail)
	assert.Equal(t, "0", po.TxnTaxDetail.TotalTax.String())

	require.NotNil(t, po.SalesTermRef)
	assert.Equal(t, "3", po.SalesTermRef.Value)

	require.NotNil(t, po.RecurDataRef)
	assert.Equal(t, "99", po.RecurDataRef.Value)
}

func TestPurchaseOrderRequestsMarshalWritablePayloads(t *testing.T) {
	create := PurchaseOrder{
		VendorRef:    &ReferenceType{Value: "41"},
		APAccountRef: &ReferenceType{Value: "33"},
		Line: []PurchaseOrderLine{{
			Amount:     json.Number("25"),
			DetailType: ItemBasedExpenseLineDetailType,
			ItemBasedExpenseLineDetail: &ItemBasedExpenseLineDetail{
				ItemRef: &ReferenceType{Value: "18"},
				Qty:     json.Number("5"),
			},
		}},
	}

	body, err := json.Marshal(create)
	require.NoError(t, err)
	assert.Contains(t, string(body), `"VendorRef"`)
	assert.Contains(t, string(body), `"APAccountRef"`)
	assert.Contains(t, string(body), `"ItemBasedExpenseLineDetail"`)
	assert.NotContains(t, string(body), `"sparse"`)
	assert.NotContains(t, string(body), `"TotalAmt"`)
}

func TestPurchaseOrderServiceSendUsesCorrectEndpoint(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/company/123/purchaseorder/259/send", r.URL.Path)
		assert.Equal(t, "vendor@example.com", r.URL.Query().Get("sendTo"))
		w.Header().Set("Content-Type", "application/json")
		_, err := io.WriteString(w, `{"PurchaseOrder":{"Id":"259","SyncToken":"1"}}`)
		require.NoError(t, err)
	}))
	defer server.Close()

	client, err := NewClient(WithHTTPClient(server.Client()), WithBaseURL(server.URL+"/v3/company"))
	require.NoError(t, err)

	err = client.Company("123", oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test-token"})).PurchaseOrders().Send(context.Background(), "259", "vendor@example.com")
	require.NoError(t, err)
}

func TestPurchaseOrderCDCExtraction(t *testing.T) {
	changedSince := time.Date(2026, time.March, 10, 12, 0, 0, 0, time.UTC)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/company/123/cdc", r.URL.Path)
		assert.Equal(t, string(EntityPurchaseOrder), r.URL.Query().Get("entities"))
		assert.Equal(t, changedSince.Format(time.RFC3339), r.URL.Query().Get("changedSince"))
		w.Header().Set("Content-Type", "application/json")
		_, err := io.WriteString(w, `{"CDCResponse":[{"QueryResponse":[{"PurchaseOrder":[{"Id":"259","SyncToken":"2","domain":"QBO","status":"Deleted"}],"startPosition":1,"maxResults":1}]}],"time":"2026-03-10T12:00:00Z"}`)
		require.NoError(t, err)
	}))
	defer server.Close()

	client, err := NewClient(WithHTTPClient(server.Client()), WithBaseURL(server.URL+"/v3/company"))
	require.NoError(t, err)

	pos, err := client.Company("123", oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test-token"})).PurchaseOrders().ChangeDataCapture(context.Background(), changedSince)
	require.NoError(t, err)
	require.Len(t, pos, 1)
	assert.Equal(t, "259", pos[0].ID())
	assert.Equal(t, "Deleted", pos[0].CDCStatus())
}

// Compile-time interface compliance checks for PurchaseOrder and PurchaseOrderService.
var (
	// PurchaseOrder entity type compliance.
	_ Entity    = PurchaseOrder{}
	_ CDCEntity = PurchaseOrder{}

	// PurchaseOrderService operation compliance.
	_ Getter[PurchaseOrder]          = (*PurchaseOrderService)(nil)
	_ Querier[PurchaseOrder]         = (*PurchaseOrderService)(nil)
	_ Lister[PurchaseOrder]          = (*PurchaseOrderService)(nil)
	_ Pager[PurchaseOrder]           = (*PurchaseOrderService)(nil)
	_ Creator[PurchaseOrder]         = (*PurchaseOrderService)(nil)
	_ Updater[PurchaseOrder]         = (*PurchaseOrderService)(nil)
	_ Deleter[PurchaseOrder]         = (*PurchaseOrderService)(nil)
	_ Sender                         = (*PurchaseOrderService)(nil)
	_ PDFGetter                      = (*PurchaseOrderService)(nil)
	_ PDFStreamer                    = (*PurchaseOrderService)(nil)
	_ EntityCDCReader[PurchaseOrder] = (*PurchaseOrderService)(nil)
)
