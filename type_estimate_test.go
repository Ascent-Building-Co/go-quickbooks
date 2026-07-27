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

func TestEstimateServiceGet(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/company/123/estimate/177", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_, err := io.WriteString(w, `{"Estimate":{"Id":"177","SyncToken":"0","domain":"QBO","status":"Active","CustomerRef":{"value":"3","name":"Cool Cars"},"TxnStatus":"Pending"}}`)
		require.NoError(t, err)
	}))
	defer server.Close()

	client, err := NewClient(WithHTTPClient(server.Client()), WithBaseURL(server.URL+"/v3/company"))
	require.NoError(t, err)

	est, err := client.Company("123", oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test-token"})).Estimates().Get(context.Background(), "177")
	require.NoError(t, err)
	assert.Equal(t, "177", est.ID())
	assert.Equal(t, EntityEstimate, est.EntityName())
	assert.Equal(t, "Active", est.CDCStatus())
	assert.Equal(t, "QBO", est.CDCDomain())
	require.NotNil(t, est.CustomerRef)
	assert.Equal(t, "3", est.CustomerRef.Value)
	assert.Equal(t, EstimatePending, est.TxnStatus)
}

func TestEstimateServiceQueryUsesQueryOptions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/company/123/query", r.URL.Path)
		assert.Equal(t, "SELECT * FROM Estimate STARTPOSITION 11 MAXRESULTS 10", r.URL.Query().Get("query"))
		w.Header().Set("Content-Type", "application/json")
		_, err := io.WriteString(w, `{"QueryResponse":{"Estimate":[{"Id":"177","SyncToken":"0"},{"Id":"178","SyncToken":"1"}],"startPosition":6,"maxResults":10}}`)
		require.NoError(t, err)
	}))
	defer server.Close()

	client, err := NewClient(WithHTTPClient(server.Client()), WithBaseURL(server.URL+"/v3/company"))
	require.NoError(t, err)

	ests, err := client.Company("123", oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test-token"})).Estimates().ListPage(context.Background(), 2, 10)
	require.NoError(t, err)
	require.Len(t, ests, 2)
	assert.Equal(t, "177", ests[0].ID())
	assert.Equal(t, "178", ests[1].ID())
}

func TestEstimateJSONDecodesV75Fields(t *testing.T) {
	const fixture = `{
		"Estimate": {
			"Id": "177",
			"SyncToken": "0",
			"domain": "QBO",
			"status": "Active",
			"MetaData": {
				"CreateTime": "2015-03-26T13:25:05-07:00",
				"LastUpdatedTime": "2015-03-26T13:25:05-07:00"
			},
			"CustomField": [
				{"DefinitionId": "1", "Type": "StringType", "Name": "Crew #"}
			],
			"CustomerRef": {"value": "3", "name": "Cool Cars"},
			"BillEmail": {"Address": "Cool_Cars@intuit.com"},
			"ProjectRef": {"value": "39298034"},
			"CurrencyRef": {"value": "USD", "name": "United States Dollar"},
			"TxnDate": "2015-03-26",
			"ExpirationDate": "2015-04-26",
			"DocNumber": "1001",
			"PrivateNote": "internal note",
			"TxnStatus": "Pending",
			"PrintStatus": "NeedToPrint",
			"EmailStatus": "NotSet",
			"ApplyTaxAfterDiscount": false,
			"AcceptedBy": "John Doe",
			"AcceptedDate": "2015-03-27",
			"TotalAmt": 31.5,
			"CustomerMemo": {"value": "Thank you for your business and have a great day!"},
			"BillAddr": {
				"Line1": "65 Ocean Dr.",
				"City": "Half Moon Bay",
				"CountrySubDivisionCode": "CA",
				"PostalCode": "94213",
				"Id": "103"
			},
			"ShipAddr": {
				"Line1": "65 Ocean Dr.",
				"City": "Half Moon Bay",
				"CountrySubDivisionCode": "CA",
				"PostalCode": "94213",
				"Id": "104"
			},
			"LinkedTxn": [
				{"TxnId": "95", "TxnType": "Invoice"}
			],
			"Line": [
				{
					"Id": "1",
					"LineNum": 1,
					"Description": "Pest Control Services",
					"Amount": 35.0,
					"DetailType": "SalesItemLineDetail",
					"SalesItemLineDetail": {
						"ItemRef": {"value": "10", "name": "Pest Control"},
						"Qty": 1,
						"UnitPrice": 35,
						"TaxCodeRef": {"value": "NON"}
					}
				},
				{
					"Amount": 35.0,
					"DetailType": "SubTotalLineDetail",
					"SubTotalLineDetail": {}
				},
				{
					"Amount": 3.5,
					"DetailType": "DiscountLineDetail",
					"DiscountLineDetail": {
						"DiscountAccountRef": {"value": "86", "name": "Discounts given"},
						"PercentBased": true,
						"DiscountPercent": 10
					}
				}
			],
			"TxnTaxDetail": {
				"TotalTax": 0
			}
		}
	}`

	var envelope struct {
		Estimate Estimate `json:"Estimate"`
	}
	require.NoError(t, json.Unmarshal([]byte(fixture), &envelope))

	est := envelope.Estimate
	assert.Equal(t, "177", est.ID())
	assert.Equal(t, "0", est.SyncToken())
	assert.Equal(t, "QBO", est.CDCDomain())
	assert.Equal(t, "Active", est.CDCStatus())

	require.NotNil(t, est.MetaData)
	require.NotNil(t, est.MetaData.CreateTime)

	require.Len(t, est.CustomField, 1)
	assert.Equal(t, "Crew #", est.CustomField[0].Name)

	require.NotNil(t, est.CustomerRef)
	assert.Equal(t, "3", est.CustomerRef.Value)

	require.NotNil(t, est.BillEmail)
	assert.Equal(t, "Cool_Cars@intuit.com", est.BillEmail.Address)

	require.NotNil(t, est.ProjectRef)
	assert.Equal(t, "39298034", est.ProjectRef.Value)

	require.NotNil(t, est.CurrencyRef)
	assert.Equal(t, "USD", est.CurrencyRef.Value)

	require.NotNil(t, est.TxnDate)
	assert.Equal(t, "2015-03-26", est.TxnDate.String())

	require.NotNil(t, est.ExpirationDate)
	assert.Equal(t, "2015-04-26", est.ExpirationDate.String())

	require.NotNil(t, est.AcceptedDate)
	assert.Equal(t, "2015-03-27", est.AcceptedDate.String())

	assert.Equal(t, "1001", est.DocNumber)
	assert.Equal(t, "internal note", est.PrivateNote)
	assert.Equal(t, EstimatePending, est.TxnStatus)
	assert.Equal(t, NeedToPrintStatus, est.PrintStatus)
	assert.Equal(t, EmailNotSetStatus, est.EmailStatus)
	assert.Equal(t, "John Doe", est.AcceptedBy)
	require.NotNil(t, est.ApplyTaxAfterDiscount)
	assert.False(t, *est.ApplyTaxAfterDiscount)

	assert.Equal(t, "31.5", est.TotalAmt.String())

	require.NotNil(t, est.CustomerMemo)
	assert.Equal(t, "Thank you for your business and have a great day!", est.CustomerMemo.Value)

	require.NotNil(t, est.BillAddr)
	assert.Equal(t, "65 Ocean Dr.", est.BillAddr.Line1)
	assert.Equal(t, "Half Moon Bay", est.BillAddr.City)

	require.NotNil(t, est.ShipAddr)
	assert.Equal(t, "65 Ocean Dr.", est.ShipAddr.Line1)

	require.Len(t, est.LinkedTxn, 1)
	assert.Equal(t, "95", est.LinkedTxn[0].TxnID)
	assert.Equal(t, "Invoice", est.LinkedTxn[0].TxnType)

	require.Len(t, est.Line, 3)

	// SalesItemLine
	line0 := est.Line[0]
	assert.Equal(t, SalesItemLineDetailType, line0.DetailType)
	assert.Equal(t, "Pest Control Services", line0.Description)
	assert.Equal(t, "35.0", line0.Amount.String())
	require.NotNil(t, line0.SalesItemLineDetail)
	assert.Equal(t, "10", line0.SalesItemLineDetail.ItemRef.Value)
	assert.Equal(t, "1", line0.SalesItemLineDetail.Qty.String())

	// SubTotalLine
	line1 := est.Line[1]
	assert.Equal(t, SubTotalLineDetailType, line1.DetailType)
	require.NotNil(t, line1.SubTotalLineDetail)

	// DiscountLine
	line2 := est.Line[2]
	assert.Equal(t, DiscountLineDetailType, line2.DetailType)
	require.NotNil(t, line2.DiscountLineDetail)
	assert.Equal(t, "86", line2.DiscountLineDetail.DiscountAccountRef.Value)
	require.NotNil(t, line2.DiscountLineDetail.PercentBased)
	assert.True(t, *line2.DiscountLineDetail.PercentBased)
	assert.Equal(t, "10", line2.DiscountLineDetail.DiscountPercent.String())

	// TxnTaxDetail
	require.NotNil(t, est.TxnTaxDetail)
	assert.Equal(t, "0", est.TxnTaxDetail.TotalTax.String())
}

func TestEstimateRequestsMarshalWritablePayloads(t *testing.T) {
	create := Estimate{
		CustomerRef: &ReferenceType{Value: "3"},
		Line: []EstimateLine{{
			Amount:     json.Number("35"),
			DetailType: SalesItemLineDetailType,
			SalesItemLineDetail: &SalesItemLineDetail{
				ItemRef: &ReferenceType{Value: "10"},
			},
		}},
	}

	body, err := json.Marshal(create)
	require.NoError(t, err)
	assert.Contains(t, string(body), `"CustomerRef"`)
	assert.Contains(t, string(body), `"SalesItemLineDetail"`)
	assert.NotContains(t, string(body), `"sparse"`)
	assert.NotContains(t, string(body), `"TotalAmt"`)
}

func TestEstimateServiceUpdateSparseInjectsSparseFlag(t *testing.T) {
	var capturedBody []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/company/123/estimate", r.URL.Path)
		var err error
		capturedBody, err = io.ReadAll(r.Body)
		require.NoError(t, err)
		w.Header().Set("Content-Type", "application/json")
		_, err = io.WriteString(w, `{"Estimate":{"Id":"41","SyncToken":"4"}}`)
		require.NoError(t, err)
	}))
	defer server.Close()

	client, err := NewClient(WithHTTPClient(server.Client()), WithBaseURL(server.URL+"/v3/company"))
	require.NoError(t, err)

	estimate := Estimate{
		IDValue:        IDValue{Id: "41"},
		SyncTokenValue: SyncTokenValue{SyncTokenValue: "3"},
		CustomerMemo:   &MemoRef{Value: "An updated memo."},
	}

	_, err = client.Company("123", oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test-token"})).Estimates().UpdateSparse(context.Background(), estimate)
	require.NoError(t, err)

	assert.Contains(t, string(capturedBody), `"Id":"41"`)
	assert.Contains(t, string(capturedBody), `"SyncToken":"3"`)
	assert.Contains(t, string(capturedBody), `"sparse":true`)
	assert.Contains(t, string(capturedBody), `"CustomerMemo"`)
}

func TestEstimateServiceSendUsesCorrectEndpoint(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/company/123/estimate/177/send", r.URL.Path)
		assert.Equal(t, "cool_cars@intuit.com", r.URL.Query().Get("sendTo"))
		w.Header().Set("Content-Type", "application/json")
		_, err := io.WriteString(w, `{"Estimate":{"Id":"177","SyncToken":"1","EmailStatus":"EmailSent"}}`)
		require.NoError(t, err)
	}))
	defer server.Close()

	client, err := NewClient(WithHTTPClient(server.Client()), WithBaseURL(server.URL+"/v3/company"))
	require.NoError(t, err)

	err = client.Company("123", oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test-token"})).Estimates().Send(context.Background(), "177", "cool_cars@intuit.com")
	require.NoError(t, err)
}

func TestEstimateCDCExtraction(t *testing.T) {
	changedSince := time.Date(2026, time.March, 10, 12, 0, 0, 0, time.UTC)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/company/123/cdc", r.URL.Path)
		assert.Equal(t, string(EntityEstimate), r.URL.Query().Get("entities"))
		assert.Equal(t, changedSince.Format(time.RFC3339), r.URL.Query().Get("changedSince"))
		w.Header().Set("Content-Type", "application/json")
		_, err := io.WriteString(w, `{"CDCResponse":[{"QueryResponse":[{"Estimate":[{"Id":"177","SyncToken":"1","domain":"QBO","status":"Deleted"}],"startPosition":1,"maxResults":1}]}],"time":"2026-03-10T12:00:00Z"}`)
		require.NoError(t, err)
	}))
	defer server.Close()

	client, err := NewClient(WithHTTPClient(server.Client()), WithBaseURL(server.URL+"/v3/company"))
	require.NoError(t, err)

	ests, err := client.Company("123", oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test-token"})).Estimates().ChangeDataCapture(context.Background(), changedSince)
	require.NoError(t, err)
	require.Len(t, ests, 1)
	assert.Equal(t, "177", ests[0].ID())
	assert.Equal(t, "Deleted", ests[0].CDCStatus())
}

// Compile-time interface compliance checks for Estimate and EstimateService.
var (
	// Estimate entity type compliance.
	_ Entity    = Estimate{}
	_ CDCEntity = Estimate{}

	// EstimateService operation compliance.
	_ Getter[Estimate]          = (*EstimateService)(nil)
	_ Querier[Estimate]         = (*EstimateService)(nil)
	_ Lister[Estimate]          = (*EstimateService)(nil)
	_ Pager[Estimate]           = (*EstimateService)(nil)
	_ Creator[Estimate]         = (*EstimateService)(nil)
	_ Updater[Estimate]         = (*EstimateService)(nil)
	_ SparseUpdater[Estimate]   = (*EstimateService)(nil)
	_ Deleter[Estimate]         = (*EstimateService)(nil)
	_ Sender                    = (*EstimateService)(nil)
	_ PDFGetter                 = (*EstimateService)(nil)
	_ PDFStreamer               = (*EstimateService)(nil)
	_ EntityCDCReader[Estimate] = (*EstimateService)(nil)
)
