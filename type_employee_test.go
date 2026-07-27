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

func TestEmployeeServiceGet(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/company/123/employee/71", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_, err := io.WriteString(w, `{"Employee":{"Id":"71","SyncToken":"0","domain":"QBO","status":"Active","DisplayName":"Bill Miller","GivenName":"Bill","FamilyName":"Miller"}}`)
		require.NoError(t, err)
	}))
	defer server.Close()

	client, err := NewClient(WithHTTPClient(server.Client()), WithBaseURL(server.URL+"/v3/company"))
	require.NoError(t, err)

	emp, err := client.Company("123", oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test-token"})).Employees().Get(context.Background(), "71")
	require.NoError(t, err)
	assert.Equal(t, "71", emp.ID())
	assert.Equal(t, EntityEmployee, emp.EntityName())
	assert.Equal(t, "Active", emp.CDCStatus())
	assert.Equal(t, "QBO", emp.CDCDomain())
	assert.Equal(t, "Bill Miller", emp.DisplayName)
}

func TestEmployeeServiceQueryUsesQueryOptions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/company/123/query", r.URL.Path)
		assert.Equal(t, "SELECT * FROM Employee STARTPOSITION 11 MAXRESULTS 10", r.URL.Query().Get("query"))
		w.Header().Set("Content-Type", "application/json")
		_, err := io.WriteString(w, `{"QueryResponse":{"Employee":[{"Id":"71","SyncToken":"0"},{"Id":"72","SyncToken":"1"}],"startPosition":6,"maxResults":10}}`)
		require.NoError(t, err)
	}))
	defer server.Close()

	client, err := NewClient(WithHTTPClient(server.Client()), WithBaseURL(server.URL+"/v3/company"))
	require.NoError(t, err)

	emps, err := client.Company("123", oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test-token"})).Employees().ListPage(context.Background(), 2, 10)
	require.NoError(t, err)
	require.Len(t, emps, 2)
	assert.Equal(t, "71", emps[0].ID())
	assert.Equal(t, "72", emps[1].ID())
}

func TestEmployeeJSONDecodesV75Fields(t *testing.T) {
	const fixture = `{
		"Employee": {
			"Id": "71",
			"SyncToken": "0",
			"domain": "QBO",
			"status": "Active",
			"MetaData": {
				"CreateTime": "2015-07-24T09:34:35-07:00",
				"LastUpdatedTime": "2015-07-24T09:34:35-07:00"
			},
			"DisplayName": "Bill Miller",
			"Title": "Mr",
			"GivenName": "Bill",
			"MiddleName": "A",
			"FamilyName": "Miller",
			"Suffix": "Jr",
			"PrintOnCheckName": "Bill Miller",
			"EmployeeNumber": "EMP-001",
			"SSN": "XXX-XX-XXXX",
			"Gender": "Male",
			"BirthDate": "1980-05-15",
			"HiredDate": "2010-01-01",
			"Active": true,
			"BillableTime": false,
			"Organization": false,
			"PrimaryPhone": {"FreeFormNumber": "234-525-1234"},
			"Mobile": {"FreeFormNumber": "415-555-9999"},
			"PrimaryEmailAddr": {"Address": "bill.miller@example.com"},
			"CostRate": 50.00,
			"BillRate": 75.00,
			"PrimaryAddr": {
				"Line1": "45 N. Elm Street",
				"City": "Middlefield",
				"CountrySubDivisionCode": "CA",
				"PostalCode": "93242",
				"Id": "116"
			},
			"V4IDPseudonym": "abc123def456"
		}
	}`

	var envelope struct {
		Employee Employee `json:"Employee"`
	}
	require.NoError(t, json.Unmarshal([]byte(fixture), &envelope))

	emp := envelope.Employee
	assert.Equal(t, "71", emp.ID())
	assert.Equal(t, "0", emp.SyncToken())
	assert.Equal(t, "QBO", emp.CDCDomain())
	assert.Equal(t, "Active", emp.CDCStatus())

	require.NotNil(t, emp.MetaData)
	require.NotNil(t, emp.MetaData.CreateTime)

	assert.Equal(t, "Bill Miller", emp.DisplayName)
	assert.Equal(t, "Mr", emp.Title)
	assert.Equal(t, "Bill", emp.GivenName)
	assert.Equal(t, "A", emp.MiddleName)
	assert.Equal(t, "Miller", emp.FamilyName)
	assert.Equal(t, "Jr", emp.Suffix)
	assert.Equal(t, "Bill Miller", emp.PrintOnCheckName)
	assert.Equal(t, "EMP-001", emp.EmployeeNumber)
	assert.Equal(t, "XXX-XX-XXXX", emp.SSN)
	assert.Equal(t, "Male", emp.Gender)

	require.NotNil(t, emp.BirthDate)
	assert.Equal(t, "1980-05-15", emp.BirthDate.String())
	require.NotNil(t, emp.HiredDate)
	assert.Equal(t, "2010-01-01", emp.HiredDate.String())

	require.NotNil(t, emp.Active)
	assert.True(t, *emp.Active)
	require.NotNil(t, emp.BillableTime)
	assert.False(t, *emp.BillableTime)
	require.NotNil(t, emp.Organization)
	assert.False(t, *emp.Organization)

	require.NotNil(t, emp.PrimaryPhone)
	assert.Equal(t, "234-525-1234", emp.PrimaryPhone.FreeFormNumber)
	require.NotNil(t, emp.Mobile)
	assert.Equal(t, "415-555-9999", emp.Mobile.FreeFormNumber)

	require.NotNil(t, emp.PrimaryEmailAddr)
	assert.Equal(t, "bill.miller@example.com", emp.PrimaryEmailAddr.Address)

	assert.Equal(t, "50.00", emp.CostRate.String())
	assert.Equal(t, "75.00", emp.BillRate.String())

	require.NotNil(t, emp.PrimaryAddr)
	assert.Equal(t, "45 N. Elm Street", emp.PrimaryAddr.Line1)
	assert.Equal(t, "Middlefield", emp.PrimaryAddr.City)

	assert.Equal(t, "abc123def456", emp.V4IDPseudonym)
}

func TestEmployeeRequestsMarshalWritablePayloads(t *testing.T) {
	create := Employee{
		GivenName:  "John",
		FamilyName: "Meuller",
		SSN:        "444-55-6666",
		PrimaryAddr: &PhysicalAddress{
			Line1:                  "45 N. Elm Street",
			City:                   "Middlefield",
			CountrySubDivisionCode: "CA",
			PostalCode:             "93242",
		},
		PrimaryPhone: &TelephoneNumber{
			FreeFormNumber: "408-525-1234",
		},
	}

	body, err := json.Marshal(create)
	require.NoError(t, err)
	assert.Contains(t, string(body), `"GivenName"`)
	assert.Contains(t, string(body), `"PrimaryAddr"`)
	assert.NotContains(t, string(body), `"sparse"`)
	assert.NotContains(t, string(body), `"V4IDPseudonym"`)
}

func TestEmployeeCDCExtraction(t *testing.T) {
	changedSince := time.Date(2026, time.March, 10, 12, 0, 0, 0, time.UTC)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/company/123/cdc", r.URL.Path)
		assert.Equal(t, string(EntityEmployee), r.URL.Query().Get("entities"))
		assert.Equal(t, changedSince.Format(time.RFC3339), r.URL.Query().Get("changedSince"))
		w.Header().Set("Content-Type", "application/json")
		_, err := io.WriteString(w, `{"CDCResponse":[{"QueryResponse":[{"Employee":[{"Id":"71","SyncToken":"1","domain":"QBO","status":"Deleted"}],"startPosition":1,"maxResults":1}]}],"time":"2026-03-10T12:00:00Z"}`)
		require.NoError(t, err)
	}))
	defer server.Close()

	client, err := NewClient(WithHTTPClient(server.Client()), WithBaseURL(server.URL+"/v3/company"))
	require.NoError(t, err)

	emps, err := client.Company("123", oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test-token"})).Employees().ChangeDataCapture(context.Background(), changedSince)
	require.NoError(t, err)
	require.Len(t, emps, 1)
	assert.Equal(t, "71", emps[0].ID())
	assert.Equal(t, "Deleted", emps[0].CDCStatus())
}

// Compile-time interface compliance checks for Employee and EmployeeService.
var (
	_ Entity    = Employee{}
	_ CDCEntity = Employee{}

	_ Getter[Employee]          = (*EmployeeService)(nil)
	_ Querier[Employee]         = (*EmployeeService)(nil)
	_ Lister[Employee]          = (*EmployeeService)(nil)
	_ Pager[Employee]           = (*EmployeeService)(nil)
	_ Creator[Employee]         = (*EmployeeService)(nil)
	_ Updater[Employee]         = (*EmployeeService)(nil)
	_ SparseUpdater[Employee]   = (*EmployeeService)(nil)
	_ EntityCDCReader[Employee] = (*EmployeeService)(nil)
)

func TestEmployeeServiceUpdateSparseInjectsSparseFlag(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/company/123/employee", r.URL.Path)
		var payload map[string]interface{}
		require.NoError(t, json.NewDecoder(r.Body).Decode(&payload))
		assert.Equal(t, true, payload["sparse"])
		w.Header().Set("Content-Type", "application/json")
		_, err := io.WriteString(w, `{"Employee":{"Id":"1","SyncToken":"2"}}`)
		require.NoError(t, err)
	}))
	defer server.Close()

	client, err := NewClient(WithHTTPClient(server.Client()), WithBaseURL(server.URL+"/v3/company"))
	require.NoError(t, err)

	employee := Employee{}
	employee.Id = "1"
	employee.SyncTokenValue = SyncTokenValue{"1"}

	_, err = client.Company("123", oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test-token"})).Employees().UpdateSparse(context.Background(), employee)
	require.NoError(t, err)
}
