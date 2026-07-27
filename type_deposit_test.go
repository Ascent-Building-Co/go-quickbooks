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

func TestDepositServiceGet(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/company/123/deposit/148", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_, err := io.WriteString(w, `{"Deposit":{"Id":"148","SyncToken":"0","domain":"QBO","status":"Active","DepositToAccountRef":{"value":"35","name":"Checking"},"TotalAmt":1675.52}}`)
		require.NoError(t, err)
	}))
	defer server.Close()

	client, err := NewClient(WithHTTPClient(server.Client()), WithBaseURL(server.URL+"/v3/company"))
	require.NoError(t, err)

	dep, err := client.Company("123", oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test-token"})).Deposits().Get(context.Background(), "148")
	require.NoError(t, err)
	assert.Equal(t, "148", dep.ID())
	assert.Equal(t, EntityDeposit, dep.EntityName())
	assert.Equal(t, "Active", dep.CDCStatus())
	assert.Equal(t, "QBO", dep.CDCDomain())
	require.NotNil(t, dep.DepositToAccountRef)
	assert.Equal(t, "35", dep.DepositToAccountRef.Value)
}

func TestDepositServiceQueryUsesQueryOptions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/company/123/query", r.URL.Path)
		assert.Equal(t, "SELECT * FROM Deposit STARTPOSITION 11 MAXRESULTS 10", r.URL.Query().Get("query"))
		w.Header().Set("Content-Type", "application/json")
		_, err := io.WriteString(w, `{"QueryResponse":{"Deposit":[{"Id":"148","SyncToken":"0"},{"Id":"149","SyncToken":"1"}],"startPosition":6,"maxResults":10}}`)
		require.NoError(t, err)
	}))
	defer server.Close()

	client, err := NewClient(WithHTTPClient(server.Client()), WithBaseURL(server.URL+"/v3/company"))
	require.NoError(t, err)

	deps, err := client.Company("123", oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test-token"})).Deposits().ListPage(context.Background(), 2, 10)
	require.NoError(t, err)
	require.Len(t, deps, 2)
	assert.Equal(t, "148", deps[0].ID())
	assert.Equal(t, "149", deps[1].ID())
}

func TestDepositJSONDecodesV75Fields(t *testing.T) {
	const fixture = `{
		"Deposit": {
			"Id": "148",
			"SyncToken": "0",
			"domain": "QBO",
			"status": "Active",
			"MetaData": {
				"CreateTime": "2014-12-22T12:46:52-08:00",
				"LastUpdatedTime": "2014-12-22T12:46:52-08:00"
			},
			"DepositToAccountRef": {"value": "35", "name": "Checking"},
			"CurrencyRef": {"value": "USD", "name": "United States Dollar"},
			"TxnDate": "2014-12-22",
			"PrivateNote": "internal note",
			"DepartmentRef": {"value": "1", "name": "Garden Services"},
			"TotalAmt": 1675.52,
			"CashBack": {
				"AccountRef": {"value": "36", "name": "Savings"},
				"Amount": 50.26,
				"Memo": "Test"
			},
			"Line": [
				{
					"Id": "1",
					"LineNum": 1,
					"Amount": 1675,
					"LinkedTxn": [
						{"TxnId": "120", "TxnType": "Payment", "TxnLineId": "0"}
					]
				},
				{
					"Id": "2",
					"LineNum": 2,
					"Description": "Direct deposit",
					"Amount": 200.0,
					"DetailType": "DepositLineDetail",
					"ProjectRef": {"value": "42991284"},
					"DepositLineDetail": {
						"AccountRef": {"value": "87", "name": "Unapplied Cash Payment Income"},
						"ClassRef": {"value": "3"},
						"PaymentMethodRef": {"value": "4", "name": "Check"},
						"CheckNum": "1042",
						"TxnType": "Deposit"
					}
				}
			],
			"TxnTaxDetail": {
				"TotalTax": 0
			}
		}
	}`

	var envelope struct {
		Deposit Deposit `json:"Deposit"`
	}
	require.NoError(t, json.Unmarshal([]byte(fixture), &envelope))

	dep := envelope.Deposit
	assert.Equal(t, "148", dep.ID())
	assert.Equal(t, "0", dep.SyncToken())
	assert.Equal(t, "QBO", dep.CDCDomain())
	assert.Equal(t, "Active", dep.CDCStatus())

	require.NotNil(t, dep.MetaData)
	require.NotNil(t, dep.MetaData.CreateTime)

	require.NotNil(t, dep.DepositToAccountRef)
	assert.Equal(t, "35", dep.DepositToAccountRef.Value)

	require.NotNil(t, dep.CurrencyRef)
	assert.Equal(t, "USD", dep.CurrencyRef.Value)

	require.NotNil(t, dep.TxnDate)
	assert.Equal(t, "2014-12-22", dep.TxnDate.String())

	assert.Equal(t, "internal note", dep.PrivateNote)

	require.NotNil(t, dep.DepartmentRef)
	assert.Equal(t, "1", dep.DepartmentRef.Value)

	assert.Equal(t, "1675.52", dep.TotalAmt.String())

	// CashBack
	require.NotNil(t, dep.CashBack)
	assert.Equal(t, "36", dep.CashBack.AccountRef.Value)
	assert.Equal(t, "50.26", dep.CashBack.Amount.String())
	assert.Equal(t, "Test", dep.CashBack.Memo)

	require.Len(t, dep.Line, 2)

	// LinkedTxn line
	line0 := dep.Line[0]
	assert.Equal(t, "1675", line0.Amount.String())
	require.Len(t, line0.LinkedTxn, 1)
	assert.Equal(t, "120", line0.LinkedTxn[0].TxnID)
	assert.Equal(t, "Payment", line0.LinkedTxn[0].TxnType)
	assert.Equal(t, "0", line0.LinkedTxn[0].TxnLineID)

	// DepositLineDetail line
	line1 := dep.Line[1]
	assert.Equal(t, DepositLineDetailType, line1.DetailType)
	assert.Equal(t, "Direct deposit", line1.Description)
	assert.Equal(t, "200.0", line1.Amount.String())
	require.NotNil(t, line1.ProjectRef)
	assert.Equal(t, "42991284", line1.ProjectRef.Value)
	require.NotNil(t, line1.DepositLineDetail)
	assert.Equal(t, "87", line1.DepositLineDetail.AccountRef.Value)
	assert.Equal(t, "3", line1.DepositLineDetail.ClassRef.Value)
	assert.Equal(t, "4", line1.DepositLineDetail.PaymentMethodRef.Value)
	assert.Equal(t, "1042", line1.DepositLineDetail.CheckNum)
	assert.Equal(t, "Deposit", line1.DepositLineDetail.TxnType)

	// TxnTaxDetail
	require.NotNil(t, dep.TxnTaxDetail)
	assert.Equal(t, "0", dep.TxnTaxDetail.TotalTax.String())
}

func TestDepositRequestsMarshalWritablePayloads(t *testing.T) {
	create := Deposit{
		DepositToAccountRef: &ReferenceType{Value: "35"},
		Line: []DepositLine{{
			Amount:     json.Number("200"),
			DetailType: DepositLineDetailType,
			DepositLineDetail: &DepositLineDetail{
				AccountRef: &ReferenceType{Value: "87"},
			},
		}},
	}

	body, err := json.Marshal(create)
	require.NoError(t, err)
	assert.Contains(t, string(body), `"DepositToAccountRef"`)
	assert.Contains(t, string(body), `"DepositLineDetail"`)
	assert.NotContains(t, string(body), `"sparse"`)
	assert.NotContains(t, string(body), `"TotalAmt"`)
}

func TestDepositServiceUpdateSparseInjectsSparseFlag(t *testing.T) {
	var capturedBody []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/company/123/deposit", r.URL.Path)
		var err error
		capturedBody, err = io.ReadAll(r.Body)
		require.NoError(t, err)
		w.Header().Set("Content-Type", "application/json")
		_, err = io.WriteString(w, `{"Deposit":{"Id":"148","SyncToken":"1"}}`)
		require.NoError(t, err)
	}))
	defer server.Close()

	client, err := NewClient(WithHTTPClient(server.Client()), WithBaseURL(server.URL+"/v3/company"))
	require.NoError(t, err)

	deposit := Deposit{
		IDValue:             IDValue{Id: "148"},
		SyncTokenValue:      SyncTokenValue{SyncTokenValue: "0"},
		DepositToAccountRef: &ReferenceType{Value: "35"},
		TxnDate:             &Date{Time: time.Date(2014, 12, 2, 0, 0, 0, 0, time.UTC)},
	}

	_, err = client.Company("123", oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test-token"})).Deposits().UpdateSparse(context.Background(), deposit)
	require.NoError(t, err)

	assert.Contains(t, string(capturedBody), `"Id":"148"`)
	assert.Contains(t, string(capturedBody), `"SyncToken":"0"`)
	assert.Contains(t, string(capturedBody), `"sparse":true`)
	assert.Contains(t, string(capturedBody), `"DepositToAccountRef"`)
}

func TestDepositCDCExtraction(t *testing.T) {
	changedSince := time.Date(2026, time.March, 10, 12, 0, 0, 0, time.UTC)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/company/123/cdc", r.URL.Path)
		assert.Equal(t, string(EntityDeposit), r.URL.Query().Get("entities"))
		assert.Equal(t, changedSince.Format(time.RFC3339), r.URL.Query().Get("changedSince"))
		w.Header().Set("Content-Type", "application/json")
		_, err := io.WriteString(w, `{"CDCResponse":[{"QueryResponse":[{"Deposit":[{"Id":"148","SyncToken":"1","domain":"QBO","status":"Deleted"}],"startPosition":1,"maxResults":1}]}],"time":"2026-03-10T12:00:00Z"}`)
		require.NoError(t, err)
	}))
	defer server.Close()

	client, err := NewClient(WithHTTPClient(server.Client()), WithBaseURL(server.URL+"/v3/company"))
	require.NoError(t, err)

	deps, err := client.Company("123", oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test-token"})).Deposits().ChangeDataCapture(context.Background(), changedSince)
	require.NoError(t, err)
	require.Len(t, deps, 1)
	assert.Equal(t, "148", deps[0].ID())
	assert.Equal(t, "Deleted", deps[0].CDCStatus())
}

// Compile-time interface compliance checks for Deposit and DepositService.
var (
	// Deposit entity type compliance.
	_ Entity    = Deposit{}
	_ CDCEntity = Deposit{}

	// DepositService operation compliance.
	_ Getter[Deposit]          = (*DepositService)(nil)
	_ Querier[Deposit]         = (*DepositService)(nil)
	_ Lister[Deposit]          = (*DepositService)(nil)
	_ Pager[Deposit]           = (*DepositService)(nil)
	_ Creator[Deposit]         = (*DepositService)(nil)
	_ Updater[Deposit]         = (*DepositService)(nil)
	_ SparseUpdater[Deposit]   = (*DepositService)(nil)
	_ Deleter[Deposit]         = (*DepositService)(nil)
	_ EntityCDCReader[Deposit] = (*DepositService)(nil)
)
