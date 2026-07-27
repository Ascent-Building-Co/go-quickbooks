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

func TestTimeActivityServiceGet(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/company/123/timeactivity/42", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_, err := io.WriteString(w, `{"TimeActivity":{"Id":"42","SyncToken":"0","domain":"QBO","status":"Active","NameOf":"Employee","TxnDate":"2026-03-10","EmployeeRef":{"value":"10","name":"John Doe"},"CustomerRef":{"value":"20","name":"ACME Corp"},"HourlyRate":"75.00","Hours":8,"Minutes":0,"BillableStatus":"Billable","Description":"Development work"}}`)
		require.NoError(t, err)
	}))
	defer server.Close()

	client, err := NewClient(WithHTTPClient(server.Client()), WithBaseURL(server.URL+"/v3/company"))
	require.NoError(t, err)

	ta, err := client.Company("123", oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test-token"})).TimeActivities().Get(context.Background(), "42")
	require.NoError(t, err)
	assert.Equal(t, "42", ta.ID())
	assert.Equal(t, EntityTimeActivity, ta.EntityName())
	assert.Equal(t, NameOfEmployee, ta.NameOf)
	assert.Equal(t, 8, ta.Hours)
	assert.Equal(t, 0, ta.Minutes)
	assert.Equal(t, "75.00", ta.HourlyRate.String())
	assert.Equal(t, BillableBillableStatus, ta.BillableStatus)
	assert.Equal(t, "Development work", ta.Description)
	require.NotNil(t, ta.TxnDate)
	require.NotNil(t, ta.EmployeeRef)
	assert.Equal(t, "10", ta.EmployeeRef.Value)
	assert.Equal(t, "John Doe", ta.EmployeeRef.Name)
	require.NotNil(t, ta.CustomerRef)
	assert.Equal(t, "20", ta.CustomerRef.Value)
}

func TestTimeActivityServiceDelete(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/company/123/timeactivity", r.URL.Path)
		assert.Equal(t, "delete", r.URL.Query().Get("operation"))
		var payload map[string]interface{}
		require.NoError(t, json.NewDecoder(r.Body).Decode(&payload))
		assert.Equal(t, "42", payload["Id"])
		assert.Equal(t, "3", payload["SyncToken"])
		w.Header().Set("Content-Type", "application/json")
		_, err := io.WriteString(w, `{"TimeActivity":{"Id":"42","SyncToken":"3","status":"Deleted"}}`)
		require.NoError(t, err)
	}))
	defer server.Close()

	client, err := NewClient(WithHTTPClient(server.Client()), WithBaseURL(server.URL+"/v3/company"))
	require.NoError(t, err)

	req := DeleteRequest{}
	req.IDValue.Id = "42"
	req.SyncTokenValue.SyncTokenValue = "3"

	ta, err := client.Company("123", oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test-token"})).TimeActivities().Delete(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, "42", ta.ID())
}

func TestTimeActivityServiceListPage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/company/123/query", r.URL.Path)
		assert.Equal(t, "SELECT * FROM TimeActivity STARTPOSITION 1 MAXRESULTS 5", r.URL.Query().Get("query"))
		w.Header().Set("Content-Type", "application/json")
		_, err := io.WriteString(w, `{"QueryResponse":{"TimeActivity":[{"Id":"42","SyncToken":"0","NameOf":"Employee","Hours":8},{"Id":"43","SyncToken":"0","NameOf":"Vendor","Hours":4}],"startPosition":1,"maxResults":2}}`)
		require.NoError(t, err)
	}))
	defer server.Close()

	client, err := NewClient(WithHTTPClient(server.Client()), WithBaseURL(server.URL+"/v3/company"))
	require.NoError(t, err)

	tas, err := client.Company("123", oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test-token"})).TimeActivities().ListPage(context.Background(), 1, 5)
	require.NoError(t, err)
	require.Len(t, tas, 2)
	assert.Equal(t, "42", tas[0].ID())
	assert.Equal(t, NameOfEmployee, tas[0].NameOf)
	assert.Equal(t, 8, tas[0].Hours)
	assert.Equal(t, "43", tas[1].ID())
	assert.Equal(t, NameOfVendor, tas[1].NameOf)
}

func TestTimeActivityJSONDecodesV75Fields(t *testing.T) {
	const fixture = `{
		"TimeActivity": {
			"Id": "42",
			"SyncToken": "1",
			"domain": "QBO",
			"status": "Active",
			"MetaData": {
				"CreateTime": "2026-03-10T09:00:00-07:00",
				"LastUpdatedTime": "2026-03-10T09:00:00-07:00"
			},
			"NameOf": "Employee",
			"TxnDate": "2026-03-10",
			"EmployeeRef": {"value": "10", "name": "John Doe"},
			"CustomerRef": {"value": "20", "name": "ACME Corp"},
			"ClassRef": {"value": "5", "name": "Engineering"},
			"ItemRef": {"value": "8", "name": "Dev Services"},
			"DepartmentRef": {"value": "2", "name": "Engineering"},
			"PayrollItemRef": {"value": "3", "name": "Regular Pay"},
			"HourlyRate": "75.00",
			"StartTime": "2026-03-10T09:00:00-07:00",
			"EndTime": "2026-03-10T17:00:00-07:00",
			"BreakHours": 0,
			"BreakMinutes": 30,
			"BillableStatus": "Billable",
			"CostRate": "50.00",
			"Taxable": true,
			"Description": "Development work",
			"TransactionLocationType": "WithinFrance"
		}
	}`

	var envelope struct {
		TimeActivity TimeActivity `json:"TimeActivity"`
	}
	require.NoError(t, json.Unmarshal([]byte(fixture), &envelope))

	ta := envelope.TimeActivity
	assert.Equal(t, "42", ta.ID())
	assert.Equal(t, "1", ta.SyncToken())

	require.NotNil(t, ta.MetaData)
	require.NotNil(t, ta.MetaData.CreateTime)

	assert.Equal(t, NameOfEmployee, ta.NameOf)
	require.NotNil(t, ta.TxnDate)
	require.NotNil(t, ta.EmployeeRef)
	assert.Equal(t, "10", ta.EmployeeRef.Value)
	require.NotNil(t, ta.CustomerRef)
	assert.Equal(t, "20", ta.CustomerRef.Value)
	require.NotNil(t, ta.ClassRef)
	assert.Equal(t, "5", ta.ClassRef.Value)
	require.NotNil(t, ta.ItemRef)
	assert.Equal(t, "8", ta.ItemRef.Value)
	require.NotNil(t, ta.DepartmentRef)
	assert.Equal(t, "2", ta.DepartmentRef.Value)
	require.NotNil(t, ta.PayrollItemRef)
	assert.Equal(t, "3", ta.PayrollItemRef.Value)

	assert.Equal(t, "75.00", ta.HourlyRate.String())
	assert.Equal(t, "50.00", ta.CostRate.String())

	require.NotNil(t, ta.StartTime)
	require.NotNil(t, ta.EndTime)
	assert.Equal(t, 0, ta.BreakHours)
	assert.Equal(t, 30, ta.BreakMinutes)

	assert.Equal(t, BillableBillableStatus, ta.BillableStatus)
	require.NotNil(t, ta.Taxable)
	assert.True(t, *ta.Taxable)
	assert.Equal(t, "Development work", ta.Description)
	assert.Equal(t, "WithinFrance", ta.TransactionLocationType)
}

func TestTimeActivityRequestsMarshalWritablePayloads(t *testing.T) {
	create := TimeActivity{
		NameOf: NameOfEmployee,
		Hours:  8,
	}
	create.EmployeeRef = &ReferenceType{Value: "10"}
	create.CustomerRef = &ReferenceType{Value: "20"}

	body, err := json.Marshal(create)
	require.NoError(t, err)
	assert.Contains(t, string(body), `"NameOf":"Employee"`)
	assert.Contains(t, string(body), `"Hours":8`)
	assert.NotContains(t, string(body), `"sparse"`)
}

// Compile-time interface compliance checks for TimeActivity and TimeActivityService.
var (
	_ Entity = TimeActivity{}

	_ Getter[TimeActivity]        = (*TimeActivityService)(nil)
	_ Querier[TimeActivity]       = (*TimeActivityService)(nil)
	_ Lister[TimeActivity]        = (*TimeActivityService)(nil)
	_ Pager[TimeActivity]         = (*TimeActivityService)(nil)
	_ Creator[TimeActivity]       = (*TimeActivityService)(nil)
	_ Updater[TimeActivity]       = (*TimeActivityService)(nil)
	_ SparseUpdater[TimeActivity] = (*TimeActivityService)(nil)
	_ Deleter[TimeActivity]       = (*TimeActivityService)(nil)
)

func TestTimeActivityServiceUpdateSparseInjectsSparseFlag(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/company/123/timeactivity", r.URL.Path)
		var payload map[string]interface{}
		require.NoError(t, json.NewDecoder(r.Body).Decode(&payload))
		assert.Equal(t, true, payload["sparse"])
		w.Header().Set("Content-Type", "application/json")
		_, err := io.WriteString(w, `{"TimeActivity":{"Id":"1","SyncToken":"2"}}`)
		require.NoError(t, err)
	}))
	defer server.Close()

	client, err := NewClient(WithHTTPClient(server.Client()), WithBaseURL(server.URL+"/v3/company"))
	require.NoError(t, err)

	ta := TimeActivity{}
	ta.Id = "1"
	ta.SyncTokenValue = SyncTokenValue{"1"}

	_, err = client.Company("123", oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test-token"})).TimeActivities().UpdateSparse(context.Background(), ta)
	require.NoError(t, err)
}
