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

func TestCreditMemoServiceGet(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/company/123/creditmemo/73", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_, err := io.WriteString(w, `{"CreditMemo":{"Id":"73","SyncToken":"3","domain":"QBO","status":"Active","CustomerRef":{"value":"1","name":"Amy's Bird Sanctuary"},"TotalAmt":100.0}}`)
		require.NoError(t, err)
	}))
	defer server.Close()

	client, err := NewClient(WithHTTPClient(server.Client()), WithBaseURL(server.URL+"/v3/company"))
	require.NoError(t, err)

	cm, err := client.Company("123", oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test-token"})).CreditMemos().Get(context.Background(), "73")
	require.NoError(t, err)
	assert.Equal(t, "73", cm.ID())
	assert.Equal(t, EntityCreditMemo, cm.EntityName())
	assert.Equal(t, "Active", cm.CDCStatus())
	assert.Equal(t, "QBO", cm.CDCDomain())
	require.NotNil(t, cm.CustomerRef)
	assert.Equal(t, "1", cm.CustomerRef.Value)
}

func TestCreditMemoServiceQueryUsesQueryOptions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/company/123/query", r.URL.Path)
		assert.Equal(t, "SELECT * FROM CreditMemo STARTPOSITION 11 MAXRESULTS 10", r.URL.Query().Get("query"))
		w.Header().Set("Content-Type", "application/json")
		_, err := io.WriteString(w, `{"QueryResponse":{"CreditMemo":[{"Id":"73","SyncToken":"3"},{"Id":"74","SyncToken":"0"}],"startPosition":6,"maxResults":10}}`)
		require.NoError(t, err)
	}))
	defer server.Close()

	client, err := NewClient(WithHTTPClient(server.Client()), WithBaseURL(server.URL+"/v3/company"))
	require.NoError(t, err)

	cms, err := client.Company("123", oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test-token"})).CreditMemos().ListPage(context.Background(), 2, 10)
	require.NoError(t, err)
	require.Len(t, cms, 2)
	assert.Equal(t, "73", cms[0].ID())
	assert.Equal(t, "74", cms[1].ID())
}

func TestCreditMemoJSONDecodesV75Fields(t *testing.T) {
	const fixture = `{
		"CreditMemo": {
			"Id": "73",
			"SyncToken": "3",
			"domain": "QBO",
			"status": "Active",
			"MetaData": {
				"CreateTime": "2014-09-18T12:51:27-07:00",
				"LastUpdatedTime": "2015-07-01T09:16:28-07:00"
			},
			"CustomField": [
				{"DefinitionId": "1", "Type": "StringType", "Name": "Crew #"}
			],
			"CustomerRef": {"value": "1", "name": "Amy's Bird Sanctuary"},
			"CurrencyRef": {"value": "USD", "name": "United States Dollar"},
			"ProjectRef": {"value": "39298034"},
			"BillEmail": {"Address": "Birds@Intuit.com"},
			"TxnDate": "2014-09-02",
			"DocNumber": "1026",
			"PrivateNote": "internal note",
			"PrintStatus": "NeedToPrint",
			"EmailStatus": "NotSet",
			"ApplyTaxAfterDiscount": false,
			"TotalAmt": 100.0,
			"Balance": 0,
			"RemainingCredit": 0,
			"CustomerMemo": {"value": "Updated customer memo."},
			"BillAddr": {
				"Line1": "Amy Lauterbach",
				"Line2": "Amy's Bird Sanctuary",
				"Line3": "4581 Finch St.",
				"Line4": "Bayshore, CA  94326",
				"Id": "79"
			},
			"ShipAddr": {
				"Line1": "4581 Finch St.",
				"City": "Bayshore",
				"CountrySubDivisionCode": "CA",
				"PostalCode": "94326",
				"Id": "108"
			},
			"Line": [
				{
					"Id": "1",
					"LineNum": 1,
					"Description": "Pest Control Services",
					"Amount": 100.0,
					"DetailType": "SalesItemLineDetail",
					"SalesItemLineDetail": {
						"ItemRef": {"value": "10", "name": "Pest Control"},
						"Qty": 1,
						"UnitPrice": 100,
						"TaxCodeRef": {"value": "NON"}
					}
				},
				{
					"Amount": 100.0,
					"DetailType": "SubTotalLineDetail",
					"SubTotalLineDetail": {}
				},
				{
					"Id": "3",
					"LineNum": 3,
					"Description": "Group line",
					"DetailType": "GroupLineDetail",
					"GroupLineDetail": {
						"GroupItemRef": {"value": "20", "name": "Pkg"},
						"Quantity": 2,
						"Line": [
							{
								"Id": "4",
								"Amount": 50.0,
								"DetailType": "SalesItemLineDetail",
								"SalesItemLineDetail": {
									"ItemRef": {"value": "5"}
								}
							}
						]
					}
				}
			],
			"TxnTaxDetail": {
				"TotalTax": 0
			}
		}
	}`

	var envelope struct {
		CreditMemo CreditMemo `json:"CreditMemo"`
	}
	require.NoError(t, json.Unmarshal([]byte(fixture), &envelope))

	cm := envelope.CreditMemo
	assert.Equal(t, "73", cm.ID())
	assert.Equal(t, "3", cm.SyncToken())
	assert.Equal(t, "QBO", cm.CDCDomain())
	assert.Equal(t, "Active", cm.CDCStatus())

	require.NotNil(t, cm.MetaData)
	require.NotNil(t, cm.MetaData.CreateTime)

	require.Len(t, cm.CustomField, 1)
	assert.Equal(t, "Crew #", cm.CustomField[0].Name)

	require.NotNil(t, cm.CustomerRef)
	assert.Equal(t, "1", cm.CustomerRef.Value)

	require.NotNil(t, cm.CurrencyRef)
	assert.Equal(t, "USD", cm.CurrencyRef.Value)

	require.NotNil(t, cm.ProjectRef)
	assert.Equal(t, "39298034", cm.ProjectRef.Value)

	require.NotNil(t, cm.BillEmail)
	assert.Equal(t, "Birds@Intuit.com", cm.BillEmail.Address)

	require.NotNil(t, cm.TxnDate)
	assert.Equal(t, "2014-09-02", cm.TxnDate.String())

	assert.Equal(t, "1026", cm.DocNumber)
	assert.Equal(t, "internal note", cm.PrivateNote)
	assert.Equal(t, NeedToPrintStatus, cm.PrintStatus)
	assert.Equal(t, EmailNotSetStatus, cm.EmailStatus)
	require.NotNil(t, cm.ApplyTaxAfterDiscount)
	assert.False(t, *cm.ApplyTaxAfterDiscount)

	assert.Equal(t, "100.0", cm.TotalAmt.String())
	assert.Equal(t, "0", cm.Balance.String())
	assert.Equal(t, "0", cm.RemainingCredit.String())

	require.NotNil(t, cm.CustomerMemo)
	assert.Equal(t, "Updated customer memo.", cm.CustomerMemo.Value)

	require.NotNil(t, cm.BillAddr)
	assert.Equal(t, "Amy Lauterbach", cm.BillAddr.Line1)

	require.NotNil(t, cm.ShipAddr)
	assert.Equal(t, "4581 Finch St.", cm.ShipAddr.Line1)
	assert.Equal(t, "Bayshore", cm.ShipAddr.City)

	require.Len(t, cm.Line, 3)

	// SalesItemLine
	line0 := cm.Line[0]
	assert.Equal(t, SalesItemLineDetailType, line0.DetailType)
	assert.Equal(t, "Pest Control Services", line0.Description)
	assert.Equal(t, "100.0", line0.Amount.String())
	require.NotNil(t, line0.SalesItemLineDetail)
	assert.Equal(t, "10", line0.SalesItemLineDetail.ItemRef.Value)
	assert.Equal(t, "1", line0.SalesItemLineDetail.Qty.String())
	assert.Equal(t, "100", line0.SalesItemLineDetail.UnitPrice.String())

	// SubTotalLine
	line1 := cm.Line[1]
	assert.Equal(t, SubTotalLineDetailType, line1.DetailType)
	require.NotNil(t, line1.SubTotalLineDetail)

	// GroupLine with nested CreditMemoLine
	line2 := cm.Line[2]
	assert.Equal(t, LineDetailType("GroupLineDetail"), line2.DetailType)
	require.NotNil(t, line2.GroupLineDetail)
	assert.Equal(t, "20", line2.GroupLineDetail.GroupItemRef.Value)
	assert.Equal(t, "2", line2.GroupLineDetail.Quantity.String())
	require.Len(t, line2.GroupLineDetail.Line, 1)
	assert.Equal(t, "5", line2.GroupLineDetail.Line[0].SalesItemLineDetail.ItemRef.Value)

	// TxnTaxDetail
	require.NotNil(t, cm.TxnTaxDetail)
	assert.Equal(t, "0", cm.TxnTaxDetail.TotalTax.String())
}

func TestCreditMemoRequestsMarshalWritablePayloads(t *testing.T) {
	create := CreditMemo{
		CustomerRef: &ReferenceType{Value: "1"},
		Line: []CreditMemoLine{{
			Amount:     json.Number("50"),
			DetailType: SalesItemLineDetailType,
			SalesItemLineDetail: &SalesItemLineDetail{
				ItemRef: &ReferenceType{Value: "3"},
			},
		}},
	}

	body, err := json.Marshal(create)
	require.NoError(t, err)
	assert.Contains(t, string(body), `"CustomerRef"`)
	assert.Contains(t, string(body), `"SalesItemLineDetail"`)
	assert.NotContains(t, string(body), `"sparse"`)
	assert.NotContains(t, string(body), `"Balance"`)
}

func TestCreditMemoServiceSendUsesCorrectEndpoint(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/company/123/creditmemo/73/send", r.URL.Path)
		assert.Equal(t, "birds@intuit.com", r.URL.Query().Get("sendTo"))
		w.Header().Set("Content-Type", "application/json")
		_, err := io.WriteString(w, `{"CreditMemo":{"Id":"73","SyncToken":"4","EmailStatus":"EmailSent"}}`)
		require.NoError(t, err)
	}))
	defer server.Close()

	client, err := NewClient(WithHTTPClient(server.Client()), WithBaseURL(server.URL+"/v3/company"))
	require.NoError(t, err)

	err = client.Company("123", oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test-token"})).CreditMemos().Send(context.Background(), "73", "birds@intuit.com")
	require.NoError(t, err)
}

func TestCreditMemoCDCExtraction(t *testing.T) {
	changedSince := time.Date(2026, time.March, 10, 12, 0, 0, 0, time.UTC)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/company/123/cdc", r.URL.Path)
		assert.Equal(t, string(EntityCreditMemo), r.URL.Query().Get("entities"))
		assert.Equal(t, changedSince.Format(time.RFC3339), r.URL.Query().Get("changedSince"))
		w.Header().Set("Content-Type", "application/json")
		_, err := io.WriteString(w, `{"CDCResponse":[{"QueryResponse":[{"CreditMemo":[{"Id":"73","SyncToken":"4","domain":"QBO","status":"Deleted"}],"startPosition":1,"maxResults":1}]}],"time":"2026-03-10T12:00:00Z"}`)
		require.NoError(t, err)
	}))
	defer server.Close()

	client, err := NewClient(WithHTTPClient(server.Client()), WithBaseURL(server.URL+"/v3/company"))
	require.NoError(t, err)

	cms, err := client.Company("123", oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test-token"})).CreditMemos().ChangeDataCapture(context.Background(), changedSince)
	require.NoError(t, err)
	require.Len(t, cms, 1)
	assert.Equal(t, "73", cms[0].ID())
	assert.Equal(t, "Deleted", cms[0].CDCStatus())
}

// Compile-time interface compliance checks for CreditMemo and CreditMemoService.
var (
	// CreditMemo entity type compliance.
	_ Entity    = CreditMemo{}
	_ CDCEntity = CreditMemo{}

	// CreditMemoService operation compliance.
	_ Getter[CreditMemo]          = (*CreditMemoService)(nil)
	_ Querier[CreditMemo]         = (*CreditMemoService)(nil)
	_ Lister[CreditMemo]          = (*CreditMemoService)(nil)
	_ Pager[CreditMemo]           = (*CreditMemoService)(nil)
	_ Creator[CreditMemo]         = (*CreditMemoService)(nil)
	_ Updater[CreditMemo]         = (*CreditMemoService)(nil)
	_ SparseUpdater[CreditMemo]   = (*CreditMemoService)(nil)
	_ Deleter[CreditMemo]         = (*CreditMemoService)(nil)
	_ Sender                      = (*CreditMemoService)(nil)
	_ PDFGetter                   = (*CreditMemoService)(nil)
	_ PDFStreamer                 = (*CreditMemoService)(nil)
	_ EntityCDCReader[CreditMemo] = (*CreditMemoService)(nil)
)

func TestCreditMemoServiceUpdateSparseInjectsSparseFlag(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/company/123/creditmemo", r.URL.Path)
		var payload map[string]interface{}
		require.NoError(t, json.NewDecoder(r.Body).Decode(&payload))
		assert.Equal(t, true, payload["sparse"])
		w.Header().Set("Content-Type", "application/json")
		_, err := io.WriteString(w, `{"CreditMemo":{"Id":"1","SyncToken":"2"}}`)
		require.NoError(t, err)
	}))
	defer server.Close()

	client, err := NewClient(WithHTTPClient(server.Client()), WithBaseURL(server.URL+"/v3/company"))
	require.NoError(t, err)

	cm := CreditMemo{}
	cm.Id = "1"
	cm.SyncTokenValue = SyncTokenValue{"1"}

	_, err = client.Company("123", oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test-token"})).CreditMemos().UpdateSparse(context.Background(), cm)
	require.NoError(t, err)
}
