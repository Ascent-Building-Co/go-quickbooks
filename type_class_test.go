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

func TestClassServiceGet(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/company/123/class/5000000000000007280", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_, err := io.WriteString(w, `{"Class":{"Id":"5000000000000007280","SyncToken":"0","domain":"QBO","status":"Active","Name":"France","FullyQualifiedName":"France","Active":true,"SubClass":false}}`)
		require.NoError(t, err)
	}))
	defer server.Close()

	client, err := NewClient(WithHTTPClient(server.Client()), WithBaseURL(server.URL+"/v3/company"))
	require.NoError(t, err)

	cl, err := client.Company("123", oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test-token"})).Classes().Get(context.Background(), "5000000000000007280")
	require.NoError(t, err)
	assert.Equal(t, "5000000000000007280", cl.ID())
	assert.Equal(t, EntityClass, cl.EntityName())
	assert.Equal(t, "Active", cl.CDCStatus())
	assert.Equal(t, "QBO", cl.CDCDomain())
	assert.Equal(t, "France", cl.Name)
	assert.Equal(t, "France", cl.FullyQualifiedName)
}

func TestClassServiceQueryUsesQueryOptions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/company/123/query", r.URL.Path)
		assert.Equal(t, "SELECT * FROM Class STARTPOSITION 11 MAXRESULTS 10", r.URL.Query().Get("query"))
		w.Header().Set("Content-Type", "application/json")
		_, err := io.WriteString(w, `{"QueryResponse":{"Class":[{"Id":"100","SyncToken":"0"},{"Id":"101","SyncToken":"1"}],"startPosition":6,"maxResults":10}}`)
		require.NoError(t, err)
	}))
	defer server.Close()

	client, err := NewClient(WithHTTPClient(server.Client()), WithBaseURL(server.URL+"/v3/company"))
	require.NoError(t, err)

	cls, err := client.Company("123", oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test-token"})).Classes().ListPage(context.Background(), 2, 10)
	require.NoError(t, err)
	require.Len(t, cls, 2)
	assert.Equal(t, "100", cls[0].ID())
	assert.Equal(t, "101", cls[1].ID())
}

func TestClassJSONDecodesV75Fields(t *testing.T) {
	const fixture = `{
		"Class": {
			"Id": "5000000000000007280",
			"SyncToken": "0",
			"domain": "QBO",
			"status": "Active",
			"MetaData": {
				"CreateTime": "2015-07-22T13:57:27-07:00",
				"LastUpdatedTime": "2015-07-22T13:57:27-07:00"
			},
			"Name": "Paris",
			"FullyQualifiedName": "France:Paris",
			"SubClass": true,
			"Active": true,
			"ParentRef": {"value": "5000000000000007279", "name": "France"}
		}
	}`

	var envelope struct {
		Class Class `json:"Class"`
	}
	require.NoError(t, json.Unmarshal([]byte(fixture), &envelope))

	cl := envelope.Class
	assert.Equal(t, "5000000000000007280", cl.ID())
	assert.Equal(t, "0", cl.SyncToken())
	assert.Equal(t, "QBO", cl.CDCDomain())
	assert.Equal(t, "Active", cl.CDCStatus())

	require.NotNil(t, cl.MetaData)
	require.NotNil(t, cl.MetaData.CreateTime)

	assert.Equal(t, "Paris", cl.Name)
	assert.Equal(t, "France:Paris", cl.FullyQualifiedName)

	require.NotNil(t, cl.SubClass)
	assert.True(t, *cl.SubClass)

	require.NotNil(t, cl.Active)
	assert.True(t, *cl.Active)

	require.NotNil(t, cl.ParentRef)
	assert.Equal(t, "5000000000000007279", cl.ParentRef.Value)
	assert.Equal(t, "France", cl.ParentRef.Name)
}

func TestClassRequestsMarshalWritablePayloads(t *testing.T) {
	create := Class{
		Name: "France",
	}

	body, err := json.Marshal(create)
	require.NoError(t, err)
	assert.Contains(t, string(body), `"Name":"France"`)
	assert.NotContains(t, string(body), `"FullyQualifiedName"`)
	assert.NotContains(t, string(body), `"sparse"`)
}

func TestClassCDCExtraction(t *testing.T) {
	changedSince := time.Date(2026, time.March, 10, 12, 0, 0, 0, time.UTC)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/company/123/cdc", r.URL.Path)
		assert.Equal(t, string(EntityClass), r.URL.Query().Get("entities"))
		assert.Equal(t, changedSince.Format(time.RFC3339), r.URL.Query().Get("changedSince"))
		w.Header().Set("Content-Type", "application/json")
		_, err := io.WriteString(w, `{"CDCResponse":[{"QueryResponse":[{"Class":[{"Id":"5000000000000007280","SyncToken":"1","domain":"QBO","status":"Deleted"}],"startPosition":1,"maxResults":1}]}],"time":"2026-03-10T12:00:00Z"}`)
		require.NoError(t, err)
	}))
	defer server.Close()

	client, err := NewClient(WithHTTPClient(server.Client()), WithBaseURL(server.URL+"/v3/company"))
	require.NoError(t, err)

	cls, err := client.Company("123", oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test-token"})).Classes().ChangeDataCapture(context.Background(), changedSince)
	require.NoError(t, err)
	require.Len(t, cls, 1)
	assert.Equal(t, "5000000000000007280", cls[0].ID())
	assert.Equal(t, "Deleted", cls[0].CDCStatus())
}

// Compile-time interface compliance checks for Class and ClassService.
var (
	_ Entity    = Class{}
	_ CDCEntity = Class{}

	_ Getter[Class]          = (*ClassService)(nil)
	_ Querier[Class]         = (*ClassService)(nil)
	_ Lister[Class]          = (*ClassService)(nil)
	_ Pager[Class]           = (*ClassService)(nil)
	_ Creator[Class]         = (*ClassService)(nil)
	_ Updater[Class]         = (*ClassService)(nil)
	_ SparseUpdater[Class]   = (*ClassService)(nil)
	_ EntityCDCReader[Class] = (*ClassService)(nil)
)

func TestClassServiceUpdateSparseInjectsSparseFlag(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/company/123/class", r.URL.Path)
		var payload map[string]interface{}
		require.NoError(t, json.NewDecoder(r.Body).Decode(&payload))
		assert.Equal(t, true, payload["sparse"])
		w.Header().Set("Content-Type", "application/json")
		_, err := io.WriteString(w, `{"Class":{"Id":"1","SyncToken":"2"}}`)
		require.NoError(t, err)
	}))
	defer server.Close()

	client, err := NewClient(WithHTTPClient(server.Client()), WithBaseURL(server.URL+"/v3/company"))
	require.NoError(t, err)

	class := Class{}
	class.Id = "1"
	class.SyncTokenValue = SyncTokenValue{"1"}

	_, err = client.Company("123", oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test-token"})).Classes().UpdateSparse(context.Background(), class)
	require.NoError(t, err)
}
