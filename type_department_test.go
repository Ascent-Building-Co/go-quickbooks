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

func TestDepartmentServiceGet(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/company/123/department/2", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_, err := io.WriteString(w, `{"Department":{"Id":"2","SyncToken":"0","domain":"QBO","status":"Active","Name":"Marketing Department","FullyQualifiedName":"Marketing Department","SubDepartment":false,"Active":true}}`)
		require.NoError(t, err)
	}))
	defer server.Close()

	client, err := NewClient(WithHTTPClient(server.Client()), WithBaseURL(server.URL+"/v3/company"))
	require.NoError(t, err)

	dept, err := client.Company("123", oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test-token"})).Departments().Get(context.Background(), "2")
	require.NoError(t, err)
	assert.Equal(t, "2", dept.ID())
	assert.Equal(t, EntityDepartment, dept.EntityName())
	assert.Equal(t, "Active", dept.CDCStatus())
	assert.Equal(t, "QBO", dept.CDCDomain())
	assert.Equal(t, "Marketing Department", dept.Name)
	assert.Equal(t, "Marketing Department", dept.FullyQualifiedName)
}

func TestDepartmentServiceQueryUsesQueryOptions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/company/123/query", r.URL.Path)
		assert.Equal(t, "SELECT * FROM Department STARTPOSITION 11 MAXRESULTS 10", r.URL.Query().Get("query"))
		w.Header().Set("Content-Type", "application/json")
		_, err := io.WriteString(w, `{"QueryResponse":{"Department":[{"Id":"2","SyncToken":"0"},{"Id":"3","SyncToken":"1"}],"startPosition":6,"maxResults":10}}`)
		require.NoError(t, err)
	}))
	defer server.Close()

	client, err := NewClient(WithHTTPClient(server.Client()), WithBaseURL(server.URL+"/v3/company"))
	require.NoError(t, err)

	depts, err := client.Company("123", oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test-token"})).Departments().ListPage(context.Background(), 2, 10)
	require.NoError(t, err)
	require.Len(t, depts, 2)
	assert.Equal(t, "2", depts[0].ID())
	assert.Equal(t, "3", depts[1].ID())
}

func TestDepartmentJSONDecodesV75Fields(t *testing.T) {
	const fixture = `{
		"Department": {
			"Id": "2",
			"SyncToken": "0",
			"domain": "QBO",
			"status": "Active",
			"MetaData": {
				"CreateTime": "2013-08-13T11:52:48-07:00",
				"LastUpdatedTime": "2013-08-13T11:52:48-07:00"
			},
			"Name": "Marketing Department",
			"FullyQualifiedName": "Marketing Department",
			"SubDepartment": false,
			"Active": true
		}
	}`

	var envelope struct {
		Department Department `json:"Department"`
	}
	require.NoError(t, json.Unmarshal([]byte(fixture), &envelope))

	dept := envelope.Department
	assert.Equal(t, "2", dept.ID())
	assert.Equal(t, "0", dept.SyncToken())
	assert.Equal(t, "QBO", dept.CDCDomain())
	assert.Equal(t, "Active", dept.CDCStatus())

	require.NotNil(t, dept.MetaData)
	require.NotNil(t, dept.MetaData.CreateTime)

	assert.Equal(t, "Marketing Department", dept.Name)
	assert.Equal(t, "Marketing Department", dept.FullyQualifiedName)
	require.NotNil(t, dept.SubDepartment)
	assert.False(t, *dept.SubDepartment)
	require.NotNil(t, dept.Active)
	assert.True(t, *dept.Active)
}

func TestDepartmentJSONDecodesSubDepartment(t *testing.T) {
	const fixture = `{
		"Department": {
			"Id": "5",
			"SyncToken": "0",
			"domain": "QBO",
			"status": "Active",
			"Name": "East",
			"FullyQualifiedName": "Marketing Department:East",
			"SubDepartment": true,
			"Active": true,
			"ParentRef": {"value": "2", "name": "Marketing Department"}
		}
	}`

	var envelope struct {
		Department Department `json:"Department"`
	}
	require.NoError(t, json.Unmarshal([]byte(fixture), &envelope))

	dept := envelope.Department
	assert.Equal(t, "5", dept.ID())
	assert.Equal(t, "Marketing Department:East", dept.FullyQualifiedName)
	require.NotNil(t, dept.SubDepartment)
	assert.True(t, *dept.SubDepartment)
	require.NotNil(t, dept.ParentRef)
	assert.Equal(t, "2", dept.ParentRef.Value)
	assert.Equal(t, "Marketing Department", dept.ParentRef.Name)
}

func TestDepartmentRequestsMarshalWritablePayloads(t *testing.T) {
	create := Department{
		Name: "Marketing Department",
	}

	body, err := json.Marshal(create)
	require.NoError(t, err)
	assert.Contains(t, string(body), `"Name":"Marketing Department"`)
	assert.NotContains(t, string(body), `"FullyQualifiedName"`)
	assert.NotContains(t, string(body), `"SubDepartment"`)
	assert.NotContains(t, string(body), `"sparse"`)
}

func TestDepartmentCDCExtraction(t *testing.T) {
	changedSince := time.Date(2026, time.March, 10, 12, 0, 0, 0, time.UTC)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/company/123/cdc", r.URL.Path)
		assert.Equal(t, string(EntityDepartment), r.URL.Query().Get("entities"))
		assert.Equal(t, changedSince.Format(time.RFC3339), r.URL.Query().Get("changedSince"))
		w.Header().Set("Content-Type", "application/json")
		_, err := io.WriteString(w, `{"CDCResponse":[{"QueryResponse":[{"Department":[{"Id":"2","SyncToken":"1","domain":"QBO","status":"Deleted"}],"startPosition":1,"maxResults":1}]}],"time":"2026-03-10T12:00:00Z"}`)
		require.NoError(t, err)
	}))
	defer server.Close()

	client, err := NewClient(WithHTTPClient(server.Client()), WithBaseURL(server.URL+"/v3/company"))
	require.NoError(t, err)

	depts, err := client.Company("123", oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test-token"})).Departments().ChangeDataCapture(context.Background(), changedSince)
	require.NoError(t, err)
	require.Len(t, depts, 1)
	assert.Equal(t, "2", depts[0].ID())
	assert.Equal(t, "Deleted", depts[0].CDCStatus())
}

// Compile-time interface compliance checks for Department and DepartmentService.
var (
	_ Entity    = Department{}
	_ CDCEntity = Department{}

	_ Getter[Department]          = (*DepartmentService)(nil)
	_ Querier[Department]         = (*DepartmentService)(nil)
	_ Lister[Department]          = (*DepartmentService)(nil)
	_ Pager[Department]           = (*DepartmentService)(nil)
	_ Creator[Department]         = (*DepartmentService)(nil)
	_ Updater[Department]         = (*DepartmentService)(nil)
	_ SparseUpdater[Department]   = (*DepartmentService)(nil)
	_ EntityCDCReader[Department] = (*DepartmentService)(nil)
)

func TestDepartmentServiceUpdateSparseInjectsSparseFlag(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/company/123/department", r.URL.Path)
		var payload map[string]interface{}
		require.NoError(t, json.NewDecoder(r.Body).Decode(&payload))
		assert.Equal(t, true, payload["sparse"])
		w.Header().Set("Content-Type", "application/json")
		_, err := io.WriteString(w, `{"Department":{"Id":"1","SyncToken":"2"}}`)
		require.NoError(t, err)
	}))
	defer server.Close()

	client, err := NewClient(WithHTTPClient(server.Client()), WithBaseURL(server.URL+"/v3/company"))
	require.NoError(t, err)

	dept := Department{}
	dept.Id = "1"
	dept.SyncTokenValue = SyncTokenValue{"1"}

	_, err = client.Company("123", oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test-token"})).Departments().UpdateSparse(context.Background(), dept)
	require.NoError(t, err)
}
