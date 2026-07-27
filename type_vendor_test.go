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

func TestVendorServiceGet(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/company/123/vendor/30", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_, err := io.WriteString(w, `{"Vendor":{"Id":"30","SyncToken":"1","domain":"QBO","status":"Active","DisplayName":"Books by Bessie","GivenName":"Bessie","FamilyName":"Williams","CompanyName":"Books by Bessie","Active":true,"Vendor1099":false,"Balance":0}}`)
		require.NoError(t, err)
	}))
	defer server.Close()

	client, err := NewClient(WithHTTPClient(server.Client()), WithBaseURL(server.URL+"/v3/company"))
	require.NoError(t, err)

	v, err := client.Company("123", oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test-token"})).Vendors().Get(context.Background(), "30")
	require.NoError(t, err)
	assert.Equal(t, "30", v.ID())
	assert.Equal(t, EntityVendor, v.EntityName())
	assert.Equal(t, "Active", v.CDCStatus())
	assert.Equal(t, "QBO", v.CDCDomain())
	assert.Equal(t, "Books by Bessie", v.DisplayName)
	assert.Equal(t, "Bessie", v.GivenName)
	assert.Equal(t, "Williams", v.FamilyName)
	require.NotNil(t, v.Active)
	assert.True(t, *v.Active)
}

func TestVendorServiceUpdateSparseInjectsSparseFlag(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/company/123/vendor", r.URL.Path)
		var payload map[string]interface{}
		require.NoError(t, json.NewDecoder(r.Body).Decode(&payload))
		assert.Equal(t, true, payload["sparse"])
		w.Header().Set("Content-Type", "application/json")
		_, err := io.WriteString(w, `{"Vendor":{"Id":"30","SyncToken":"2","DisplayName":"Books by Bessie"}}`)
		require.NoError(t, err)
	}))
	defer server.Close()

	client, err := NewClient(WithHTTPClient(server.Client()), WithBaseURL(server.URL+"/v3/company"))
	require.NoError(t, err)

	vendor := Vendor{}
	vendor.Id = "30"
	vendor.SyncTokenValue = SyncTokenValue{"1"}
	vendor.DisplayName = "Books by Bessie"

	_, err = client.Company("123", oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test-token"})).Vendors().UpdateSparse(context.Background(), vendor)
	require.NoError(t, err)
}

func TestVendorServiceListPage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/company/123/query", r.URL.Path)
		assert.Equal(t, "SELECT * FROM Vendor STARTPOSITION 1 MAXRESULTS 5", r.URL.Query().Get("query"))
		w.Header().Set("Content-Type", "application/json")
		_, err := io.WriteString(w, `{"QueryResponse":{"Vendor":[{"Id":"30","SyncToken":"1","DisplayName":"Books by Bessie"},{"Id":"31","SyncToken":"0","DisplayName":"Acme Supplies"}],"startPosition":1,"maxResults":2}}`)
		require.NoError(t, err)
	}))
	defer server.Close()

	client, err := NewClient(WithHTTPClient(server.Client()), WithBaseURL(server.URL+"/v3/company"))
	require.NoError(t, err)

	vendors, err := client.Company("123", oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test-token"})).Vendors().ListPage(context.Background(), 1, 5)
	require.NoError(t, err)
	require.Len(t, vendors, 2)
	assert.Equal(t, "30", vendors[0].ID())
	assert.Equal(t, "Books by Bessie", vendors[0].DisplayName)
	assert.Equal(t, "31", vendors[1].ID())
	assert.Equal(t, "Acme Supplies", vendors[1].DisplayName)
}

func TestVendorJSONDecodesV75Fields(t *testing.T) {
	const fixture = `{
		"Vendor": {
			"Id": "30",
			"SyncToken": "1",
			"domain": "QBO",
			"status": "Active",
			"MetaData": {
				"CreateTime": "2014-09-12T10:07:56-07:00",
				"LastUpdatedTime": "2015-07-28T13:34:38-07:00"
			},
			"Title": "Ms.",
			"GivenName": "Bessie",
			"MiddleName": "A",
			"FamilyName": "Williams",
			"Suffix": "Jr",
			"DisplayName": "Books by Bessie",
			"CompanyName": "Books by Bessie",
			"PrintOnCheckName": "Books by Bessie and Joan",
			"Active": true,
			"PrimaryPhone": {"FreeFormNumber": "(650) 555-7745"},
			"AlternatePhone": {"FreeFormNumber": "(650) 555-0000"},
			"Mobile": {"FreeFormNumber": "(650) 555-1111"},
			"Fax": {"FreeFormNumber": "(650) 555-2222"},
			"PrimaryEmailAddr": {"Address": "Books@Intuit.com"},
			"WebAddr": {"URI": "http://www.booksbybessie.co"},
			"BillAddr": {
				"Id": "31",
				"Line1": "15 Main St.",
				"City": "Palo Alto",
				"CountrySubDivisionCode": "CA",
				"PostalCode": "94303"
			},
			"OtherContactInfo": {
				"Type": "TelephoneNumber",
				"Telephone": {"FreeFormNumber": "(650) 555-9999"}
			},
			"APAccountRef": {"value": "49", "name": "Accounts Payable"},
			"TermRef": {"value": "3", "name": "Net 30"},
			"CurrencyRef": {"value": "USD", "name": "United States Dollar"},
			"AcctNum": "13451234",
			"TaxIdentifier": "99-1234567",
			"Vendor1099": false,
			"CostRate": "50.00",
			"BillRate": "75.00",
			"Balance": 0
		}
	}`

	var envelope struct {
		Vendor Vendor `json:"Vendor"`
	}
	require.NoError(t, json.Unmarshal([]byte(fixture), &envelope))

	v := envelope.Vendor
	assert.Equal(t, "30", v.ID())
	assert.Equal(t, "1", v.SyncToken())
	assert.Equal(t, "QBO", v.CDCDomain())
	assert.Equal(t, "Active", v.CDCStatus())

	require.NotNil(t, v.MetaData)
	require.NotNil(t, v.MetaData.CreateTime)

	assert.Equal(t, "Ms.", v.Title)
	assert.Equal(t, "Bessie", v.GivenName)
	assert.Equal(t, "A", v.MiddleName)
	assert.Equal(t, "Williams", v.FamilyName)
	assert.Equal(t, "Jr", v.Suffix)
	assert.Equal(t, "Books by Bessie", v.DisplayName)
	assert.Equal(t, "Books by Bessie", v.CompanyName)
	assert.Equal(t, "Books by Bessie and Joan", v.PrintOnCheckName)
	assert.Equal(t, "13451234", v.AcctNum)
	assert.Equal(t, "99-1234567", v.TaxIdentifier)
	assert.Equal(t, "50.00", v.CostRate.String())
	assert.Equal(t, "75.00", v.BillRate.String())

	require.NotNil(t, v.Active)
	assert.True(t, *v.Active)
	require.NotNil(t, v.Vendor1099)
	assert.False(t, *v.Vendor1099)

	require.NotNil(t, v.PrimaryPhone)
	assert.Equal(t, "(650) 555-7745", v.PrimaryPhone.FreeFormNumber)
	require.NotNil(t, v.PrimaryEmailAddr)
	assert.Equal(t, "Books@Intuit.com", v.PrimaryEmailAddr.Address)
	require.NotNil(t, v.WebAddr)
	assert.Equal(t, "http://www.booksbybessie.co", v.WebAddr.URI)
	require.NotNil(t, v.BillAddr)
	assert.Equal(t, "15 Main St.", v.BillAddr.Line1)
	assert.Equal(t, "Palo Alto", v.BillAddr.City)

	require.NotNil(t, v.OtherContactInfo)
	assert.Equal(t, "TelephoneNumber", v.OtherContactInfo.Type)
	require.NotNil(t, v.OtherContactInfo.Telephone)
	assert.Equal(t, "(650) 555-9999", v.OtherContactInfo.Telephone.FreeFormNumber)

	require.NotNil(t, v.APAccountRef)
	assert.Equal(t, "49", v.APAccountRef.Value)
	require.NotNil(t, v.TermRef)
	assert.Equal(t, "3", v.TermRef.Value)
	require.NotNil(t, v.CurrencyRef)
	assert.Equal(t, "USD", v.CurrencyRef.Value)
}

func TestVendorRequestsMarshalWritablePayloads(t *testing.T) {
	create := Vendor{
		DisplayName: "Books by Bessie",
		GivenName:   "Bessie",
		FamilyName:  "Williams",
	}

	body, err := json.Marshal(create)
	require.NoError(t, err)
	assert.Contains(t, string(body), `"DisplayName":"Books by Bessie"`)
	assert.NotContains(t, string(body), `"Balance"`)
	assert.NotContains(t, string(body), `"sparse"`)
}

func TestVendorCDCExtraction(t *testing.T) {
	changedSince := time.Date(2026, time.March, 10, 12, 0, 0, 0, time.UTC)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/company/123/cdc", r.URL.Path)
		assert.Equal(t, string(EntityVendor), r.URL.Query().Get("entities"))
		assert.Equal(t, changedSince.Format(time.RFC3339), r.URL.Query().Get("changedSince"))
		w.Header().Set("Content-Type", "application/json")
		_, err := io.WriteString(w, `{"CDCResponse":[{"QueryResponse":[{"Vendor":[{"Id":"30","SyncToken":"2","domain":"QBO","status":"Deleted"}],"startPosition":1,"maxResults":1}]}],"time":"2026-03-10T12:00:00Z"}`)
		require.NoError(t, err)
	}))
	defer server.Close()

	client, err := NewClient(WithHTTPClient(server.Client()), WithBaseURL(server.URL+"/v3/company"))
	require.NoError(t, err)

	vendors, err := client.Company("123", oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test-token"})).Vendors().ChangeDataCapture(context.Background(), changedSince)
	require.NoError(t, err)
	require.Len(t, vendors, 1)
	assert.Equal(t, "30", vendors[0].ID())
	assert.Equal(t, "Deleted", vendors[0].CDCStatus())
}

// Compile-time interface compliance checks for Vendor and VendorService.
var (
	_ Entity    = Vendor{}
	_ CDCEntity = Vendor{}

	_ Getter[Vendor]          = (*VendorService)(nil)
	_ Querier[Vendor]         = (*VendorService)(nil)
	_ Lister[Vendor]          = (*VendorService)(nil)
	_ Pager[Vendor]           = (*VendorService)(nil)
	_ Creator[Vendor]         = (*VendorService)(nil)
	_ Updater[Vendor]         = (*VendorService)(nil)
	_ SparseUpdater[Vendor]   = (*VendorService)(nil)
	_ EntityCDCReader[Vendor] = (*VendorService)(nil)
)
