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

func TestTaxAgencyServiceGet(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/company/123/taxagency/1", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_, err := io.WriteString(w, `{"TaxAgency":{"Id":"1","SyncToken":"0","DisplayName":"State Board of Equalization","TaxTrackedOnSales":true,"TaxTrackedOnPurchases":false,"TaxRegistrationNumber":"CA-1234","TaxAgencyConfig":"AUTOMATED_SALES_TAX"}}`)
		require.NoError(t, err)
	}))
	defer server.Close()

	client, err := NewClient(WithHTTPClient(server.Client()), WithBaseURL(server.URL+"/v3/company"))
	require.NoError(t, err)

	ta, err := client.Company("123", oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test-token"})).TaxAgencies().Get(context.Background(), "1")
	require.NoError(t, err)
	assert.Equal(t, "1", ta.ID())
	assert.Equal(t, EntityTaxAgency, ta.EntityName())
	assert.Equal(t, "State Board of Equalization", ta.DisplayName)
	assert.Equal(t, "CA-1234", ta.TaxRegistrationNumber)
	assert.Equal(t, "AUTOMATED_SALES_TAX", ta.TaxAgencyConfig)
	require.NotNil(t, ta.TaxTrackedOnSales)
	assert.True(t, *ta.TaxTrackedOnSales)
	require.NotNil(t, ta.TaxTrackedOnPurchases)
	assert.False(t, *ta.TaxTrackedOnPurchases)
}

func TestTaxAgencyServiceListPage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/company/123/query", r.URL.Path)
		assert.Equal(t, "SELECT * FROM TaxAgency STARTPOSITION 1 MAXRESULTS 5", r.URL.Query().Get("query"))
		w.Header().Set("Content-Type", "application/json")
		_, err := io.WriteString(w, `{"QueryResponse":{"TaxAgency":[{"Id":"1","SyncToken":"0","DisplayName":"State Board of Equalization"},{"Id":"2","SyncToken":"0","DisplayName":"IRS"}],"startPosition":1,"maxResults":2}}`)
		require.NoError(t, err)
	}))
	defer server.Close()

	client, err := NewClient(WithHTTPClient(server.Client()), WithBaseURL(server.URL+"/v3/company"))
	require.NoError(t, err)

	agencies, err := client.Company("123", oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test-token"})).TaxAgencies().ListPage(context.Background(), 1, 5)
	require.NoError(t, err)
	require.Len(t, agencies, 2)
	assert.Equal(t, "1", agencies[0].ID())
	assert.Equal(t, "State Board of Equalization", agencies[0].DisplayName)
	assert.Equal(t, "2", agencies[1].ID())
	assert.Equal(t, "IRS", agencies[1].DisplayName)
}

func TestTaxAgencyJSONDecodesV75Fields(t *testing.T) {
	const fixture = `{
		"TaxAgency": {
			"Id": "1",
			"SyncToken": "0",
			"MetaData": {
				"CreateTime": "2015-07-22T13:57:27-07:00",
				"LastUpdatedTime": "2015-07-22T13:57:27-07:00"
			},
			"DisplayName": "State Board of Equalization",
			"TaxTrackedOnSales": true,
			"TaxTrackedOnPurchases": false,
			"LastFileDate": "2026-01-31",
			"TaxRegistrationNumber": "CA-1234",
			"TaxAgencyConfig": "AUTOMATED_SALES_TAX"
		}
	}`

	var envelope struct {
		TaxAgency TaxAgency `json:"TaxAgency"`
	}
	require.NoError(t, json.Unmarshal([]byte(fixture), &envelope))

	ta := envelope.TaxAgency
	assert.Equal(t, "1", ta.ID())
	assert.Equal(t, "0", ta.SyncToken())

	require.NotNil(t, ta.MetaData)
	require.NotNil(t, ta.MetaData.CreateTime)

	assert.Equal(t, "State Board of Equalization", ta.DisplayName)
	assert.Equal(t, "CA-1234", ta.TaxRegistrationNumber)
	assert.Equal(t, "AUTOMATED_SALES_TAX", ta.TaxAgencyConfig)

	require.NotNil(t, ta.TaxTrackedOnSales)
	assert.True(t, *ta.TaxTrackedOnSales)
	require.NotNil(t, ta.TaxTrackedOnPurchases)
	assert.False(t, *ta.TaxTrackedOnPurchases)

	require.NotNil(t, ta.LastFileDate)
	assert.Equal(t, "2026-01-31", ta.LastFileDate.String())
}

func TestTaxAgencyRequestsMarshalWritablePayloads(t *testing.T) {
	create := TaxAgency{
		DisplayName: "State Board of Equalization",
	}

	body, err := json.Marshal(create)
	require.NoError(t, err)
	assert.Contains(t, string(body), `"DisplayName":"State Board of Equalization"`)
	// TaxAgencyConfig is read-only; omitempty ensures it is not sent when empty
	assert.NotContains(t, string(body), `"TaxAgencyConfig"`)
	assert.NotContains(t, string(body), `"sparse"`)
}

// Compile-time interface compliance checks for TaxAgency and TaxAgencyService.
var (
	_ Entity = TaxAgency{}

	_ Getter[TaxAgency]  = (*TaxAgencyService)(nil)
	_ Querier[TaxAgency] = (*TaxAgencyService)(nil)
	_ Lister[TaxAgency]  = (*TaxAgencyService)(nil)
	_ Pager[TaxAgency]   = (*TaxAgencyService)(nil)
	_ Creator[TaxAgency] = (*TaxAgencyService)(nil)
)
