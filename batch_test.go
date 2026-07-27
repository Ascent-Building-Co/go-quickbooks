package quickbooks

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
)

func TestBatchRequestMarshalAndInvoiceExtraction(t *testing.T) {
	client, err := NewClient()
	require.NoError(t, err)

	service := client.Company("123", oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test-token"})).Invoices()
	batch := NewBatchRequest(
		service.BatchCreateItem("create-1", Invoice{DocNumber: "INV-1"}),
		service.BatchVoidItem("void-1", VoidRequest{IDValue: IDValue{Id: "3"}, SyncTokenValue: SyncTokenValue{SyncTokenValue: "2"}}),
		service.BatchQueryItem("query-1", "SELECT * FROM Invoice", WithPageSize(10)),
	)

	body, err := json.Marshal(batch)
	require.NoError(t, err)
	assert.Contains(t, string(body), `"operation":"create"`)
	assert.Contains(t, string(body), `"Invoice":{"DocNumber":"INV-1"`)
	assert.Contains(t, string(body), `"optionsData":"void"`)
	assert.Contains(t, string(body), `"Query":"SELECT * FROM Invoice MAXRESULTS 10"`)

	var resp BatchResponse
	err = json.Unmarshal([]byte(`{"BatchItemResponse":[{"bId":"create-1","Invoice":{"Id":"1","SyncToken":"0"}},{"bId":"void-1","Fault":{"type":"ValidationFault","Error":[{"Message":"invalid sync token","Detail":"stale object","code":"5010","element":"SyncToken"}]}},{"bId":"query-1","QueryResponse":{"Invoice":[{"Id":"2","SyncToken":"1"}]}}]}`), &resp)
	require.NoError(t, err)

	item := resp.Item("void-1")
	require.NotNil(t, item)
	assert.True(t, item.HasFaults())
	assert.Len(t, resp.Faults(), 1)
	assert.Equal(t, "void-1", resp.Faults()[0].ID)

	ok, bad := PartitionBatch(resp.BatchItemResponse)
	assert.Len(t, ok, 2)
	assert.Len(t, bad, 1)

	invoices, err := service.ExtractBatch(&resp)
	require.NoError(t, err)
	require.Len(t, invoices, 2)
	assert.Equal(t, "1", invoices[0].ID())
	assert.Equal(t, "2", invoices[1].ID())
}

func TestBatchRequestRequiresItemIDs(t *testing.T) {
	batch := NewBatchRequest(NewBatchQuery("", "SELECT * FROM Invoice"))
	rID := ""
	err := batch.Validate(&rID)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "batch item id is required")
}

func TestBatchRequestRequiresUniqueItemIDs(t *testing.T) {
	batch := NewBatchRequest(
		NewBatchQuery("dup-1", "SELECT * FROM Invoice"),
		NewBatchQuery("dup-1", "SELECT * FROM Invoice STARTPOSITION 2"),
	)

	rID := ""

	err := batch.Validate(&rID)
	require.Error(t, err)
	assert.Contains(t, err.Error(), `duplicate batch item id "dup-1"`)
}

func TestBatchRejectsMoreThanMaxItems(t *testing.T) {
	items := make([]BatchItemRequest, BatchMaxItems+1)
	for i := range items {
		items[i] = NewBatchQuery(fmt.Sprintf("q%d", i), "SELECT * FROM Invoice")
	}

	client, err := NewClient()
	require.NoError(t, err)

	_, err = client.Company("123", oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test-token"})).
		Batch(context.Background(), NewBatchRequest(items...))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "too many items")
}

func TestBatchItemIDExceedsAbsoluteMax(t *testing.T) {
	batch := NewBatchRequest(NewBatchQuery(strings.Repeat("x", RequestIDMaxLength+1), "SELECT * FROM Invoice"))
	rID := ""
	err := batch.Validate(&rID)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "exceeds maximum length of 50")
}

func TestBatchRequestValidateWithRequestID(t *testing.T) {
	// When a requestid is present, bIds are limited to 10 characters.
	batch := NewBatchRequest(NewBatchQuery("eleven-char", "SELECT * FROM Invoice"))
	rID := "some-req-id"
	err := batch.Validate(&rID)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "exceeds maximum length of 10")
}

func TestBatchDefaultRequestIDConstrainsBIds(t *testing.T) {
	// With no WithRequestID opt, a UUID is auto-generated (requestid present),
	// so bIds are limited to 10 characters.
	client, err := NewClient()
	require.NoError(t, err)

	batch := NewBatchRequest(NewBatchQuery("eleven-char", "SELECT * FROM Invoice")) // 11 chars
	_, err = client.Company("123", oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test-token"})).
		Batch(context.Background(), batch)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "exceeds maximum length of 10")
}

func TestBatchWithExplicitRequestIDConstrainsBIds(t *testing.T) {
	client, err := NewClient()
	require.NoError(t, err)

	batch := NewBatchRequest(NewBatchQuery("eleven-char", "SELECT * FROM Invoice")) // 11 chars
	_, err = client.Company("123", oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test-token"})).
		Batch(context.Background(), batch, WithRequestID("my-req-id"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "exceeds maximum length of 10")
}

func TestBatchWithNoRequestIDAllowsLongBIds(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Empty(t, r.URL.Query().Get("requestid"))

		var payload struct {
			BatchItemRequest []map[string]any `json:"BatchItemRequest"`
		}
		require.NoError(t, json.NewDecoder(r.Body).Decode(&payload))

		response := map[string]any{"BatchItemResponse": []map[string]any{}}
		items := response["BatchItemResponse"].([]map[string]any)
		for _, item := range payload.BatchItemRequest {
			id, _ := item["bId"].(string)
			items = append(items, map[string]any{"bId": id})
		}
		response["BatchItemResponse"] = items

		w.Header().Set("Content-Type", "application/json")
		require.NoError(t, json.NewEncoder(w).Encode(response))
	}))
	defer server.Close()

	client, err := NewClient(WithHTTPClient(server.Client()), WithBaseURL(server.URL+"/v3/company"))
	require.NoError(t, err)

	longID := strings.Repeat("x", RequestIDMaxLength) // 50 chars — valid when no requestid
	batch := NewBatchRequest(NewBatchQuery(longID, "SELECT * FROM Invoice"))
	resp, err := client.Company("123", oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test-token"})).
		Batch(context.Background(), batch, WithRequestID(""))
	require.NoError(t, err)
	require.Len(t, resp.BatchItemResponse, 1)
	assert.Equal(t, longID, resp.BatchItemResponse[0].ID)
}

func TestBatchRequestIDTooLong(t *testing.T) {
	client, err := NewClient()
	require.NoError(t, err)

	batch := NewBatchRequest(NewBatchQuery("q1", "SELECT * FROM Invoice"))
	_, err = client.Company("123", oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test-token"})).
		Batch(context.Background(), batch, WithRequestID(strings.Repeat("x", BatchRequestIDMaxLength+1)))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "exceeds maximum length of 36")
}

func TestBatchItemID(t *testing.T) {
	tests := []struct {
		name    string
		entity  string
		op      string
		seq     int
		want    string
		wantErr bool
	}{
		// All parts present — using BatchEntityAbbr* constants.
		{name: "all parts", entity: BatchEntityAbbrInvoice, op: BatchOpAbbrCreate, seq: 1, want: "in-c1"},
		{name: "longest entity abbr", entity: BatchEntityAbbrReimburseCharge, op: BatchOpAbbrUpdate, seq: 3, want: "rc-u3"},
		// Entity only.
		{name: "entity only", entity: BatchEntityAbbrInvoice, op: "", seq: 0, want: "in"},
		// Entity + seq, no op.
		{name: "entity and seq", entity: BatchEntityAbbrInvoice, op: "", seq: 5, want: "in-5"},
		// Entity + op, no seq.
		{name: "entity and op", entity: BatchEntityAbbrInvoice, op: BatchOpAbbrCreate, seq: 0, want: "in-c"},
		// Op + seq, no entity.
		{name: "op and seq", entity: "", op: BatchOpAbbrQuery, seq: 2, want: "q2"},
		// Op only.
		{name: "op only", entity: "", op: BatchOpAbbrDelete, seq: 0, want: "d"},
		// Seq only.
		{name: "seq only", entity: "", op: "", seq: 7, want: "7"},
		// Maximum seq that still fits with entity+op present (in-c + 6 digits = 10).
		{name: "max seq with entity and op", entity: BatchEntityAbbrInvoice, op: BatchOpAbbrCreate, seq: 999999, want: "in-c999999"},
		// All zero — error.
		{name: "all empty", entity: "", op: "", seq: 0, wantErr: true},
		// Custom abbreviation — accepted as-is.
		{name: "custom entity abbr", entity: "inv", op: BatchOpAbbrCreate, seq: 1, want: "inv-c1"},
		// seq overflow with entity and op present.
		{name: "overflow with entity and op", entity: BatchEntityAbbrInvoice, op: BatchOpAbbrCreate, seq: 1000000, wantErr: true},
		// seq overflow with entity and no op (in-<seq>).
		{name: "overflow entity only", entity: BatchEntityAbbrInvoice, op: "", seq: 100000000, wantErr: true},
		// All operation abbreviations round-trip.
		{name: "op update", entity: BatchEntityAbbrBill, op: BatchOpAbbrUpdate, seq: 1, want: "bi-u1"},
		{name: "op delete", entity: BatchEntityAbbrBill, op: BatchOpAbbrDelete, seq: 1, want: "bi-d1"},
		{name: "op void", entity: BatchEntityAbbrBill, op: BatchOpAbbrVoid, seq: 1, want: "bi-v1"},
		{name: "op query", entity: BatchEntityAbbrBill, op: BatchOpAbbrQuery, seq: 1, want: "bi-q1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := BatchItemID(tt.entity, tt.op, tt.seq)
			if tt.wantErr {
				require.Error(t, err)
				assert.Empty(t, got)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
				assert.LessOrEqual(t, len(got), BatchItemIDMaxLength)
			}
		})
	}
}

func TestEntityAbbr(t *testing.T) {
	// Every EntityName constant must have a known abbreviation.
	for _, entity := range EntityNames() {
		abbr, ok := EntityAbbr(entity)
		assert.True(t, ok, "EntityAbbr(%q) returned false", entity)
		assert.NotEmpty(t, abbr, "EntityAbbr(%q) returned empty string", entity)
	}

	// All abbreviations must be unique.
	seen := make(map[string]EntityName)
	for _, entity := range EntityNames() {
		abbr, _ := EntityAbbr(entity)
		if prev, dup := seen[abbr]; dup {
			t.Errorf("abbreviation %q is shared by %q and %q", abbr, prev, entity)
		}
		seen[abbr] = entity
	}

	// Unknown entity returns false.
	_, ok := EntityAbbr("Bogus")
	assert.False(t, ok)
}

func TestEntityFromAbbr(t *testing.T) {
	// Round-trip: every EntityName should survive EntityAbbr → EntityFromAbbr.
	for _, entity := range EntityNames() {
		abbr, ok := EntityAbbr(entity)
		require.True(t, ok)
		got, ok := EntityFromAbbr(abbr)
		assert.True(t, ok, "EntityFromAbbr(%q) returned false", abbr)
		assert.Equal(t, entity, got)
	}

	// Unknown abbreviation returns false.
	_, ok := EntityFromAbbr("zz")
	assert.False(t, ok)
}

func TestExtractBatchItemEntities(t *testing.T) {
	mustUnmarshalItems := func(t *testing.T, raw string) []BatchItemResponse {
		t.Helper()
		var resp BatchResponse
		require.NoError(t, json.Unmarshal([]byte(raw), &resp))
		return resp.BatchItemResponse
	}

	t.Run("nil slice returns nil", func(t *testing.T) {
		got, err := ExtractBatchItemEntities[Invoice](nil, EntityInvoice)
		require.NoError(t, err)
		assert.Nil(t, got)
	})

	t.Run("empty slice returns nil", func(t *testing.T) {
		got, err := ExtractBatchItemEntities[Invoice]([]BatchItemResponse{}, EntityInvoice)
		require.NoError(t, err)
		assert.Nil(t, got)
	})

	t.Run("single query response item", func(t *testing.T) {
		items := mustUnmarshalItems(t, `{"BatchItemResponse":[
			{"bId":"q1","QueryResponse":{"Invoice":[{"Id":"1","SyncToken":"0"},{"Id":"2","SyncToken":"0"}]}}
		]}`)
		got, err := ExtractBatchItemEntities[Invoice](items, EntityInvoice)
		require.NoError(t, err)
		require.Len(t, got, 2)
		assert.Equal(t, "1", got[0].ID())
		assert.Equal(t, "2", got[1].ID())
	})

	t.Run("multiple paginated query items collected together", func(t *testing.T) {
		// Simulates the primary use case: three pages of Invoice queries in one batch.
		items := mustUnmarshalItems(t, `{"BatchItemResponse":[
			{"bId":"q1","QueryResponse":{"Invoice":[{"Id":"1","SyncToken":"0"},{"Id":"2","SyncToken":"0"}]}},
			{"bId":"q2","QueryResponse":{"Invoice":[{"Id":"3","SyncToken":"0"},{"Id":"4","SyncToken":"0"}]}},
			{"bId":"q3","QueryResponse":{"Invoice":[{"Id":"5","SyncToken":"0"}]}}
		]}`)
		got, err := ExtractBatchItemEntities[Invoice](items, EntityInvoice)
		require.NoError(t, err)
		require.Len(t, got, 5)
		assert.Equal(t, "1", got[0].ID())
		assert.Equal(t, "5", got[4].ID())
	})

	t.Run("subset of items from a larger response", func(t *testing.T) {
		// Caller selects only pages 2 and 3, skipping page 1.
		all := mustUnmarshalItems(t, `{"BatchItemResponse":[
			{"bId":"q1","QueryResponse":{"Invoice":[{"Id":"1","SyncToken":"0"}]}},
			{"bId":"q2","QueryResponse":{"Invoice":[{"Id":"2","SyncToken":"0"}]}},
			{"bId":"q3","QueryResponse":{"Invoice":[{"Id":"3","SyncToken":"0"}]}}
		]}`)
		got, err := ExtractBatchItemEntities[Invoice](all[1:], EntityInvoice)
		require.NoError(t, err)
		require.Len(t, got, 2)
		assert.Equal(t, "2", got[0].ID())
		assert.Equal(t, "3", got[1].ID())
	})

	t.Run("mutation response item collected alongside query response", func(t *testing.T) {
		items := mustUnmarshalItems(t, `{"BatchItemResponse":[
			{"bId":"c1","Invoice":{"Id":"10","SyncToken":"0"}},
			{"bId":"q1","QueryResponse":{"Invoice":[{"Id":"11","SyncToken":"0"}]}}
		]}`)
		got, err := ExtractBatchItemEntities[Invoice](items, EntityInvoice)
		require.NoError(t, err)
		require.Len(t, got, 2)
		assert.Equal(t, "10", got[0].ID())
		assert.Equal(t, "11", got[1].ID())
	})

	t.Run("only matching entity type collected from mixed response", func(t *testing.T) {
		// Items contain both Invoice and Bill query responses; only Invoices extracted.
		items := mustUnmarshalItems(t, `{"BatchItemResponse":[
			{"bId":"iq1","QueryResponse":{"Invoice":[{"Id":"1","SyncToken":"0"}]}},
			{"bId":"bq1","QueryResponse":{"Bill":[{"Id":"99","SyncToken":"0"}]}},
			{"bId":"iq2","QueryResponse":{"Invoice":[{"Id":"2","SyncToken":"0"}]}}
		]}`)
		got, err := ExtractBatchItemEntities[Invoice](items, EntityInvoice)
		require.NoError(t, err)
		require.Len(t, got, 2)
		assert.Equal(t, "1", got[0].ID())
		assert.Equal(t, "2", got[1].ID())
	})
}

func TestExtractBatchItemsServiceMethod(t *testing.T) {
	client, err := NewClient()
	require.NoError(t, err)
	svc := client.Company("123", oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test-token"})).Invoices()

	var resp BatchResponse
	require.NoError(t, json.Unmarshal([]byte(`{"BatchItemResponse":[
		{"bId":"q1","QueryResponse":{"Invoice":[{"Id":"1","SyncToken":"0"}]}},
		{"bId":"q2","QueryResponse":{"Invoice":[{"Id":"2","SyncToken":"0"}]}}
	]}`), &resp))

	// ExtractBatchItems on a subset matches ExtractBatchItemEntities directly.
	got, err := svc.ExtractBatchItems(resp.BatchItemResponse[1:])
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, "2", got[0].ID())

	// ExtractBatch on the full response collects all items.
	all, err := svc.ExtractBatch(&resp)
	require.NoError(t, err)
	require.Len(t, all, 2)
}
