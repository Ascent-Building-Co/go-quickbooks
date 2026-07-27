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

func TestTaxRateServiceGet(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/company/123/taxrate/20000000000000085547", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_, err := io.WriteString(w, `{"TaxRate":{"Id":"20000000000000085547","SyncToken":"0","domain":"QBO","status":"Active","Name":"California State Tax","RateValue":"8.25","Active":true,"AgencyRef":{"value":"1","name":"State Board of Equalization"},"SpecialTaxType":"NONE","DisplayType":"ReadOnly"}}`)
		require.NoError(t, err)
	}))
	defer server.Close()

	client, err := NewClient(WithHTTPClient(server.Client()), WithBaseURL(server.URL+"/v3/company"))
	require.NoError(t, err)

	tr, err := client.Company("123", oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test-token"})).TaxRates().Get(context.Background(), "20000000000000085547")
	require.NoError(t, err)
	assert.Equal(t, "20000000000000085547", tr.ID())
	assert.Equal(t, EntityTaxRate, tr.EntityName())
	assert.Equal(t, "California State Tax", tr.Name)
	assert.Equal(t, "8.25", tr.RateValue.String())
	assert.Equal(t, "NONE", tr.SpecialTaxType)
	assert.Equal(t, "ReadOnly", tr.DisplayType)
	require.NotNil(t, tr.Active)
	assert.True(t, *tr.Active)
	require.NotNil(t, tr.AgencyRef)
	assert.Equal(t, "1", tr.AgencyRef.Value)
	assert.Equal(t, "State Board of Equalization", tr.AgencyRef.Name)
}

func TestTaxRateServiceListPage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/company/123/query", r.URL.Path)
		assert.Equal(t, "SELECT * FROM TaxRate STARTPOSITION 1 MAXRESULTS 5", r.URL.Query().Get("query"))
		w.Header().Set("Content-Type", "application/json")
		_, err := io.WriteString(w, `{"QueryResponse":{"TaxRate":[{"Id":"1","SyncToken":"0","Name":"California State Tax","RateValue":"8.25"},{"Id":"2","SyncToken":"0","Name":"NY City Tax","RateValue":"4.5"}],"startPosition":1,"maxResults":2}}`)
		require.NoError(t, err)
	}))
	defer server.Close()

	client, err := NewClient(WithHTTPClient(server.Client()), WithBaseURL(server.URL+"/v3/company"))
	require.NoError(t, err)

	trs, err := client.Company("123", oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test-token"})).TaxRates().ListPage(context.Background(), 1, 5)
	require.NoError(t, err)
	require.Len(t, trs, 2)
	assert.Equal(t, "1", trs[0].ID())
	assert.Equal(t, "California State Tax", trs[0].Name)
	assert.Equal(t, "8.25", trs[0].RateValue.String())
	assert.Equal(t, "2", trs[1].ID())
	assert.Equal(t, "NY City Tax", trs[1].Name)
}

func TestTaxRateJSONDecodesV75Fields(t *testing.T) {
	const fixture = `{
		"TaxRate": {
			"Id": "20000000000000085547",
			"SyncToken": "1",
			"domain": "QBO",
			"status": "Active",
			"MetaData": {
				"CreateTime": "2015-07-22T13:57:27-07:00",
				"LastUpdatedTime": "2015-07-22T13:57:27-07:00"
			},
			"Name": "California State Tax",
			"Description": "CA state sales tax",
			"RateValue": "8.25",
			"Active": true,
			"AgencyRef": {"value": "1", "name": "State Board of Equalization"},
			"TaxReturnLineRef": {"value": "10", "name": "Line 1"},
			"SpecialTaxType": "NONE",
			"DisplayType": "ReadOnly",
			"OriginalTaxRate": "20000000000000085540",
			"EffectiveTaxRate": [
				{
					"RateValue": "8.25",
					"EffectiveDate": "2020-01-01T00:00:00-08:00",
					"EndDate": "2099-12-31T00:00:00-08:00"
				}
			]
		}
	}`

	var envelope struct {
		TaxRate TaxRate `json:"TaxRate"`
	}
	require.NoError(t, json.Unmarshal([]byte(fixture), &envelope))

	tr := envelope.TaxRate
	assert.Equal(t, "20000000000000085547", tr.ID())
	assert.Equal(t, "1", tr.SyncToken())
	require.NotNil(t, tr.MetaData)
	require.NotNil(t, tr.MetaData.CreateTime)

	assert.Equal(t, "California State Tax", tr.Name)
	assert.Equal(t, "CA state sales tax", tr.Description)
	assert.Equal(t, "8.25", tr.RateValue.String())
	assert.Equal(t, "NONE", tr.SpecialTaxType)
	assert.Equal(t, "ReadOnly", tr.DisplayType)
	assert.Equal(t, "20000000000000085540", tr.OriginalTaxRate)

	require.NotNil(t, tr.Active)
	assert.True(t, *tr.Active)

	require.NotNil(t, tr.AgencyRef)
	assert.Equal(t, "1", tr.AgencyRef.Value)

	require.NotNil(t, tr.TaxReturnLineRef)
	assert.Equal(t, "10", tr.TaxReturnLineRef.Value)

	require.Len(t, tr.EffectiveTaxRate, 1)
	assert.Equal(t, "8.25", tr.EffectiveTaxRate[0].RateValue.String())
	require.NotNil(t, tr.EffectiveTaxRate[0].EffectiveDate)
	require.NotNil(t, tr.EffectiveTaxRate[0].EndDate)
}

// Compile-time interface compliance checks for TaxRate and TaxRateService.
var (
	_ Entity = TaxRate{}

	_ Getter[TaxRate]  = (*TaxRateService)(nil)
	_ Querier[TaxRate] = (*TaxRateService)(nil)
	_ Lister[TaxRate]  = (*TaxRateService)(nil)
	_ Pager[TaxRate]   = (*TaxRateService)(nil)
)
