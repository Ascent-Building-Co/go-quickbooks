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

func TestCustomerServiceGet(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/company/123/customer/2", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_, err := io.WriteString(w, `{"Customer":{"Id":"2","SyncToken":"0","domain":"QBO","status":"Active","DisplayName":"Bill's Windsurf Shop","FullyQualifiedName":"Bill's Windsurf Shop"}}`)
		require.NoError(t, err)
	}))
	defer server.Close()

	client, err := NewClient(WithHTTPClient(server.Client()), WithBaseURL(server.URL+"/v3/company"))
	require.NoError(t, err)

	cust, err := client.Company("123", oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test-token"})).Customers().Get(context.Background(), "2")
	require.NoError(t, err)
	assert.Equal(t, "2", cust.ID())
	assert.Equal(t, EntityCustomer, cust.EntityName())
	assert.Equal(t, "Active", cust.CDCStatus())
	assert.Equal(t, "QBO", cust.CDCDomain())
	assert.Equal(t, "Bill's Windsurf Shop", cust.DisplayName)
	assert.Equal(t, "Bill's Windsurf Shop", cust.FullyQualifiedName)
}

func TestCustomerServiceQueryUsesQueryOptions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/company/123/query", r.URL.Path)
		assert.Equal(t, "SELECT * FROM Customer STARTPOSITION 11 MAXRESULTS 10", r.URL.Query().Get("query"))
		w.Header().Set("Content-Type", "application/json")
		_, err := io.WriteString(w, `{"QueryResponse":{"Customer":[{"Id":"2","SyncToken":"0"},{"Id":"3","SyncToken":"1"}],"startPosition":6,"maxResults":10}}`)
		require.NoError(t, err)
	}))
	defer server.Close()

	client, err := NewClient(WithHTTPClient(server.Client()), WithBaseURL(server.URL+"/v3/company"))
	require.NoError(t, err)

	custs, err := client.Company("123", oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test-token"})).Customers().ListPage(context.Background(), 2, 10)
	require.NoError(t, err)
	require.Len(t, custs, 2)
	assert.Equal(t, "2", custs[0].ID())
	assert.Equal(t, "3", custs[1].ID())
}

func TestCustomerJSONDecodesV75Fields(t *testing.T) {
	const fixture = `{
		"Customer": {
			"Id": "2",
			"SyncToken": "0",
			"domain": "QBO",
			"status": "Active",
			"MetaData": {
				"CreateTime": "2014-09-11T16:49:28-07:00",
				"LastUpdatedTime": "2014-09-18T12:56:01-07:00"
			},
			"DisplayName": "Bill's Windsurf Shop",
			"Title": "Mr",
			"GivenName": "Bill",
			"MiddleName": "A",
			"FamilyName": "Lucchini",
			"Suffix": "Jr",
			"CompanyName": "Bill's Windsurf Shop",
			"PrintOnCheckName": "Bill's Windsurf Shop",
			"FullyQualifiedName": "Bill's Windsurf Shop",
			"Level": 0,
			"Active": true,
			"Taxable": false,
			"Balance": 85.0,
			"BalanceWithJobs": 85.0,
			"Job": false,
			"BillWithParent": false,
			"PreferredDeliveryMethod": "Print",
			"Notes": "Preferred customer",
			"PrimaryPhone": {"FreeFormNumber": "(415) 444-6538"},
			"AlternatePhone": {"FreeFormNumber": "(415) 444-6539"},
			"Mobile": {"FreeFormNumber": "(415) 555-1234"},
			"Fax": {"FreeFormNumber": "(415) 555-5678"},
			"PrimaryEmailAddr": {"Address": "Surf@Intuit.com"},
			"WebAddr": {"URI": "http://billswindsurfshop.com"},
			"BillAddr": {
				"Line1": "12 Ocean Dr.",
				"City": "Half Moon Bay",
				"CountrySubDivisionCode": "CA",
				"PostalCode": "94213",
				"Id": "3"
			},
			"ShipAddr": {
				"Line1": "12 Ocean Dr.",
				"City": "Half Moon Bay",
				"CountrySubDivisionCode": "CA",
				"PostalCode": "94213",
				"Id": "4"
			},
			"CurrencyRef": {"value": "USD", "name": "United States Dollar"},
			"PaymentMethodRef": {"value": "2"},
			"SalesTermRef": {"value": "3"}
		}
	}`

	var envelope struct {
		Customer Customer `json:"Customer"`
	}
	require.NoError(t, json.Unmarshal([]byte(fixture), &envelope))

	cust := envelope.Customer
	assert.Equal(t, "2", cust.ID())
	assert.Equal(t, "0", cust.SyncToken())
	assert.Equal(t, "QBO", cust.CDCDomain())
	assert.Equal(t, "Active", cust.CDCStatus())

	require.NotNil(t, cust.MetaData)
	require.NotNil(t, cust.MetaData.CreateTime)

	assert.Equal(t, "Bill's Windsurf Shop", cust.DisplayName)
	assert.Equal(t, "Mr", cust.Title)
	assert.Equal(t, "Bill", cust.GivenName)
	assert.Equal(t, "A", cust.MiddleName)
	assert.Equal(t, "Lucchini", cust.FamilyName)
	assert.Equal(t, "Jr", cust.Suffix)
	assert.Equal(t, "Bill's Windsurf Shop", cust.CompanyName)
	assert.Equal(t, "Bill's Windsurf Shop", cust.PrintOnCheckName)
	assert.Equal(t, "Bill's Windsurf Shop", cust.FullyQualifiedName)
	assert.Equal(t, 0, cust.Level)

	require.NotNil(t, cust.Active)
	assert.True(t, *cust.Active)
	require.NotNil(t, cust.Taxable)
	assert.False(t, *cust.Taxable)
	require.NotNil(t, cust.Job)
	assert.False(t, *cust.Job)
	require.NotNil(t, cust.BillWithParent)
	assert.False(t, *cust.BillWithParent)

	assert.Equal(t, "85.0", cust.Balance.String())
	assert.Equal(t, "85.0", cust.BalanceWithJobs.String())
	assert.Equal(t, "Print", cust.PreferredDeliveryMethod)
	assert.Equal(t, "Preferred customer", cust.Notes)

	require.NotNil(t, cust.PrimaryPhone)
	assert.Equal(t, "(415) 444-6538", cust.PrimaryPhone.FreeFormNumber)
	require.NotNil(t, cust.AlternatePhone)
	assert.Equal(t, "(415) 444-6539", cust.AlternatePhone.FreeFormNumber)
	require.NotNil(t, cust.Mobile)
	assert.Equal(t, "(415) 555-1234", cust.Mobile.FreeFormNumber)
	require.NotNil(t, cust.Fax)
	assert.Equal(t, "(415) 555-5678", cust.Fax.FreeFormNumber)

	require.NotNil(t, cust.PrimaryEmailAddr)
	assert.Equal(t, "Surf@Intuit.com", cust.PrimaryEmailAddr.Address)

	require.NotNil(t, cust.WebAddr)
	assert.Equal(t, "http://billswindsurfshop.com", cust.WebAddr.URI)

	require.NotNil(t, cust.BillAddr)
	assert.Equal(t, "12 Ocean Dr.", cust.BillAddr.Line1)
	assert.Equal(t, "Half Moon Bay", cust.BillAddr.City)

	require.NotNil(t, cust.ShipAddr)
	assert.Equal(t, "12 Ocean Dr.", cust.ShipAddr.Line1)

	require.NotNil(t, cust.CurrencyRef)
	assert.Equal(t, "USD", cust.CurrencyRef.Value)
	require.NotNil(t, cust.PaymentMethodRef)
	assert.Equal(t, "2", cust.PaymentMethodRef.Value)
	require.NotNil(t, cust.SalesTermRef)
	assert.Equal(t, "3", cust.SalesTermRef.Value)
}

func TestCustomerJSONDecodesSubCustomer(t *testing.T) {
	const fixture = `{
		"Customer": {
			"Id": "99",
			"SyncToken": "0",
			"domain": "QBO",
			"status": "Active",
			"DisplayName": "Bill's Windsurf Shop:Job A",
			"FullyQualifiedName": "Bill's Windsurf Shop:Job A",
			"Level": 1,
			"Job": true,
			"BillWithParent": true,
			"ParentRef": {"value": "2", "name": "Bill's Windsurf Shop"},
			"IsProject": false,
			"Active": true
		}
	}`

	var envelope struct {
		Customer Customer `json:"Customer"`
	}
	require.NoError(t, json.Unmarshal([]byte(fixture), &envelope))

	cust := envelope.Customer
	assert.Equal(t, "99", cust.ID())
	assert.Equal(t, 1, cust.Level)
	require.NotNil(t, cust.Job)
	assert.True(t, *cust.Job)
	require.NotNil(t, cust.BillWithParent)
	assert.True(t, *cust.BillWithParent)
	require.NotNil(t, cust.ParentRef)
	assert.Equal(t, "2", cust.ParentRef.Value)
	require.NotNil(t, cust.IsProject)
	assert.False(t, *cust.IsProject)
}

func TestCustomerRequestsMarshalWritablePayloads(t *testing.T) {
	create := Customer{
		DisplayName: "King's Groceries",
		GivenName:   "James",
		FamilyName:  "King",
		CompanyName: "King Groceries",
		PrimaryPhone: &TelephoneNumber{
			FreeFormNumber: "(555) 555-5555",
		},
		PrimaryEmailAddr: &EmailAddress{
			Address: "jdrew@myemail.com",
		},
	}

	body, err := json.Marshal(create)
	require.NoError(t, err)
	assert.Contains(t, string(body), `"DisplayName"`)
	assert.Contains(t, string(body), `"GivenName"`)
	assert.Contains(t, string(body), `"PrimaryPhone"`)
	assert.NotContains(t, string(body), `"sparse"`)
	assert.NotContains(t, string(body), `"FullyQualifiedName"`)
	assert.NotContains(t, string(body), `"Level"`)
}

func TestCustomerServiceUpdateSparseInjectsSparseFlag(t *testing.T) {
	var capturedBody []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/company/123/customer", r.URL.Path)
		var err error
		capturedBody, err = io.ReadAll(r.Body)
		require.NoError(t, err)
		w.Header().Set("Content-Type", "application/json")
		_, err = io.WriteString(w, `{"Customer":{"Id":"2","SyncToken":"1"}}`)
		require.NoError(t, err)
	}))
	defer server.Close()

	client, err := NewClient(WithHTTPClient(server.Client()), WithBaseURL(server.URL+"/v3/company"))
	require.NoError(t, err)

	customer := Customer{
		IDValue:        IDValue{Id: "2"},
		SyncTokenValue: SyncTokenValue{SyncTokenValue: "0"},
		MiddleName:     "Mark",
	}

	_, err = client.Company("123", oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test-token"})).Customers().UpdateSparse(context.Background(), customer)
	require.NoError(t, err)

	assert.Contains(t, string(capturedBody), `"Id":"2"`)
	assert.Contains(t, string(capturedBody), `"SyncToken":"0"`)
	assert.Contains(t, string(capturedBody), `"sparse":true`)
	assert.Contains(t, string(capturedBody), `"MiddleName":"Mark"`)
}

func TestCustomerCDCExtraction(t *testing.T) {
	changedSince := time.Date(2026, time.March, 10, 12, 0, 0, 0, time.UTC)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/company/123/cdc", r.URL.Path)
		assert.Equal(t, string(EntityCustomer), r.URL.Query().Get("entities"))
		assert.Equal(t, changedSince.Format(time.RFC3339), r.URL.Query().Get("changedSince"))
		w.Header().Set("Content-Type", "application/json")
		_, err := io.WriteString(w, `{"CDCResponse":[{"QueryResponse":[{"Customer":[{"Id":"2","SyncToken":"1","domain":"QBO","status":"Deleted"}],"startPosition":1,"maxResults":1}]}],"time":"2026-03-10T12:00:00Z"}`)
		require.NoError(t, err)
	}))
	defer server.Close()

	client, err := NewClient(WithHTTPClient(server.Client()), WithBaseURL(server.URL+"/v3/company"))
	require.NoError(t, err)

	custs, err := client.Company("123", oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test-token"})).Customers().ChangeDataCapture(context.Background(), changedSince)
	require.NoError(t, err)
	require.Len(t, custs, 1)
	assert.Equal(t, "2", custs[0].ID())
	assert.Equal(t, "Deleted", custs[0].CDCStatus())
}

// Compile-time interface compliance checks for Customer and CustomerService.
var (
	_ Entity    = Customer{}
	_ CDCEntity = Customer{}

	_ Getter[Customer]          = (*CustomerService)(nil)
	_ Querier[Customer]         = (*CustomerService)(nil)
	_ Lister[Customer]          = (*CustomerService)(nil)
	_ Pager[Customer]           = (*CustomerService)(nil)
	_ Creator[Customer]         = (*CustomerService)(nil)
	_ Updater[Customer]         = (*CustomerService)(nil)
	_ SparseUpdater[Customer]   = (*CustomerService)(nil)
	_ EntityCDCReader[Customer] = (*CustomerService)(nil)
)
