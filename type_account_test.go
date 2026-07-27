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

func TestAccountServiceGet(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/company/123/account/1", r.URL.Path)
		assert.Equal(t, http.MethodGet, r.Method)
		w.Header().Set("Content-Type", "application/json")
		_, err := io.WriteString(w, `{"Account":{"Id":"1","SyncToken":"0","domain":"QBO","status":"Deleted","Name":"Checking","AccountType":"Bank","AccountSubType":"Checking","Classification":"Asset","CurrentBalance":"1500.00","Active":true}}`)
		require.NoError(t, err)
	}))
	defer server.Close()

	client, err := NewClient(WithHTTPClient(server.Client()), WithBaseURL(server.URL+"/v3/company"))
	require.NoError(t, err)

	account, err := client.Company("123", oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test-token"})).Accounts().Get(context.Background(), "1")
	require.NoError(t, err)
	assert.Equal(t, "1", account.ID())
	assert.Equal(t, EntityAccount, account.EntityName())
	assert.Equal(t, "Deleted", account.CDCStatus())
	assert.Equal(t, "QBO", account.CDCDomain())
	assert.Equal(t, "Checking", account.Name)
	assert.Equal(t, AccountTypeBank, account.AccountType)
}

func TestAccountServiceQueryUsesQueryOptions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/company/123/query", r.URL.Path)
		assert.Equal(t, "SELECT * FROM Account STARTPOSITION 1 MAXRESULTS 10", r.URL.Query().Get("query"))
		w.Header().Set("Content-Type", "application/json")
		_, err := io.WriteString(w, `{"QueryResponse":{"Account":[{"Id":"1","SyncToken":"0","Name":"Checking"}],"startPosition":1,"maxResults":10}}`)
		require.NoError(t, err)
	}))
	defer server.Close()

	client, err := NewClient(WithHTTPClient(server.Client()), WithBaseURL(server.URL+"/v3/company"))
	require.NoError(t, err)

	accounts, err := client.Company("123", oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test-token"})).Accounts().ListPage(context.Background(), 1, 10)
	require.NoError(t, err)
	require.Len(t, accounts, 1)
	assert.Equal(t, "1", accounts[0].ID())
}

func TestAccountServiceCreate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/company/123/account", r.URL.Path)
		assert.Equal(t, http.MethodPost, r.Method)

		var payload Account
		require.NoError(t, json.NewDecoder(r.Body).Decode(&payload))
		assert.Equal(t, "New Savings", payload.Name)
		assert.Equal(t, AccountTypeBank, payload.AccountType)

		w.Header().Set("Content-Type", "application/json")
		_, err := io.WriteString(w, `{"Account":{"Id":"42","SyncToken":"0","Name":"New Savings","AccountType":"Bank","AccountSubType":"Savings","Classification":"Asset","CurrentBalance":"0.00","Active":true}}`)
		require.NoError(t, err)
	}))
	defer server.Close()

	client, err := NewClient(WithHTTPClient(server.Client()), WithBaseURL(server.URL+"/v3/company"))
	require.NoError(t, err)

	account, err := client.Company("123", oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test-token"})).Accounts().Create(
		context.Background(),
		Account{Name: "New Savings", AccountType: AccountTypeBank},
	)
	require.NoError(t, err)
	assert.Equal(t, "42", account.ID())
	assert.Equal(t, "New Savings", account.Name)
	assert.Equal(t, AccountClassificationAsset, account.Classification)
}

func TestAccountServiceUpdate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/company/123/account", r.URL.Path)
		assert.Equal(t, http.MethodPost, r.Method)

		var payload Account
		require.NoError(t, json.NewDecoder(r.Body).Decode(&payload))
		assert.Equal(t, "42", payload.ID())
		assert.Equal(t, "1", payload.SyncToken())
		assert.Equal(t, "Updated Checking", payload.Name)

		w.Header().Set("Content-Type", "application/json")
		_, err := io.WriteString(w, `{"Account":{"Id":"42","SyncToken":"2","Name":"Updated Checking","AccountType":"Bank","Classification":"Asset","CurrentBalance":"2500.00","Active":true}}`)
		require.NoError(t, err)
	}))
	defer server.Close()

	client, err := NewClient(WithHTTPClient(server.Client()), WithBaseURL(server.URL+"/v3/company"))
	require.NoError(t, err)

	account, err := client.Company("123", oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test-token"})).Accounts().Update(
		context.Background(),
		Account{
			IDValue:        IDValue{Id: "42"},
			SyncTokenValue: SyncTokenValue{SyncTokenValue: "1"},
			Name:           "Updated Checking",
			AccountType:    AccountTypeBank,
		},
	)
	require.NoError(t, err)
	assert.Equal(t, "42", account.ID())
	assert.Equal(t, "2", account.SyncToken())
	assert.Equal(t, "Updated Checking", account.Name)
}

func TestAccountServiceChangeDataCapture(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/company/123/cdc", r.URL.Path)
		assert.Equal(t, "Account", r.URL.Query().Get("entities"))
		assert.NotEmpty(t, r.URL.Query().Get("changedSince"))

		w.Header().Set("Content-Type", "application/json")
		_, err := io.WriteString(w, `{"CDCResponse":[{"QueryResponse":[{"Account":[{"Id":"1","SyncToken":"0","domain":"QBO","status":"Deleted","Name":"Old Savings"}],"startPosition":1,"maxResults":1}]}],"time":"2026-03-15T10:00:00-07:00"}`)
		require.NoError(t, err)
	}))
	defer server.Close()

	client, err := NewClient(WithHTTPClient(server.Client()), WithBaseURL(server.URL+"/v3/company"))
	require.NoError(t, err)

	accounts, err := client.Company("123", oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test-token"})).
		Accounts().ChangeDataCapture(context.Background(), time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	require.NoError(t, err)
	require.Len(t, accounts, 1)
	assert.Equal(t, "1", accounts[0].ID())
	assert.Equal(t, "Deleted", accounts[0].CDCStatus())
	assert.Equal(t, "Old Savings", accounts[0].Name)
}

func TestAccountJSONDecodes(t *testing.T) {
	var envelope accountEnvelope
	err := json.Unmarshal([]byte(`{
		"Account": {
			"Id": "35",
			"SyncToken": "2",
			"domain": "QBO",
			"status": "Deleted",
			"MetaData": {
				"CreateTime": "2026-01-01T00:00:00-07:00",
				"LastUpdatedTime": "2026-03-01T00:00:00-07:00"
			},
			"Name": "Accounts Payable (A/P)",
			"AcctNum": "2000",
			"AccountType": "Accounts Payable",
			"AccountSubType": "AccountsPayable",
			"Classification": "Liability",
			"FullyQualifiedName": "Accounts Payable (A/P)",
			"SubAccount": false,
			"Active": true,
			"CurrentBalance": "-1602.67",
			"CurrentBalanceWithSubAccounts": "-1602.67",
			"CurrencyRef": {"value": "USD", "name": "United States Dollar"},
			"TaxCodeRef": {"value": "NON"},
			"Description": "Default accounts payable"
		}
	}`), &envelope)
	require.NoError(t, err)

	a := envelope.Account
	assert.Equal(t, "35", a.ID())
	assert.Equal(t, "2", a.SyncToken())
	assert.Equal(t, "Deleted", a.CDCStatus())
	assert.Equal(t, "Accounts Payable (A/P)", a.Name)
	assert.Equal(t, "2000", a.AcctNum)
	assert.Equal(t, AccountTypeAccountsPayable, a.AccountType)
	assert.Equal(t, AccountSubTypeAccountsPayable, a.AccountSubType)
	assert.Equal(t, AccountClassificationLiability, a.Classification)
	assert.Equal(t, "Accounts Payable (A/P)", a.FullyQualifiedName)
	require.NotNil(t, a.SubAccount)
	assert.False(t, *a.SubAccount)
	require.NotNil(t, a.Active)
	assert.True(t, *a.Active)
	assert.Equal(t, json.Number("-1602.67"), a.CurrentBalance)
	assert.Equal(t, json.Number("-1602.67"), a.CurrentBalanceWithSubAccounts)
	require.NotNil(t, a.CurrencyRef)
	assert.Equal(t, "USD", a.CurrencyRef.Value)
	require.NotNil(t, a.TaxCodeRef)
	assert.Equal(t, "NON", a.TaxCodeRef.Value)
	assert.Equal(t, "Default accounts payable", a.Description)
	require.NotNil(t, a.MetaData)
	assert.False(t, a.MetaData.CreateTime.IsZero())
}

// Compile-time interface compliance checks for Account and AccountService.
var (
	// Account entity type compliance.
	_ Entity    = Account{}
	_ CDCEntity = Account{}

	// AccountService operation compliance.
	_ Getter[Account]          = (*AccountService)(nil)
	_ Querier[Account]         = (*AccountService)(nil)
	_ Lister[Account]          = (*AccountService)(nil)
	_ Pager[Account]           = (*AccountService)(nil)
	_ Creator[Account]         = (*AccountService)(nil)
	_ Updater[Account]         = (*AccountService)(nil)
	_ SparseUpdater[Account]   = (*AccountService)(nil)
	_ EntityCDCReader[Account] = (*AccountService)(nil)
)

func TestAccountServiceUpdateSparseInjectsSparseFlag(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/company/123/account", r.URL.Path)
		var payload map[string]interface{}
		require.NoError(t, json.NewDecoder(r.Body).Decode(&payload))
		assert.Equal(t, true, payload["sparse"])
		w.Header().Set("Content-Type", "application/json")
		_, err := io.WriteString(w, `{"Account":{"Id":"1","SyncToken":"2"}}`)
		require.NoError(t, err)
	}))
	defer server.Close()

	client, err := NewClient(WithHTTPClient(server.Client()), WithBaseURL(server.URL+"/v3/company"))
	require.NoError(t, err)

	account := Account{}
	account.Id = "1"
	account.SyncTokenValue = SyncTokenValue{"1"}

	_, err = client.Company("123", oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test-token"})).Accounts().UpdateSparse(context.Background(), account)
	require.NoError(t, err)
}
