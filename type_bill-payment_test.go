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

func TestBillPaymentServiceGet(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/company/123/billpayment/236", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_, err := io.WriteString(w, `{"BillPayment":{"Id":"236","SyncToken":"0","domain":"QBO","status":"Active","VendorRef":{"value":"56","name":"Bob's Burger Joint"},"PayType":"Check","TotalAmt":200.0,"CheckPayment":{"BankAccountRef":{"value":"35","name":"Checking"},"PrintStatus":"NeedToPrint"}}}`)
		require.NoError(t, err)
	}))
	defer server.Close()

	client, err := NewClient(WithHTTPClient(server.Client()), WithBaseURL(server.URL+"/v3/company"))
	require.NoError(t, err)

	bp, err := client.Company("123", oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test-token"})).BillPayments().Get(context.Background(), "236")
	require.NoError(t, err)
	assert.Equal(t, "236", bp.ID())
	assert.Equal(t, EntityBillPayment, bp.EntityName())
	assert.Equal(t, "Active", bp.CDCStatus())
	assert.Equal(t, "QBO", bp.CDCDomain())
	assert.Equal(t, CheckBillPaymentType, bp.PayType)
	require.NotNil(t, bp.VendorRef)
	assert.Equal(t, "56", bp.VendorRef.Value)
	require.NotNil(t, bp.CheckPayment)
	assert.Equal(t, "35", bp.CheckPayment.BankAccountRef.Value)
	assert.Equal(t, NeedToPrintStatus, bp.CheckPayment.PrintStatus)
}

func TestBillPaymentServiceQueryUsesQueryOptions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/company/123/query", r.URL.Path)
		assert.Equal(t, "SELECT * FROM BillPayment STARTPOSITION 11 MAXRESULTS 10", r.URL.Query().Get("query"))
		w.Header().Set("Content-Type", "application/json")
		_, err := io.WriteString(w, `{"QueryResponse":{"BillPayment":[{"Id":"236","SyncToken":"0"},{"Id":"237","SyncToken":"1"}],"startPosition":6,"maxResults":10}}`)
		require.NoError(t, err)
	}))
	defer server.Close()

	client, err := NewClient(WithHTTPClient(server.Client()), WithBaseURL(server.URL+"/v3/company"))
	require.NoError(t, err)

	bps, err := client.Company("123", oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test-token"})).BillPayments().ListPage(context.Background(), 2, 10)
	require.NoError(t, err)
	require.Len(t, bps, 2)
	assert.Equal(t, "236", bps[0].ID())
	assert.Equal(t, "237", bps[1].ID())
}

func TestBillPaymentJSONDecodesCheckVariant(t *testing.T) {
	const fixture = `{
		"BillPayment": {
			"Id": "236",
			"SyncToken": "0",
			"domain": "QBO",
			"status": "Active",
			"MetaData": {
				"CreateTime": "2015-07-14T12:34:04-07:00",
				"LastUpdatedTime": "2015-07-14T12:34:04-07:00"
			},
			"VendorRef": {"value": "56", "name": "Bob's Burger Joint"},
			"APAccountRef": {"value": "33", "name": "Accounts Payable (A/P)"},
			"TxnDate": "2015-07-14",
			"TotalAmt": 200.0,
			"PayType": "Check",
			"PrivateNote": "Acct. 1JK90",
			"DocNumber": "BP-236",
			"DepartmentRef": {"value": "1"},
			"CurrencyRef": {"value": "USD", "name": "United States Dollar"},
			"Line": [
				{
					"Amount": 200.0,
					"LinkedTxn": [
						{"TxnId": "234", "TxnType": "Bill"}
					]
				}
			],
			"CheckPayment": {
				"BankAccountRef": {"value": "35", "name": "Checking"},
				"PrintStatus": "NeedToPrint"
			}
		}
	}`

	var envelope struct {
		BillPayment BillPayment `json:"BillPayment"`
	}
	require.NoError(t, json.Unmarshal([]byte(fixture), &envelope))

	bp := envelope.BillPayment
	assert.Equal(t, "236", bp.ID())
	assert.Equal(t, "0", bp.SyncToken())
	assert.Equal(t, "QBO", bp.CDCDomain())
	assert.Equal(t, "Active", bp.CDCStatus())

	require.NotNil(t, bp.MetaData)
	require.NotNil(t, bp.MetaData.CreateTime)

	require.NotNil(t, bp.VendorRef)
	assert.Equal(t, "56", bp.VendorRef.Value)

	require.NotNil(t, bp.APAccountRef)
	assert.Equal(t, "33", bp.APAccountRef.Value)

	require.NotNil(t, bp.TxnDate)
	assert.Equal(t, "2015-07-14", bp.TxnDate.String())

	assert.Equal(t, "200.0", bp.TotalAmt.String())
	assert.Equal(t, CheckBillPaymentType, bp.PayType)
	assert.Equal(t, "Acct. 1JK90", bp.PrivateNote)
	assert.Equal(t, "BP-236", bp.DocNumber)

	require.NotNil(t, bp.CurrencyRef)
	assert.Equal(t, "USD", bp.CurrencyRef.Value)

	require.NotNil(t, bp.DepartmentRef)
	assert.Equal(t, "1", bp.DepartmentRef.Value)

	require.Len(t, bp.Line, 1)
	assert.Equal(t, "200.0", bp.Line[0].Amount.String())
	require.Len(t, bp.Line[0].LinkedTxn, 1)
	assert.Equal(t, "234", bp.Line[0].LinkedTxn[0].TxnID)
	assert.Equal(t, "Bill", bp.Line[0].LinkedTxn[0].TxnType)

	require.NotNil(t, bp.CheckPayment)
	assert.Equal(t, "35", bp.CheckPayment.BankAccountRef.Value)
	assert.Equal(t, NeedToPrintStatus, bp.CheckPayment.PrintStatus)
	assert.Nil(t, bp.CreditCardPayment)
}

func TestBillPaymentJSONDecodesCreditCardVariant(t *testing.T) {
	const fixture = `{
		"BillPayment": {
			"Id": "104",
			"SyncToken": "0",
			"domain": "QBO",
			"status": "Active",
			"VendorRef": {"value": "40", "name": "Hall Properties"},
			"TotalAmt": 500.0,
			"PayType": "CreditCard",
			"Line": [
				{
					"Amount": 500.0,
					"LinkedTxn": [{"TxnId": "89", "TxnType": "Bill"}]
				}
			],
			"CreditCardPayment": {
				"CCAccountRef": {"value": "42", "name": "Mastercard"},
				"CreditChargeInfo": {
					"NameOnAcct": "John Doe",
					"CcExpiryMonth": 12,
					"CcExpiryYear": 2027,
					"Amount": 500.0,
					"Type": "Charge"
				},
				"CreditChargeResponse": {
					"Status": "Completed",
					"AuthCode": "AUTH123",
					"CCTransId": "TXN456"
				}
			}
		}
	}`

	var envelope struct {
		BillPayment BillPayment `json:"BillPayment"`
	}
	require.NoError(t, json.Unmarshal([]byte(fixture), &envelope))

	bp := envelope.BillPayment
	assert.Equal(t, "104", bp.ID())
	assert.Equal(t, CreditCardBillPaymentType, bp.PayType)
	assert.Nil(t, bp.CheckPayment)

	require.NotNil(t, bp.CreditCardPayment)
	assert.Equal(t, "42", bp.CreditCardPayment.CCAccountRef.Value)

	require.NotNil(t, bp.CreditCardPayment.CreditChargeInfo)
	assert.Equal(t, "John Doe", bp.CreditCardPayment.CreditChargeInfo.NameOnAcct)
	assert.Equal(t, 12, bp.CreditCardPayment.CreditChargeInfo.CcExpiryMonth)
	assert.Equal(t, 2027, bp.CreditCardPayment.CreditChargeInfo.CcExpiryYear)
	assert.Equal(t, "Charge", bp.CreditCardPayment.CreditChargeInfo.Type)

	require.NotNil(t, bp.CreditCardPayment.CreditChargeResponse)
	assert.Equal(t, "Completed", bp.CreditCardPayment.CreditChargeResponse.Status)
	assert.Equal(t, "AUTH123", bp.CreditCardPayment.CreditChargeResponse.AuthCode)
	assert.Equal(t, "TXN456", bp.CreditCardPayment.CreditChargeResponse.CCTransId)
}

func TestBillPaymentRequestsMarshalWritablePayloads(t *testing.T) {
	create := BillPayment{
		VendorRef: &ReferenceType{Value: "56"},
		TotalAmt:  json.Number("200"),
		PayType:   CheckBillPaymentType,
		Line: []BillPaymentLine{{
			Amount:    json.Number("200"),
			LinkedTxn: []LinkedTxn{{TxnID: "234", TxnType: "Bill"}},
		}},
		CheckPayment: &BillPaymentCheck{
			BankAccountRef: &ReferenceType{Value: "35"},
		},
	}

	body, err := json.Marshal(create)
	require.NoError(t, err)
	assert.Contains(t, string(body), `"VendorRef"`)
	assert.Contains(t, string(body), `"PayType":"Check"`)
	assert.Contains(t, string(body), `"CheckPayment"`)
	assert.NotContains(t, string(body), `"sparse"`)
}

func TestBillPaymentVoidUsesCorrectEndpoint(t *testing.T) {
	var capturedBody []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/company/123/billpayment", r.URL.Path)
		assert.Equal(t, "update", r.URL.Query().Get("operation"))
		assert.Equal(t, "void", r.URL.Query().Get("include"))
		var err error
		capturedBody, err = io.ReadAll(r.Body)
		require.NoError(t, err)
		w.Header().Set("Content-Type", "application/json")
		_, err = io.WriteString(w, `{"BillPayment":{"Id":"104","SyncToken":"1","TotalAmt":0,"PrivateNote":"Voided"}}`)
		require.NoError(t, err)
	}))
	defer server.Close()

	client, err := NewClient(WithHTTPClient(server.Client()), WithBaseURL(server.URL+"/v3/company"))
	require.NoError(t, err)

	result, err := client.Company("123", oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test-token"})).BillPayments().Void(
		context.Background(),
		VoidRequest{IDValue: IDValue{Id: "104"}, SyncTokenValue: SyncTokenValue{SyncTokenValue: "0"}},
	)
	require.NoError(t, err)
	assert.Equal(t, "104", result.ID())
	assert.Equal(t, "Voided", result.PrivateNote)

	assert.Contains(t, string(capturedBody), `"Id":"104"`)
	assert.Contains(t, string(capturedBody), `"SyncToken":"0"`)
	assert.Contains(t, string(capturedBody), `"sparse":true`)
}

func TestBillPaymentCDCExtraction(t *testing.T) {
	changedSince := time.Date(2026, time.March, 10, 12, 0, 0, 0, time.UTC)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/company/123/cdc", r.URL.Path)
		assert.Equal(t, string(EntityBillPayment), r.URL.Query().Get("entities"))
		assert.Equal(t, changedSince.Format(time.RFC3339), r.URL.Query().Get("changedSince"))
		w.Header().Set("Content-Type", "application/json")
		_, err := io.WriteString(w, `{"CDCResponse":[{"QueryResponse":[{"BillPayment":[{"Id":"236","SyncToken":"1","domain":"QBO","status":"Deleted"}],"startPosition":1,"maxResults":1}]}],"time":"2026-03-10T12:00:00Z"}`)
		require.NoError(t, err)
	}))
	defer server.Close()

	client, err := NewClient(WithHTTPClient(server.Client()), WithBaseURL(server.URL+"/v3/company"))
	require.NoError(t, err)

	bps, err := client.Company("123", oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test-token"})).BillPayments().ChangeDataCapture(context.Background(), changedSince)
	require.NoError(t, err)
	require.Len(t, bps, 1)
	assert.Equal(t, "236", bps[0].ID())
	assert.Equal(t, "Deleted", bps[0].CDCStatus())
}

// Compile-time interface compliance checks for BillPayment and BillPaymentService.
var (
	// BillPayment entity type compliance.
	_ Entity    = BillPayment{}
	_ CDCEntity = BillPayment{}

	// BillPaymentService operation compliance.
	_ Getter[BillPayment]          = (*BillPaymentService)(nil)
	_ Querier[BillPayment]         = (*BillPaymentService)(nil)
	_ Lister[BillPayment]          = (*BillPaymentService)(nil)
	_ Pager[BillPayment]           = (*BillPaymentService)(nil)
	_ Creator[BillPayment]         = (*BillPaymentService)(nil)
	_ Updater[BillPayment]         = (*BillPaymentService)(nil)
	_ SparseUpdater[BillPayment]   = (*BillPaymentService)(nil)
	_ Deleter[BillPayment]         = (*BillPaymentService)(nil)
	_ Voider[BillPayment]          = (*BillPaymentService)(nil)
	_ EntityCDCReader[BillPayment] = (*BillPaymentService)(nil)
)

func TestBillPaymentServiceUpdateSparseInjectsSparseFlag(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/company/123/billpayment", r.URL.Path)
		var payload map[string]interface{}
		require.NoError(t, json.NewDecoder(r.Body).Decode(&payload))
		assert.Equal(t, true, payload["sparse"])
		w.Header().Set("Content-Type", "application/json")
		_, err := io.WriteString(w, `{"BillPayment":{"Id":"1","SyncToken":"2"}}`)
		require.NoError(t, err)
	}))
	defer server.Close()

	client, err := NewClient(WithHTTPClient(server.Client()), WithBaseURL(server.URL+"/v3/company"))
	require.NoError(t, err)

	bp := BillPayment{}
	bp.Id = "1"
	bp.SyncTokenValue = SyncTokenValue{"1"}

	_, err = client.Company("123", oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test-token"})).BillPayments().UpdateSparse(context.Background(), bp)
	require.NoError(t, err)
}
