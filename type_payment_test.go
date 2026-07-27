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

func TestPaymentServiceGet(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/company/123/payment/163", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_, err := io.WriteString(w, `{"Payment":{"Id":"163","SyncToken":"0","domain":"QBO","status":"Active","CustomerRef":{"value":"20","name":"Red Rock Diner"},"TotalAmt":65.0}}`)
		require.NoError(t, err)
	}))
	defer server.Close()

	client, err := NewClient(WithHTTPClient(server.Client()), WithBaseURL(server.URL+"/v3/company"))
	require.NoError(t, err)

	pay, err := client.Company("123", oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test-token"})).Payments().Get(context.Background(), "163")
	require.NoError(t, err)
	assert.Equal(t, "163", pay.ID())
	assert.Equal(t, EntityPayment, pay.EntityName())
	assert.Equal(t, "Active", pay.CDCStatus())
	assert.Equal(t, "QBO", pay.CDCDomain())
	require.NotNil(t, pay.CustomerRef)
	assert.Equal(t, "20", pay.CustomerRef.Value)
	assert.Equal(t, "65.0", pay.TotalAmt.String())
}

func TestPaymentServiceQueryUsesQueryOptions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/company/123/query", r.URL.Path)
		assert.Equal(t, "SELECT * FROM Payment STARTPOSITION 11 MAXRESULTS 10", r.URL.Query().Get("query"))
		w.Header().Set("Content-Type", "application/json")
		_, err := io.WriteString(w, `{"QueryResponse":{"Payment":[{"Id":"163","SyncToken":"0"},{"Id":"164","SyncToken":"1"}],"startPosition":6,"maxResults":10}}`)
		require.NoError(t, err)
	}))
	defer server.Close()

	client, err := NewClient(WithHTTPClient(server.Client()), WithBaseURL(server.URL+"/v3/company"))
	require.NoError(t, err)

	pays, err := client.Company("123", oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test-token"})).Payments().ListPage(context.Background(), 2, 10)
	require.NoError(t, err)
	require.Len(t, pays, 2)
	assert.Equal(t, "163", pays[0].ID())
	assert.Equal(t, "164", pays[1].ID())
}

func TestPaymentJSONDecodesV75Fields(t *testing.T) {
	const fixture = `{
		"Payment": {
			"Id": "163",
			"SyncToken": "0",
			"domain": "QBO",
			"status": "Active",
			"MetaData": {
				"CreateTime": "2015-01-16T15:08:12-08:00",
				"LastUpdatedTime": "2015-01-16T15:08:12-08:00"
			},
			"CustomerRef": {"value": "20", "name": "Red Rock Diner"},
			"TotalAmt": 65.0,
			"UnappliedAmt": 10.0,
			"TxnDate": "2015-01-16",
			"PaymentRefNum": "PAY-163",
			"DepositToAccountRef": {"value": "4"},
			"PaymentMethodRef": {"value": "2", "name": "Check"},
			"CurrencyRef": {"value": "USD", "name": "United States Dollar"},
			"ProjectRef": {"value": "39298034"},
			"ProcessPayment": false,
			"Line": [
				{
					"Amount": 55.0,
					"LinkedTxn": [
						{"TxnId": "70", "TxnType": "Invoice"}
					]
				}
			],
			"CreditCardPayment": {
				"CreditChargeResponse": {
					"Status": "Completed",
					"AuthCode": "AUTH789",
					"CCTransId": "TXN321"
				}
			}
		}
	}`

	var envelope struct {
		Payment Payment `json:"Payment"`
	}
	require.NoError(t, json.Unmarshal([]byte(fixture), &envelope))

	pay := envelope.Payment
	assert.Equal(t, "163", pay.ID())
	assert.Equal(t, "0", pay.SyncToken())
	assert.Equal(t, "QBO", pay.CDCDomain())
	assert.Equal(t, "Active", pay.CDCStatus())

	require.NotNil(t, pay.MetaData)
	require.NotNil(t, pay.MetaData.CreateTime)

	require.NotNil(t, pay.CustomerRef)
	assert.Equal(t, "20", pay.CustomerRef.Value)

	assert.Equal(t, "65.0", pay.TotalAmt.String())
	assert.Equal(t, "10.0", pay.UnappliedAmt.String())

	require.NotNil(t, pay.TxnDate)
	assert.Equal(t, "2015-01-16", pay.TxnDate.String())

	assert.Equal(t, "PAY-163", pay.PaymentRefNum)

	require.NotNil(t, pay.DepositToAccountRef)
	assert.Equal(t, "4", pay.DepositToAccountRef.Value)

	require.NotNil(t, pay.PaymentMethodRef)
	assert.Equal(t, "2", pay.PaymentMethodRef.Value)

	require.NotNil(t, pay.CurrencyRef)
	assert.Equal(t, "USD", pay.CurrencyRef.Value)

	require.NotNil(t, pay.ProjectRef)
	assert.Equal(t, "39298034", pay.ProjectRef.Value)

	require.NotNil(t, pay.ProcessPayment)
	assert.False(t, *pay.ProcessPayment)

	require.Len(t, pay.Line, 1)
	assert.Equal(t, "55.0", pay.Line[0].Amount.String())
	assert.Equal(t, "55.0", pay.Line[0].Amount.String())
	float, err := pay.Line[0].Amount.Float64()
	require.NoError(t, err)
	assert.Equal(t, 55.0, float)
	require.Len(t, pay.Line[0].LinkedTxn, 1)
	assert.Equal(t, "70", pay.Line[0].LinkedTxn[0].TxnID)
	assert.Equal(t, "Invoice", pay.Line[0].LinkedTxn[0].TxnType)

	require.NotNil(t, pay.CreditCardPayment)
	require.NotNil(t, pay.CreditCardPayment.CreditChargeResponse)
	assert.Equal(t, "Completed", pay.CreditCardPayment.CreditChargeResponse.Status)
	assert.Equal(t, "AUTH789", pay.CreditCardPayment.CreditChargeResponse.AuthCode)
	assert.Equal(t, "TXN321", pay.CreditCardPayment.CreditChargeResponse.CCTransId)
}

func TestPaymentRequestsMarshalWritablePayloads(t *testing.T) {
	create := Payment{
		TotalAmt:    json.Number("55"),
		CustomerRef: &ReferenceType{Value: "20"},
		Line: []PaymentLine{{
			Amount:    json.Number("55"),
			LinkedTxn: []LinkedTxn{{TxnID: "69", TxnType: "Invoice"}},
		}},
	}

	body, err := json.Marshal(create)
	require.NoError(t, err)
	assert.Contains(t, string(body), `"TotalAmt"`)
	assert.Contains(t, string(body), `"CustomerRef"`)
	assert.Contains(t, string(body), `"LinkedTxn"`)
	assert.NotContains(t, string(body), `"sparse"`)
	assert.NotContains(t, string(body), `"PaymentRefNum"`)
}

func TestPaymentVoidUsesCorrectEndpoint(t *testing.T) {
	var capturedBody []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/company/123/payment", r.URL.Path)
		assert.Equal(t, "update", r.URL.Query().Get("operation"))
		assert.Equal(t, "void", r.URL.Query().Get("include"))
		var err error
		capturedBody, err = io.ReadAll(r.Body)
		require.NoError(t, err)
		w.Header().Set("Content-Type", "application/json")
		_, err = io.WriteString(w, `{"Payment":{"Id":"33","SyncToken":"2","TotalAmt":0,"PrivateNote":"Voided"}}`)
		require.NoError(t, err)
	}))
	defer server.Close()

	client, err := NewClient(WithHTTPClient(server.Client()), WithBaseURL(server.URL+"/v3/company"))
	require.NoError(t, err)

	result, err := client.Company("123", oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test-token"})).Payments().Void(
		context.Background(),
		VoidRequest{IDValue: IDValue{Id: "33"}, SyncTokenValue: SyncTokenValue{SyncTokenValue: "1"}},
	)
	require.NoError(t, err)
	assert.Equal(t, "33", result.ID())
	assert.Equal(t, "Voided", result.PrivateNote)

	assert.Contains(t, string(capturedBody), `"Id":"33"`)
	assert.Contains(t, string(capturedBody), `"SyncToken":"1"`)
	assert.Contains(t, string(capturedBody), `"sparse":true`)
}

func TestPaymentServiceSendUsesCorrectEndpoint(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/company/123/payment/163/send", r.URL.Path)
		assert.Equal(t, "redrock@intuit.com", r.URL.Query().Get("sendTo"))
		w.Header().Set("Content-Type", "application/json")
		_, err := io.WriteString(w, `{"Payment":{"Id":"163","SyncToken":"1"}}`)
		require.NoError(t, err)
	}))
	defer server.Close()

	client, err := NewClient(WithHTTPClient(server.Client()), WithBaseURL(server.URL+"/v3/company"))
	require.NoError(t, err)

	err = client.Company("123", oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test-token"})).Payments().Send(context.Background(), "163", "redrock@intuit.com")
	require.NoError(t, err)
}

func TestPaymentCDCExtraction(t *testing.T) {
	changedSince := time.Date(2026, time.March, 10, 12, 0, 0, 0, time.UTC)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/company/123/cdc", r.URL.Path)
		assert.Equal(t, string(EntityPayment), r.URL.Query().Get("entities"))
		assert.Equal(t, changedSince.Format(time.RFC3339), r.URL.Query().Get("changedSince"))
		w.Header().Set("Content-Type", "application/json")
		_, err := io.WriteString(w, `{"CDCResponse":[{"QueryResponse":[{"Payment":[{"Id":"163","SyncToken":"1","domain":"QBO","status":"Deleted"}],"startPosition":1,"maxResults":1}]}],"time":"2026-03-10T12:00:00Z"}`)
		require.NoError(t, err)
	}))
	defer server.Close()

	client, err := NewClient(WithHTTPClient(server.Client()), WithBaseURL(server.URL+"/v3/company"))
	require.NoError(t, err)

	pays, err := client.Company("123", oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test-token"})).Payments().ChangeDataCapture(context.Background(), changedSince)
	require.NoError(t, err)
	require.Len(t, pays, 1)
	assert.Equal(t, "163", pays[0].ID())
	assert.Equal(t, "Deleted", pays[0].CDCStatus())
}

// Compile-time interface compliance checks for Payment and PaymentService.
var (
	// Payment entity type compliance.
	_ Entity    = Payment{}
	_ CDCEntity = Payment{}

	// PaymentService operation compliance.
	_ Getter[Payment]          = (*PaymentService)(nil)
	_ Querier[Payment]         = (*PaymentService)(nil)
	_ Lister[Payment]          = (*PaymentService)(nil)
	_ Pager[Payment]           = (*PaymentService)(nil)
	_ Creator[Payment]         = (*PaymentService)(nil)
	_ Updater[Payment]         = (*PaymentService)(nil)
	_ SparseUpdater[Payment]   = (*PaymentService)(nil)
	_ Deleter[Payment]         = (*PaymentService)(nil)
	_ Voider[Payment]          = (*PaymentService)(nil)
	_ Sender                   = (*PaymentService)(nil)
	_ PDFGetter                = (*PaymentService)(nil)
	_ PDFStreamer              = (*PaymentService)(nil)
	_ EntityCDCReader[Payment] = (*PaymentService)(nil)
)

func TestPaymentServiceUpdateSparseInjectsSparseFlag(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/company/123/payment", r.URL.Path)
		var payload map[string]interface{}
		require.NoError(t, json.NewDecoder(r.Body).Decode(&payload))
		assert.Equal(t, true, payload["sparse"])
		w.Header().Set("Content-Type", "application/json")
		_, err := io.WriteString(w, `{"Payment":{"Id":"1","SyncToken":"2"}}`)
		require.NoError(t, err)
	}))
	defer server.Close()

	client, err := NewClient(WithHTTPClient(server.Client()), WithBaseURL(server.URL+"/v3/company"))
	require.NoError(t, err)

	payment := Payment{}
	payment.Id = "1"
	payment.SyncTokenValue = SyncTokenValue{"1"}

	_, err = client.Company("123", oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test-token"})).Payments().UpdateSparse(context.Background(), payment)
	require.NoError(t, err)
}
