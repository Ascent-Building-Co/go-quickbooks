package quickbooks

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
)

func TestCompanyInfoServiceGet(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// realmID is used as the entity ID
		assert.Equal(t, "/v3/company/123/companyinfo/123", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_, err := io.WriteString(w, `{"CompanyInfo":{"Id":"123","SyncToken":"0","CompanyName":"Sandbox Company_US_1","CompanyAddr":{"Line1":"2500 Lake Park Blvd","City":"Salt Lake City","CountrySubDivisionCode":"UT","PostalCode":"84120"},"FiscalYearStartMonth":"January","Country":"US"}}`)
		require.NoError(t, err)
	}))
	defer server.Close()

	client, err := NewClient(WithHTTPClient(server.Client()), WithBaseURL(server.URL+"/v3/company"))
	require.NoError(t, err)

	ci, err := client.Company("123", oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test-token"})).CompanyInfo().Get(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "123", ci.ID())
	assert.Equal(t, EntityCompanyInfo, ci.EntityName())
	assert.Equal(t, "Sandbox Company_US_1", ci.CompanyName)
	assert.Equal(t, MonthJanuary, ci.FiscalYearStartMonth)
	assert.Equal(t, "US", ci.Country)
	require.NotNil(t, ci.CompanyAddr)
	assert.Equal(t, "2500 Lake Park Blvd", ci.CompanyAddr.Line1)
	assert.Equal(t, "Salt Lake City", ci.CompanyAddr.City)
}

func TestCompanyInfoServiceUpdateSparseInjectsSparseFlag(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/company/123/companyinfo", r.URL.Path)
		var payload map[string]interface{}
		require.NoError(t, json.NewDecoder(r.Body).Decode(&payload))
		assert.Equal(t, true, payload["sparse"])
		w.Header().Set("Content-Type", "application/json")
		_, err := io.WriteString(w, `{"CompanyInfo":{"Id":"123","SyncToken":"1","CompanyName":"Sandbox Company_US_1"}}`)
		require.NoError(t, err)
	}))
	defer server.Close()

	client, err := NewClient(WithHTTPClient(server.Client()), WithBaseURL(server.URL+"/v3/company"))
	require.NoError(t, err)

	ci := CompanyInfo{}
	ci.Id = "123"
	ci.SyncTokenValue = SyncTokenValue{"0"}
	ci.CompanyName = "Sandbox Company_US_1"

	_, err = client.Company("123", oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test-token"})).CompanyInfo().UpdateSparse(context.Background(), ci)
	require.NoError(t, err)
}

func TestCompanyInfoJSONDecodesV75Fields(t *testing.T) {
	const fixture = `{
		"CompanyInfo": {
			"Id": "123",
			"SyncToken": "1",
			"MetaData": {
				"CreateTime": "2015-07-22T13:57:27-07:00",
				"LastUpdatedTime": "2026-01-01T10:00:00-07:00"
			},
			"CompanyName": "Sandbox Company_US_1",
			"LegalName": "Sandbox Legal Name LLC",
			"CompanyAddr": {
				"Id": "1",
				"Line1": "2500 Lake Park Blvd",
				"City": "Salt Lake City",
				"CountrySubDivisionCode": "UT",
				"PostalCode": "84120",
				"Country": "US"
			},
			"LegalAddr": {
				"Id": "2",
				"Line1": "123 Legal St.",
				"City": "Salt Lake City",
				"CountrySubDivisionCode": "UT",
				"PostalCode": "84101"
			},
			"CustomerCommunicationAddr": {
				"Id": "3",
				"Line1": "PO Box 999",
				"City": "Salt Lake City"
			},
			"PrimaryPhone": {"FreeFormNumber": "(801) 555-1234"},
			"Email": {"Address": "admin@example.com"},
			"WebAddr": {"URI": "https://www.example.com"},
			"FiscalYearStartMonth": "January",
			"Country": "US",
			"SupportedLanguages": "en",
			"EmployerId": "12-3456789",
			"CompanyStartDate": "2015-07-22T13:57:27-07:00",
			"NameValue": [
				{"Name": "NeoEnabled", "Value": "true"},
				{"Name": "TimeTrackingFeature", "Value": "true"}
			]
		}
	}`

	var envelope struct {
		CompanyInfo CompanyInfo `json:"CompanyInfo"`
	}
	require.NoError(t, json.Unmarshal([]byte(fixture), &envelope))

	ci := envelope.CompanyInfo
	assert.Equal(t, "123", ci.ID())
	assert.Equal(t, "1", ci.SyncToken())

	require.NotNil(t, ci.MetaData)
	require.NotNil(t, ci.MetaData.CreateTime)

	assert.Equal(t, "Sandbox Company_US_1", ci.CompanyName)
	assert.Equal(t, "Sandbox Legal Name LLC", ci.LegalName)
	assert.Equal(t, "US", ci.Country)
	assert.Equal(t, "en", ci.SupportedLanguages)
	assert.Equal(t, "12-3456789", ci.EmployerId)
	assert.Equal(t, MonthJanuary, ci.FiscalYearStartMonth)

	require.NotNil(t, ci.CompanyAddr)
	assert.Equal(t, "2500 Lake Park Blvd", ci.CompanyAddr.Line1)
	assert.Equal(t, "Salt Lake City", ci.CompanyAddr.City)

	require.NotNil(t, ci.LegalAddr)
	assert.Equal(t, "123 Legal St.", ci.LegalAddr.Line1)

	require.NotNil(t, ci.CustomerCommunicationAddr)
	assert.Equal(t, "PO Box 999", ci.CustomerCommunicationAddr.Line1)

	require.NotNil(t, ci.PrimaryPhone)
	assert.Equal(t, "(801) 555-1234", ci.PrimaryPhone.FreeFormNumber)
	require.NotNil(t, ci.Email)
	assert.Equal(t, "admin@example.com", ci.Email.Address)
	require.NotNil(t, ci.WebAddr)
	assert.Equal(t, "https://www.example.com", ci.WebAddr.URI)

	require.NotNil(t, ci.CompanyStartDate)

	require.Len(t, ci.NameValue, 2)
	assert.Equal(t, "NeoEnabled", ci.NameValue[0].Name)
	assert.Equal(t, "true", ci.NameValue[0].Value)
}

func TestCompanyInfoRequestsMarshalWritablePayloads(t *testing.T) {
	update := CompanyInfo{
		CompanyName: "Sandbox Company_US_1",
		CompanyAddr: &PhysicalAddress{Line1: "2500 Lake Park Blvd"},
	}

	body, err := json.Marshal(update)
	require.NoError(t, err)
	assert.Contains(t, string(body), `"CompanyName":"Sandbox Company_US_1"`)
	assert.Contains(t, string(body), `"CompanyAddr"`)
	// CompanyStartDate is read-only; omitempty ensures it is absent when zero
	assert.NotContains(t, string(body), `"CompanyStartDate"`)
	assert.NotContains(t, string(body), `"sparse"`)
}

// Compile-time interface compliance checks for CompanyInfo and CompanyInfoService.
// CompanyInfo.Get does not satisfy Getter[T] by design — the realmID is used
// as the entity ID and is bound to the CompanyClient rather than passed explicitly.
var (
	_ Entity    = CompanyInfo{}
	_ CDCEntity = CompanyInfo{}

	_ Querier[CompanyInfo]         = (*CompanyInfoService)(nil)
	_ Lister[CompanyInfo]          = (*CompanyInfoService)(nil)
	_ Updater[CompanyInfo]         = (*CompanyInfoService)(nil)
	_ SparseUpdater[CompanyInfo]   = (*CompanyInfoService)(nil)
	_ EntityCDCReader[CompanyInfo] = (*CompanyInfoService)(nil)
)
