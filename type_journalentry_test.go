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

func TestJournalEntryServiceGet(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/company/123/journalentry/227", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_, err := io.WriteString(w, `{"JournalEntry":{"Id":"227","SyncToken":"0","domain":"QBO","status":"Active","TxnDate":"2015-06-29"}}`)
		require.NoError(t, err)
	}))
	defer server.Close()

	client, err := NewClient(WithHTTPClient(server.Client()), WithBaseURL(server.URL+"/v3/company"))
	require.NoError(t, err)

	je, err := client.Company("123", oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test-token"})).JournalEntries().Get(context.Background(), "227")
	require.NoError(t, err)
	assert.Equal(t, "227", je.ID())
	assert.Equal(t, EntityJournalEntry, je.EntityName())
	assert.Equal(t, "Active", je.CDCStatus())
	assert.Equal(t, "QBO", je.CDCDomain())
}

func TestJournalEntryServiceQueryUsesQueryOptions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/company/123/query", r.URL.Path)
		assert.Equal(t, "SELECT * FROM JournalEntry STARTPOSITION 11 MAXRESULTS 10", r.URL.Query().Get("query"))
		w.Header().Set("Content-Type", "application/json")
		_, err := io.WriteString(w, `{"QueryResponse":{"JournalEntry":[{"Id":"227","SyncToken":"0"},{"Id":"228","SyncToken":"1"}],"startPosition":6,"maxResults":10}}`)
		require.NoError(t, err)
	}))
	defer server.Close()

	client, err := NewClient(WithHTTPClient(server.Client()), WithBaseURL(server.URL+"/v3/company"))
	require.NoError(t, err)

	jes, err := client.Company("123", oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test-token"})).JournalEntries().ListPage(context.Background(), 2, 10)
	require.NoError(t, err)
	require.Len(t, jes, 2)
	assert.Equal(t, "227", jes[0].ID())
	assert.Equal(t, "228", jes[1].ID())
}

func TestJournalEntryJSONDecodesV75Fields(t *testing.T) {
	const fixture = `{
		"JournalEntry": {
			"Id": "227",
			"SyncToken": "0",
			"domain": "QBO",
			"status": "Active",
			"MetaData": {
				"CreateTime": "2015-06-29T12:33:57-07:00",
				"LastUpdatedTime": "2015-06-29T12:33:57-07:00"
			},
			"TxnDate": "2015-06-29",
			"DocNumber": "JE-001",
			"PrivateNote": "Sprinkler repair adjustment",
			"Adjustment": false,
			"TotalAmt": 25.54,
			"Line": [
				{
					"Id": "0",
					"Description": "Four sprinkler heads damaged",
					"Amount": 25.54,
					"DetailType": "JournalEntryLineDetail",
					"ProjectRef": {"value": "39298034"},
					"JournalEntryLineDetail": {
						"PostingType": "Debit",
						"AccountRef": {"value": "65", "name": "Job Expenses:Job Materials:Fountain and Garden Lighting"},
						"ClassRef": {"value": "3"},
						"Entity": {
							"Type": "Customer",
							"EntityRef": {"value": "1", "name": "Amy's Bird Sanctuary"}
						}
					}
				},
				{
					"Id": "1",
					"Description": "Sprinkler Hds - Inventory Adjustment",
					"Amount": 25.54,
					"DetailType": "JournalEntryLineDetail",
					"JournalEntryLineDetail": {
						"PostingType": "Credit",
						"AccountRef": {"value": "44", "name": "Notes Payable"},
						"DepartmentRef": {"value": "2"},
						"Entity": {
							"Type": "Vendor",
							"EntityRef": {"value": "2", "name": "IDX Vendor"}
						}
					}
				},
				{
					"Id": "2",
					"Description": "Memo line only",
					"DetailType": "DescriptionOnly",
					"DescriptionLineDetail": {
						"TaxCodeRef": {"value": "NON"}
					}
				}
			],
			"TxnTaxDetail": {
				"TotalTax": 0
			}
		}
	}`

	var envelope struct {
		JournalEntry JournalEntry `json:"JournalEntry"`
	}
	require.NoError(t, json.Unmarshal([]byte(fixture), &envelope))

	je := envelope.JournalEntry
	assert.Equal(t, "227", je.ID())
	assert.Equal(t, "0", je.SyncToken())
	assert.Equal(t, "QBO", je.CDCDomain())
	assert.Equal(t, "Active", je.CDCStatus())

	require.NotNil(t, je.MetaData)
	require.NotNil(t, je.MetaData.CreateTime)

	require.NotNil(t, je.TxnDate)
	assert.Equal(t, "2015-06-29", je.TxnDate.String())

	assert.Equal(t, "JE-001", je.DocNumber)
	assert.Equal(t, "Sprinkler repair adjustment", je.PrivateNote)
	require.NotNil(t, je.Adjustment)
	assert.False(t, *je.Adjustment)
	assert.Equal(t, "25.54", je.TotalAmt.String())

	require.Len(t, je.Line, 3)

	// Debit line with Customer entity and ProjectRef
	line0 := je.Line[0]
	assert.Equal(t, JournalEntryLineDetailType, line0.DetailType)
	assert.Equal(t, "Four sprinkler heads damaged", line0.Description)
	assert.Equal(t, "25.54", line0.Amount.String())
	require.NotNil(t, line0.ProjectRef)
	assert.Equal(t, "39298034", line0.ProjectRef.Value)
	require.NotNil(t, line0.JournalEntryLineDetail)
	assert.Equal(t, DebitPostingType, line0.JournalEntryLineDetail.PostingType)
	assert.Equal(t, "65", line0.JournalEntryLineDetail.AccountRef.Value)
	assert.Equal(t, "3", line0.JournalEntryLineDetail.ClassRef.Value)
	require.NotNil(t, line0.JournalEntryLineDetail.Entity)
	assert.Equal(t, "Customer", line0.JournalEntryLineDetail.Entity.Type)
	assert.Equal(t, "1", line0.JournalEntryLineDetail.Entity.EntityRef.Value)

	// Credit line with Vendor entity and DepartmentRef
	line1 := je.Line[1]
	assert.Equal(t, JournalEntryLineDetailType, line1.DetailType)
	require.NotNil(t, line1.JournalEntryLineDetail)
	assert.Equal(t, CreditPostingType, line1.JournalEntryLineDetail.PostingType)
	assert.Equal(t, "44", line1.JournalEntryLineDetail.AccountRef.Value)
	assert.Equal(t, "2", line1.JournalEntryLineDetail.DepartmentRef.Value)
	require.NotNil(t, line1.JournalEntryLineDetail.Entity)
	assert.Equal(t, "Vendor", line1.JournalEntryLineDetail.Entity.Type)
	assert.Equal(t, "2", line1.JournalEntryLineDetail.Entity.EntityRef.Value)

	// DescriptionOnly line
	line2 := je.Line[2]
	assert.Equal(t, DescriptionOnlyLineDetailType, line2.DetailType)
	assert.Equal(t, "Memo line only", line2.Description)
	require.NotNil(t, line2.DescriptionLineDetail)
	assert.Equal(t, "NON", line2.DescriptionLineDetail.TaxCodeRef.Value)

	// TxnTaxDetail
	require.NotNil(t, je.TxnTaxDetail)
	assert.Equal(t, "0", je.TxnTaxDetail.TotalTax.String())
}

func TestJournalEntryRequestsMarshalWritablePayloads(t *testing.T) {
	create := JournalEntry{
		Line: []JournalEntryLine{
			{
				Amount:     json.Number("100"),
				DetailType: JournalEntryLineDetailType,
				JournalEntryLineDetail: &JournalEntryLineDetail{
					PostingType: DebitPostingType,
					AccountRef:  &ReferenceType{Value: "39"},
				},
			},
			{
				Amount:     json.Number("100"),
				DetailType: JournalEntryLineDetailType,
				JournalEntryLineDetail: &JournalEntryLineDetail{
					PostingType: CreditPostingType,
					AccountRef:  &ReferenceType{Value: "44"},
				},
			},
		},
	}

	body, err := json.Marshal(create)
	require.NoError(t, err)
	assert.Contains(t, string(body), `"JournalEntryLineDetail"`)
	assert.Contains(t, string(body), `"PostingType":"Debit"`)
	assert.Contains(t, string(body), `"PostingType":"Credit"`)
	assert.NotContains(t, string(body), `"sparse"`)
	assert.NotContains(t, string(body), `"TotalAmt"`)
}

func TestJournalEntryServiceUpdateSparseInjectsSparseFlag(t *testing.T) {
	var capturedBody []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/company/123/journalentry", r.URL.Path)
		var err error
		capturedBody, err = io.ReadAll(r.Body)
		require.NoError(t, err)
		w.Header().Set("Content-Type", "application/json")
		_, err = io.WriteString(w, `{"JournalEntry":{"Id":"227","SyncToken":"1"}}`)
		require.NoError(t, err)
	}))
	defer server.Close()

	client, err := NewClient(WithHTTPClient(server.Client()), WithBaseURL(server.URL+"/v3/company"))
	require.NoError(t, err)

	je := JournalEntry{
		IDValue:        IDValue{Id: "227"},
		SyncTokenValue: SyncTokenValue{SyncTokenValue: "0"},
		PrivateNote:    "Revised private note via sparse update",
		TxnDate:        &Date{Time: time.Date(2015, 11, 30, 0, 0, 0, 0, time.UTC)},
	}

	_, err = client.Company("123", oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test-token"})).JournalEntries().UpdateSparse(context.Background(), je)
	require.NoError(t, err)

	assert.Contains(t, string(capturedBody), `"Id":"227"`)
	assert.Contains(t, string(capturedBody), `"SyncToken":"0"`)
	assert.Contains(t, string(capturedBody), `"sparse":true`)
	assert.Contains(t, string(capturedBody), `"PrivateNote":"Revised private note via sparse update"`)
}

func TestJournalEntryCDCExtraction(t *testing.T) {
	changedSince := time.Date(2026, time.March, 10, 12, 0, 0, 0, time.UTC)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/company/123/cdc", r.URL.Path)
		assert.Equal(t, string(EntityJournalEntry), r.URL.Query().Get("entities"))
		assert.Equal(t, changedSince.Format(time.RFC3339), r.URL.Query().Get("changedSince"))
		w.Header().Set("Content-Type", "application/json")
		_, err := io.WriteString(w, `{"CDCResponse":[{"QueryResponse":[{"JournalEntry":[{"Id":"227","SyncToken":"1","domain":"QBO","status":"Deleted"}],"startPosition":1,"maxResults":1}]}],"time":"2026-03-10T12:00:00Z"}`)
		require.NoError(t, err)
	}))
	defer server.Close()

	client, err := NewClient(WithHTTPClient(server.Client()), WithBaseURL(server.URL+"/v3/company"))
	require.NoError(t, err)

	jes, err := client.Company("123", oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test-token"})).JournalEntries().ChangeDataCapture(context.Background(), changedSince)
	require.NoError(t, err)
	require.Len(t, jes, 1)
	assert.Equal(t, "227", jes[0].ID())
	assert.Equal(t, "Deleted", jes[0].CDCStatus())
}

// Compile-time interface compliance checks for JournalEntry and JournalEntryService.
var (
	// JournalEntry entity type compliance.
	_ Entity    = JournalEntry{}
	_ CDCEntity = JournalEntry{}

	// JournalEntryService operation compliance.
	_ Getter[JournalEntry]          = (*JournalEntryService)(nil)
	_ Querier[JournalEntry]         = (*JournalEntryService)(nil)
	_ Lister[JournalEntry]          = (*JournalEntryService)(nil)
	_ Pager[JournalEntry]           = (*JournalEntryService)(nil)
	_ Creator[JournalEntry]         = (*JournalEntryService)(nil)
	_ Updater[JournalEntry]         = (*JournalEntryService)(nil)
	_ SparseUpdater[JournalEntry]   = (*JournalEntryService)(nil)
	_ Deleter[JournalEntry]         = (*JournalEntryService)(nil)
	_ EntityCDCReader[JournalEntry] = (*JournalEntryService)(nil)
)
