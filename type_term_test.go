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

func TestTermServiceGet(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/company/123/term/3", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_, err := io.WriteString(w, `{"Term":{"Id":"3","SyncToken":"0","domain":"QBO","status":"Active","Name":"Net 30","Type":"STANDARD","DueDays":30,"Active":true}}`)
		require.NoError(t, err)
	}))
	defer server.Close()

	client, err := NewClient(WithHTTPClient(server.Client()), WithBaseURL(server.URL+"/v3/company"))
	require.NoError(t, err)

	term, err := client.Company("123", oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test-token"})).Terms().Get(context.Background(), "3")
	require.NoError(t, err)
	assert.Equal(t, "3", term.ID())
	assert.Equal(t, EntityTerm, term.EntityName())
	assert.Equal(t, "Active", term.CDCStatus())
	assert.Equal(t, "QBO", term.CDCDomain())
	assert.Equal(t, "Net 30", term.Name)
	assert.Equal(t, TermTypeStandard, term.Type)
	assert.Equal(t, 30, term.DueDays)
	require.NotNil(t, term.Active)
	assert.True(t, *term.Active)
}

func TestTermServiceListPage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/company/123/query", r.URL.Path)
		assert.Equal(t, "SELECT * FROM Term STARTPOSITION 1 MAXRESULTS 5", r.URL.Query().Get("query"))
		w.Header().Set("Content-Type", "application/json")
		_, err := io.WriteString(w, `{"QueryResponse":{"Term":[{"Id":"3","SyncToken":"0","Name":"Net 30","DueDays":30},{"Id":"4","SyncToken":"0","Name":"Net 15","DueDays":15}],"startPosition":1,"maxResults":2}}`)
		require.NoError(t, err)
	}))
	defer server.Close()

	client, err := NewClient(WithHTTPClient(server.Client()), WithBaseURL(server.URL+"/v3/company"))
	require.NoError(t, err)

	terms, err := client.Company("123", oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test-token"})).Terms().ListPage(context.Background(), 1, 5)
	require.NoError(t, err)
	require.Len(t, terms, 2)
	assert.Equal(t, "3", terms[0].ID())
	assert.Equal(t, "Net 30", terms[0].Name)
	assert.Equal(t, 30, terms[0].DueDays)
	assert.Equal(t, "4", terms[1].ID())
	assert.Equal(t, "Net 15", terms[1].Name)
}

func TestTermJSONDecodesV75Fields(t *testing.T) {
	const fixture = `{
		"Term": {
			"Id": "3",
			"SyncToken": "1",
			"domain": "QBO",
			"status": "Active",
			"MetaData": {
				"CreateTime": "2015-07-22T13:57:27-07:00",
				"LastUpdatedTime": "2015-07-22T13:57:27-07:00"
			},
			"Name": "2% 10 Net 30",
			"Type": "STANDARD",
			"DueDays": 30,
			"DiscountDays": 10,
			"DiscountPercent": "2.0",
			"Active": true
		}
	}`

	var envelope struct {
		Term Term `json:"Term"`
	}
	require.NoError(t, json.Unmarshal([]byte(fixture), &envelope))

	term := envelope.Term
	assert.Equal(t, "3", term.ID())
	assert.Equal(t, "1", term.SyncToken())
	assert.Equal(t, "QBO", term.CDCDomain())
	assert.Equal(t, "Active", term.CDCStatus())

	require.NotNil(t, term.MetaData)
	require.NotNil(t, term.MetaData.CreateTime)

	assert.Equal(t, "2% 10 Net 30", term.Name)
	assert.Equal(t, TermTypeStandard, term.Type)
	assert.Equal(t, 30, term.DueDays)
	assert.Equal(t, 10, term.DiscountDays)
	assert.Equal(t, "2.0", term.DiscountPercent.String())
	require.NotNil(t, term.Active)
	assert.True(t, *term.Active)
}

func TestTermDateDrivenJSONDecodes(t *testing.T) {
	const fixture = `{
		"Term": {
			"Id": "5",
			"SyncToken": "0",
			"Name": "Net 15th",
			"Type": "DATE_DRIVEN",
			"DayOfMonthDue": 15,
			"DueNextMonthDays": 20,
			"DiscountDayOfMonth": 5,
			"DiscountPercent": "1.5"
		}
	}`

	var envelope struct {
		Term Term `json:"Term"`
	}
	require.NoError(t, json.Unmarshal([]byte(fixture), &envelope))

	term := envelope.Term
	assert.Equal(t, "Net 15th", term.Name)
	assert.Equal(t, TermTypeDateDriven, term.Type)
	assert.Equal(t, 15, term.DayOfMonthDue)
	assert.Equal(t, 20, term.DueNextMonthDays)
	assert.Equal(t, 5, term.DiscountDayOfMonth)
	assert.Equal(t, "1.5", term.DiscountPercent.String())
}

func TestTermRequestsMarshalWritablePayloads(t *testing.T) {
	create := Term{
		Name:    "Net 30",
		DueDays: 30,
	}

	body, err := json.Marshal(create)
	require.NoError(t, err)
	assert.Contains(t, string(body), `"Name":"Net 30"`)
	assert.Contains(t, string(body), `"DueDays":30`)
	assert.NotContains(t, string(body), `"sparse"`)
}

func TestTermCDCExtraction(t *testing.T) {
	changedSince := time.Date(2026, time.March, 10, 12, 0, 0, 0, time.UTC)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/company/123/cdc", r.URL.Path)
		assert.Equal(t, string(EntityTerm), r.URL.Query().Get("entities"))
		assert.Equal(t, changedSince.Format(time.RFC3339), r.URL.Query().Get("changedSince"))
		w.Header().Set("Content-Type", "application/json")
		_, err := io.WriteString(w, `{"CDCResponse":[{"QueryResponse":[{"Term":[{"Id":"3","SyncToken":"2","domain":"QBO","status":"Deleted"}],"startPosition":1,"maxResults":1}]}],"time":"2026-03-10T12:00:00Z"}`)
		require.NoError(t, err)
	}))
	defer server.Close()

	client, err := NewClient(WithHTTPClient(server.Client()), WithBaseURL(server.URL+"/v3/company"))
	require.NoError(t, err)

	terms, err := client.Company("123", oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test-token"})).Terms().ChangeDataCapture(context.Background(), changedSince)
	require.NoError(t, err)
	require.Len(t, terms, 1)
	assert.Equal(t, "3", terms[0].ID())
	assert.Equal(t, "Deleted", terms[0].CDCStatus())
}

// Compile-time interface compliance checks for Term and TermService.
var (
	_ Entity    = Term{}
	_ CDCEntity = Term{}

	_ Getter[Term]          = (*TermService)(nil)
	_ Querier[Term]         = (*TermService)(nil)
	_ Lister[Term]          = (*TermService)(nil)
	_ Pager[Term]           = (*TermService)(nil)
	_ Creator[Term]         = (*TermService)(nil)
	_ Updater[Term]         = (*TermService)(nil)
	_ SparseUpdater[Term]   = (*TermService)(nil)
	_ EntityCDCReader[Term] = (*TermService)(nil)
)

func TestTermServiceUpdateSparseInjectsSparseFlag(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/company/123/term", r.URL.Path)
		var payload map[string]interface{}
		require.NoError(t, json.NewDecoder(r.Body).Decode(&payload))
		assert.Equal(t, true, payload["sparse"])
		w.Header().Set("Content-Type", "application/json")
		_, err := io.WriteString(w, `{"Term":{"Id":"1","SyncToken":"2"}}`)
		require.NoError(t, err)
	}))
	defer server.Close()

	client, err := NewClient(WithHTTPClient(server.Client()), WithBaseURL(server.URL+"/v3/company"))
	require.NoError(t, err)

	term := Term{}
	term.Id = "1"
	term.SyncTokenValue = SyncTokenValue{"1"}

	_, err = client.Company("123", oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test-token"})).Terms().UpdateSparse(context.Background(), term)
	require.NoError(t, err)
}
