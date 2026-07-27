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

func TestTaxCodeServiceGet(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/company/123/taxcode/20000000000000085548", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_, err := io.WriteString(w, `{"TaxCode":{"Id":"20000000000000085548","SyncToken":"0","domain":"QBO","status":"Active","Name":"CA","Active":true,"Taxable":true,"TaxGroup":false,"TaxCodeConfigType":"MANUAL","SalesTaxRateList":{"TaxRateDetail":[{"TaxRateRef":{"value":"20000000000000085547","name":"California State Tax"},"TaxTypeApplicable":"TaxOnAmount","TaxOrder":0}]}}}`)
		require.NoError(t, err)
	}))
	defer server.Close()

	client, err := NewClient(WithHTTPClient(server.Client()), WithBaseURL(server.URL+"/v3/company"))
	require.NoError(t, err)

	tc, err := client.Company("123", oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test-token"})).TaxCodes().Get(context.Background(), "20000000000000085548")
	require.NoError(t, err)
	assert.Equal(t, "20000000000000085548", tc.ID())
	assert.Equal(t, EntityTaxCode, tc.EntityName())
	assert.Equal(t, "CA", tc.Name)
	assert.Equal(t, "MANUAL", tc.TaxCodeConfigType)

	require.NotNil(t, tc.Active)
	assert.True(t, *tc.Active)
	require.NotNil(t, tc.Taxable)
	assert.True(t, *tc.Taxable)
	require.NotNil(t, tc.TaxGroup)
	assert.False(t, *tc.TaxGroup)

	require.NotNil(t, tc.SalesTaxRateList)
	require.Len(t, tc.SalesTaxRateList.TaxRateDetail, 1)
	detail := tc.SalesTaxRateList.TaxRateDetail[0]
	require.NotNil(t, detail.TaxRateRef)
	assert.Equal(t, "20000000000000085547", detail.TaxRateRef.Value)
	assert.Equal(t, "California State Tax", detail.TaxRateRef.Name)
	assert.Equal(t, "TaxOnAmount", detail.TaxTypeApplicable)
	assert.Equal(t, 0, detail.TaxOrder)
}

func TestTaxCodeServiceListPage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/company/123/query", r.URL.Path)
		assert.Equal(t, "SELECT * FROM TaxCode STARTPOSITION 1 MAXRESULTS 5", r.URL.Query().Get("query"))
		w.Header().Set("Content-Type", "application/json")
		_, err := io.WriteString(w, `{"QueryResponse":{"TaxCode":[{"Id":"1","SyncToken":"0","Name":"CA"},{"Id":"2","SyncToken":"0","Name":"NY"}],"startPosition":1,"maxResults":2}}`)
		require.NoError(t, err)
	}))
	defer server.Close()

	client, err := NewClient(WithHTTPClient(server.Client()), WithBaseURL(server.URL+"/v3/company"))
	require.NoError(t, err)

	tcs, err := client.Company("123", oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test-token"})).TaxCodes().ListPage(context.Background(), 1, 5)
	require.NoError(t, err)
	require.Len(t, tcs, 2)
	assert.Equal(t, "1", tcs[0].ID())
	assert.Equal(t, "CA", tcs[0].Name)
	assert.Equal(t, "2", tcs[1].ID())
	assert.Equal(t, "NY", tcs[1].Name)
}

func TestTaxCodeJSONDecodesV75Fields(t *testing.T) {
	const fixture = `{
		"TaxCode": {
			"Id": "20000000000000085548",
			"SyncToken": "1",
			"domain": "QBO",
			"status": "Active",
			"MetaData": {
				"CreateTime": "2015-07-22T13:57:27-07:00",
				"LastUpdatedTime": "2015-07-22T13:57:27-07:00"
			},
			"Name": "CA",
			"Description": "California sales tax",
			"Active": true,
			"Taxable": true,
			"TaxGroup": false,
			"Hidden": false,
			"TaxCodeConfigType": "MANUAL",
			"SalesTaxRateList": {
				"TaxRateDetail": [
					{
						"TaxRateRef": {"value": "20000000000000085547", "name": "California State Tax"},
						"TaxTypeApplicable": "TaxOnAmount",
						"TaxOrder": 0
					}
				]
			},
			"PurchaseTaxRateList": {
				"TaxRateDetail": [
					{
						"TaxRateRef": {"value": "20000000000000085547", "name": "California State Tax"},
						"TaxTypeApplicable": "TaxOnAmount",
						"TaxOrder": 0
					}
				]
			}
		}
	}`

	var envelope struct {
		TaxCode TaxCode `json:"TaxCode"`
	}
	require.NoError(t, json.Unmarshal([]byte(fixture), &envelope))

	tc := envelope.TaxCode
	assert.Equal(t, "20000000000000085548", tc.ID())
	assert.Equal(t, "1", tc.SyncToken())
	require.NotNil(t, tc.MetaData)
	require.NotNil(t, tc.MetaData.CreateTime)

	assert.Equal(t, "CA", tc.Name)
	assert.Equal(t, "California sales tax", tc.Description)
	assert.Equal(t, "MANUAL", tc.TaxCodeConfigType)

	require.NotNil(t, tc.Active)
	assert.True(t, *tc.Active)
	require.NotNil(t, tc.Taxable)
	assert.True(t, *tc.Taxable)
	require.NotNil(t, tc.TaxGroup)
	assert.False(t, *tc.TaxGroup)
	require.NotNil(t, tc.Hidden)
	assert.False(t, *tc.Hidden)

	require.NotNil(t, tc.SalesTaxRateList)
	require.Len(t, tc.SalesTaxRateList.TaxRateDetail, 1)
	assert.Equal(t, "20000000000000085547", tc.SalesTaxRateList.TaxRateDetail[0].TaxRateRef.Value)

	require.NotNil(t, tc.PurchaseTaxRateList)
	require.Len(t, tc.PurchaseTaxRateList.TaxRateDetail, 1)
	assert.Equal(t, "20000000000000085547", tc.PurchaseTaxRateList.TaxRateDetail[0].TaxRateRef.Value)
}

// Compile-time interface compliance checks for TaxCode and TaxCodeService.
var (
	_ Entity = TaxCode{}

	_ Getter[TaxCode]  = (*TaxCodeService)(nil)
	_ Querier[TaxCode] = (*TaxCodeService)(nil)
	_ Lister[TaxCode]  = (*TaxCodeService)(nil)
	_ Pager[TaxCode]   = (*TaxCodeService)(nil)
)
