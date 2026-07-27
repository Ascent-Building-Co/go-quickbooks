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

func TestCompanyCurrencyServiceGet(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/company/123/companycurrency/1", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_, err := io.WriteString(w, `{"CompanyCurrency":{"Id":"1","SyncToken":"0","Code":"GBP","Name":"British Pound Sterling","Active":true}}`)
		require.NoError(t, err)
	}))
	defer server.Close()

	client, err := NewClient(WithHTTPClient(server.Client()), WithBaseURL(server.URL+"/v3/company"))
	require.NoError(t, err)

	cc, err := client.Company("123", oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test-token"})).CompanyCurrencies().Get(context.Background(), "1")
	require.NoError(t, err)
	assert.Equal(t, "1", cc.ID())
	assert.Equal(t, EntityCompanyCurrency, cc.EntityName())
	assert.Equal(t, "GBP", cc.Code)
	assert.Equal(t, "British Pound Sterling", cc.Name)
	require.NotNil(t, cc.Active)
	assert.True(t, *cc.Active)
}

func TestCompanyCurrencyServiceListPage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/company/123/query", r.URL.Path)
		assert.Equal(t, "SELECT * FROM CompanyCurrency STARTPOSITION 1 MAXRESULTS 5", r.URL.Query().Get("query"))
		w.Header().Set("Content-Type", "application/json")
		_, err := io.WriteString(w, `{"QueryResponse":{"CompanyCurrency":[{"Id":"1","SyncToken":"0","Code":"GBP","Name":"British Pound Sterling"},{"Id":"2","SyncToken":"0","Code":"EUR","Name":"Euro"}],"startPosition":1,"maxResults":2}}`)
		require.NoError(t, err)
	}))
	defer server.Close()

	client, err := NewClient(WithHTTPClient(server.Client()), WithBaseURL(server.URL+"/v3/company"))
	require.NoError(t, err)

	ccs, err := client.Company("123", oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test-token"})).CompanyCurrencies().ListPage(context.Background(), 1, 5)
	require.NoError(t, err)
	require.Len(t, ccs, 2)
	assert.Equal(t, "1", ccs[0].ID())
	assert.Equal(t, "GBP", ccs[0].Code)
	assert.Equal(t, "2", ccs[1].ID())
	assert.Equal(t, "EUR", ccs[1].Code)
}

func TestCompanyCurrencyJSONDecodesV75Fields(t *testing.T) {
	const fixture = `{
		"CompanyCurrency": {
			"Id": "3",
			"SyncToken": "1",
			"MetaData": {
				"CreateTime": "2026-01-15T10:00:00-07:00",
				"LastUpdatedTime": "2026-01-15T10:00:00-07:00"
			},
			"Code": "AUD",
			"Name": "Australian Dollar",
			"Active": true
		}
	}`

	var envelope struct {
		CompanyCurrency CompanyCurrency `json:"CompanyCurrency"`
	}
	require.NoError(t, json.Unmarshal([]byte(fixture), &envelope))

	cc := envelope.CompanyCurrency
	assert.Equal(t, "3", cc.ID())
	assert.Equal(t, "1", cc.SyncToken())
	assert.Equal(t, "AUD", cc.Code)
	assert.Equal(t, "Australian Dollar", cc.Name)
	require.NotNil(t, cc.Active)
	assert.True(t, *cc.Active)
	require.NotNil(t, cc.MetaData)
	require.NotNil(t, cc.MetaData.CreateTime)
}

func TestCompanyCurrencyRequestsMarshalWritablePayloads(t *testing.T) {
	create := CompanyCurrency{
		Code: "GBP",
	}

	body, err := json.Marshal(create)
	require.NoError(t, err)
	assert.Contains(t, string(body), `"Code":"GBP"`)
	assert.NotContains(t, string(body), `"Id"`)
	assert.NotContains(t, string(body), `"sparse"`)
}

// Compile-time interface compliance checks for CompanyCurrency and CompanyCurrencyService.
var (
	_ Entity    = CompanyCurrency{}
	_ CDCEntity = CompanyCurrency{}

	_ Getter[CompanyCurrency]          = (*CompanyCurrencyService)(nil)
	_ Querier[CompanyCurrency]         = (*CompanyCurrencyService)(nil)
	_ Lister[CompanyCurrency]          = (*CompanyCurrencyService)(nil)
	_ Pager[CompanyCurrency]           = (*CompanyCurrencyService)(nil)
	_ Creator[CompanyCurrency]         = (*CompanyCurrencyService)(nil)
	_ Updater[CompanyCurrency]         = (*CompanyCurrencyService)(nil)
	_ EntityCDCReader[CompanyCurrency] = (*CompanyCurrencyService)(nil)
)
