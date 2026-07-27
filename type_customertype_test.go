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

func TestCustomerTypeServiceGet(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/company/123/customertype/1", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_, err := io.WriteString(w, `{"CustomerType":{"Id":"1","SyncToken":"0","Name":"Commercial","Active":true}}`)
		require.NoError(t, err)
	}))
	defer server.Close()

	client, err := NewClient(WithHTTPClient(server.Client()), WithBaseURL(server.URL+"/v3/company"))
	require.NoError(t, err)

	ct, err := client.Company("123", oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test-token"})).CustomerTypes().Get(context.Background(), "1")
	require.NoError(t, err)
	assert.Equal(t, "1", ct.ID())
	assert.Equal(t, EntityCustomerType, ct.EntityName())
	assert.Equal(t, "Commercial", ct.Name)
	require.NotNil(t, ct.Active)
	assert.True(t, *ct.Active)
}

func TestCustomerTypeServiceListPage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/company/123/query", r.URL.Path)
		assert.Equal(t, "SELECT * FROM CustomerType STARTPOSITION 1 MAXRESULTS 5", r.URL.Query().Get("query"))
		w.Header().Set("Content-Type", "application/json")
		_, err := io.WriteString(w, `{"QueryResponse":{"CustomerType":[{"Id":"1","SyncToken":"0","Name":"Commercial"},{"Id":"2","SyncToken":"0","Name":"Retail"}],"startPosition":1,"maxResults":2}}`)
		require.NoError(t, err)
	}))
	defer server.Close()

	client, err := NewClient(WithHTTPClient(server.Client()), WithBaseURL(server.URL+"/v3/company"))
	require.NoError(t, err)

	cts, err := client.Company("123", oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test-token"})).CustomerTypes().ListPage(context.Background(), 1, 5)
	require.NoError(t, err)
	require.Len(t, cts, 2)
	assert.Equal(t, "1", cts[0].ID())
	assert.Equal(t, "Commercial", cts[0].Name)
	assert.Equal(t, "2", cts[1].ID())
	assert.Equal(t, "Retail", cts[1].Name)
}

func TestCustomerTypeJSONDecodesV75Fields(t *testing.T) {
	const fixture = `{
		"CustomerType": {
			"Id": "1",
			"SyncToken": "0",
			"MetaData": {
				"CreateTime": "2015-07-22T13:57:27-07:00",
				"LastUpdatedTime": "2015-07-22T13:57:27-07:00"
			},
			"Name": "Commercial",
			"Active": true
		}
	}`

	var envelope struct {
		CustomerType CustomerType `json:"CustomerType"`
	}
	require.NoError(t, json.Unmarshal([]byte(fixture), &envelope))

	ct := envelope.CustomerType
	assert.Equal(t, "1", ct.ID())
	assert.Equal(t, "0", ct.SyncToken())
	require.NotNil(t, ct.MetaData)
	require.NotNil(t, ct.MetaData.CreateTime)
	assert.Equal(t, "Commercial", ct.Name)
	require.NotNil(t, ct.Active)
	assert.True(t, *ct.Active)
}

// Compile-time interface compliance checks for CustomerType and CustomerTypeService.
var (
	_ Entity    = CustomerType{}
	_ CDCEntity = CustomerType{}

	_ Getter[CustomerType]          = (*CustomerTypeService)(nil)
	_ Querier[CustomerType]         = (*CustomerTypeService)(nil)
	_ Lister[CustomerType]          = (*CustomerTypeService)(nil)
	_ Pager[CustomerType]           = (*CustomerTypeService)(nil)
	_ EntityCDCReader[CustomerType] = (*CustomerTypeService)(nil)
)
