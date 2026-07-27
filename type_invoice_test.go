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

func TestInvoiceServiceGet(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/company/123/invoice/10", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_, err := io.WriteString(w, `{"Invoice":{"Id":"10","SyncToken":"1","domain":"QBO","status":"Deleted","DocNumber":"INV-10"}}`)
		require.NoError(t, err)
	}))
	defer server.Close()

	client, err := NewClient(WithHTTPClient(server.Client()), WithBaseURL(server.URL+"/v3/company"))
	require.NoError(t, err)

	invoice, err := client.Company("123", oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test-token"})).Invoices().Get(context.Background(), "10")
	require.NoError(t, err)
	assert.Equal(t, "10", invoice.ID())
	assert.Equal(t, EntityInvoice, invoice.EntityName())
	assert.Equal(t, "Deleted", invoice.CDCStatus())
	assert.Equal(t, "QBO", invoice.CDCDomain())
}

func TestInvoiceServiceQueryUsesQueryOptions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/company/123/query", r.URL.Path)
		assert.Equal(t, "SELECT * FROM Invoice STARTPOSITION 26 MAXRESULTS 25", r.URL.Query().Get("query"))
		w.Header().Set("Content-Type", "application/json")
		_, err := io.WriteString(w, `{"QueryResponse":{"Invoice":[{"Id":"11","SyncToken":"0"}],"startPosition":11,"maxResults":25}}`)
		require.NoError(t, err)
	}))
	defer server.Close()

	client, err := NewClient(WithHTTPClient(server.Client()), WithBaseURL(server.URL+"/v3/company"))
	require.NoError(t, err)

	invoices, err := client.Company("123", oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test-token"})).Invoices().ListPage(context.Background(), 2, 25)
	require.NoError(t, err)
	require.Len(t, invoices, 1)
	assert.Equal(t, "11", invoices[0].ID())
}

func TestCompanyChangeDataCaptureAndInvoiceExtraction(t *testing.T) {
	changedSince := time.Date(2026, time.March, 10, 12, 0, 0, 0, time.UTC)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/company/123/cdc", r.URL.Path)
		assert.Equal(t, string(EntityInvoice), r.URL.Query().Get("entities"))
		assert.Equal(t, changedSince.Format(time.RFC3339), r.URL.Query().Get("changedSince"))
		w.Header().Set("Content-Type", "application/json")
		_, err := io.WriteString(w, `{"CDCResponse":[{"QueryResponse":[{"Invoice":[{"Id":"99","SyncToken":"2","domain":"QBO","status":"Deleted"}],"startPosition":1,"maxResults":1}]}],"time":"2026-03-10T12:00:00Z"}`)
		require.NoError(t, err)
	}))
	defer server.Close()

	client, err := NewClient(WithHTTPClient(server.Client()), WithBaseURL(server.URL+"/v3/company"))
	require.NoError(t, err)

	company := client.Company("123", oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test-token"}))
	resp, err := company.ChangeDataCapture(context.Background(), changedSince, []EntityName{EntityInvoice})
	require.NoError(t, err)

	invoices, err := company.Invoices().ExtractCDC(resp)
	require.NoError(t, err)
	require.Len(t, invoices, 1)
	assert.Equal(t, "99", invoices[0].ID())
	assert.Equal(t, "Deleted", invoices[0].CDCStatus())
}

func TestInvoiceJSONDecodesV75Fields(t *testing.T) {
	var envelope struct {
		Invoice Invoice `json:"Invoice"`
	}

	err := json.Unmarshal([]byte(`{
		"Invoice": {
			"Id": "42",
			"SyncToken": "3",
			"MetaData": {
				"CreateTime": "2026-03-10T12:00:00-07:00",
				"LastUpdatedTime": "2026-03-11T12:00:00-07:00"
			},
			"domain": "QBO",
			"status": "Active",
			"DocNumber": "INV-42",
			"TxnDate": "2026-03-10",
			"DueDate": "2026-03-25",
			"CustomerRef": {"value": "10", "name": "Acme Co"},
			"BillAddr": {"Line1": "123 Main St", "City": "Denver", "PostalCode": "80202"},
			"BillEmail": {"Address": "billing@example.com"},
			"Line": [
				{
					"Amount": 100,
					"DetailType": "SalesItemLineDetail",
					"SalesItemLineDetail": {
						"ItemRef": {"value": "5"},
						"Qty": 2,
						"UnitPrice": 50,
						"ServiceDate": "2026-03-10",
						"TaxCodeRef": {"value": "TAX"}
					}
				},
				{
					"Amount": 10,
					"DetailType": "DiscountLineDetail",
					"DiscountLineDetail": {
						"PercentBased": true,
						"DiscountPercent": 10
					}
				}
			],
			"TxnTaxDetail": {
				"TxnTaxCodeRef": {"value": "8"},
				"TotalTax": 8,
				"TaxLine": [
					{
						"Amount": 8,
						"DetailType": "TaxLineDetail",
						"TaxLineDetail": {
							"TaxRateRef": {"value": "1"},
							"NetAmountTaxable": 90
						}
					}
				]
			}
		}
	}`), &envelope)
	require.NoError(t, err)

	invoice := envelope.Invoice
	assert.Equal(t, "42", invoice.ID())
	assert.Equal(t, "INV-42", invoice.DocNumber)
	require.NotNil(t, invoice.TxnDate)
	assert.Equal(t, "2026-03-10", invoice.TxnDate.String())
	require.NotNil(t, invoice.DueDate)
	assert.Equal(t, "2026-03-25", invoice.DueDate.String())
	require.NotNil(t, invoice.CustomerRef)
	assert.Equal(t, "10", invoice.CustomerRef.Value)
	require.NotNil(t, invoice.BillAddr)
	assert.Equal(t, "123 Main St", invoice.BillAddr.Line1)
	require.NotNil(t, invoice.BillEmail)
	assert.Equal(t, "billing@example.com", invoice.BillEmail.Address)
	require.Len(t, invoice.Line, 2)
	assert.Equal(t, SalesItemLineDetailType, invoice.Line[0].DetailType)
	require.NotNil(t, invoice.Line[0].SalesItemLineDetail)
	assert.Equal(t, "5", invoice.Line[0].SalesItemLineDetail.ItemRef.Value)
	assert.Equal(t, "2", invoice.Line[0].SalesItemLineDetail.Qty.String())
	assert.Equal(t, DiscountLineDetailType, invoice.Line[1].DetailType)
	require.NotNil(t, invoice.Line[1].DiscountLineDetail)
	assert.Equal(t, "10", invoice.Line[1].DiscountLineDetail.DiscountPercent.String())
	require.NotNil(t, invoice.TxnTaxDetail)
	assert.Equal(t, "8", invoice.TxnTaxDetail.TxnTaxCodeRef.Value)
	require.Len(t, invoice.TxnTaxDetail.TaxLine, 1)
	require.NotNil(t, invoice.TxnTaxDetail.TaxLine[0].TaxLineDetail)
	assert.Equal(t, "90", invoice.TxnTaxDetail.TaxLine[0].TaxLineDetail.NetAmountTaxable.String())
	require.NotNil(t, invoice.MetaData)
	require.NotNil(t, invoice.MetaData.CreateTime)
	assert.Equal(t, "QBO", invoice.CDCDomain())
	assert.Equal(t, "Active", invoice.CDCStatus())
}

func TestInvoiceRequestsMarshalWritablePayloads(t *testing.T) {
	create := Invoice{
		DocNumber: "INV-100",
		Line: []InvoiceLine{{
			Amount:     json.Number("25"),
			DetailType: SalesItemLineDetailType,
			SalesItemLineDetail: &SalesItemLineDetail{
				ItemRef: &ReferenceType{Value: "5"},
			},
		}},
		CustomerRef:        &ReferenceType{Value: "10"},
		BillEmail:          &EmailAddress{Address: "billing@example.com"},
		AllowOnlinePayment: boolPtr(true),
	}

	body, err := json.Marshal(create)
	require.NoError(t, err)
	assert.Contains(t, string(body), `"DocNumber":"INV-100"`)
	assert.Contains(t, string(body), `"AllowOnlinePayment":true`)
	assert.NotContains(t, string(body), `"InvoiceStatus"`)
	assert.NotContains(t, string(body), `"Balance"`)
	assert.NotContains(t, string(body), `"sparse"`)
}

func TestInvoiceServiceUpdateSparseInjectsSparseFlag(t *testing.T) {
	var capturedBody []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/company/123/invoice", r.URL.Path)
		var err error
		capturedBody, err = io.ReadAll(r.Body)
		require.NoError(t, err)
		w.Header().Set("Content-Type", "application/json")
		_, err = io.WriteString(w, `{"Invoice":{"Id":"100","SyncToken":"3"}}`)
		require.NoError(t, err)
	}))
	defer server.Close()

	client, err := NewClient(WithHTTPClient(server.Client()), WithBaseURL(server.URL+"/v3/company"))
	require.NoError(t, err)

	invoice := Invoice{
		IDValue:        IDValue{Id: "100"},
		SyncTokenValue: SyncTokenValue{SyncTokenValue: "2"},
		PrivateNote:    "updated",
	}

	_, err = client.Company("123", oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test-token"})).Invoices().UpdateSparse(context.Background(), invoice)
	require.NoError(t, err)

	assert.Contains(t, string(capturedBody), `"Id":"100"`)
	assert.Contains(t, string(capturedBody), `"SyncToken":"2"`)
	assert.Contains(t, string(capturedBody), `"sparse":true`)
	assert.Contains(t, string(capturedBody), `"PrivateNote":"updated"`)
}

func boolPtr(v bool) *bool {
	return &v
}

func TestChangeDataCaptureRejectsUnsupportedEntity(t *testing.T) {
	client, err := NewClient()
	require.NoError(t, err)

	_, err = client.Company("123", oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test-token"})).ChangeDataCapture(
		context.Background(),
		time.Date(2026, time.March, 10, 12, 0, 0, 0, time.UTC),
		[]EntityName{"TaxAgency"},
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "does not support change data capture")
}

// Compile-time interface compliance checks for Invoice and InvoiceService.
// These assignments fail at build time if a method signature drifts out of
// alignment with the declared interface, providing a clear signal when an
// entity service needs to be updated.
var (
	// Invoice entity type compliance.
	_ Entity    = Invoice{}
	_ CDCEntity = Invoice{}

	// InvoiceService operation compliance.
	_ Getter[Invoice]          = (*InvoiceService)(nil)
	_ Querier[Invoice]         = (*InvoiceService)(nil)
	_ Lister[Invoice]          = (*InvoiceService)(nil)
	_ Pager[Invoice]           = (*InvoiceService)(nil)
	_ Creator[Invoice]         = (*InvoiceService)(nil)
	_ Updater[Invoice]         = (*InvoiceService)(nil)
	_ SparseUpdater[Invoice]   = (*InvoiceService)(nil)
	_ Sender                   = (*InvoiceService)(nil)
	_ Deleter[Invoice]         = (*InvoiceService)(nil)
	_ Voider[Invoice]          = (*InvoiceService)(nil)
	_ EntityCDCReader[Invoice] = (*InvoiceService)(nil)
	_ PDFGetter                = (*InvoiceService)(nil)
	_ PDFStreamer              = (*InvoiceService)(nil)
)
