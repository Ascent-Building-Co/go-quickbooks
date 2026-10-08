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

func TestExchangeRateServiceGet(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/company/123/exchangerate", r.URL.Path)
		assert.Equal(t, "EUR", r.URL.Query().Get("sourcecurrencycode"))
		assert.Equal(t, "2015-07-07", r.URL.Query().Get("asofdate"))
		w.Header().Set("Content-Type", "application/json")
		_, err := io.WriteString(w, `{"ExchangeRate":{"SyncToken":"1","AsOfDate":"2015-07-07","SourceCurrencyCode":"EUR","Rate":2.5,"TargetCurrencyCode":"USD"}}`)
		require.NoError(t, err)
	}))
	defer server.Close()

	client, err := NewClient(WithHTTPClient(server.Client()), WithBaseURL(server.URL+"/v3/company"))
	require.NoError(t, err)

	asOfDate := &Date{Time: time.Date(2015, 7, 7, 0, 0, 0, 0, time.UTC)}
	er, err := client.Company("123", oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test-token"})).ExchangeRates().Get(context.Background(), "EUR", asOfDate)
	require.NoError(t, err)
	assert.Equal(t, EntityExchangeRate, er.EntityName())
	assert.Equal(t, "1", er.SyncToken())
	assert.Equal(t, "EUR", er.SourceCurrencyCode)
	assert.Equal(t, "USD", er.TargetCurrencyCode)
	assert.Equal(t, "2.5", er.Rate.String())
	require.NotNil(t, er.AsOfDate)
	assert.Equal(t, "2015-07-07", er.AsOfDate.String())
}

func TestExchangeRateServiceGetWithoutDate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "INR", r.URL.Query().Get("sourcecurrencycode"))
		assert.Equal(t, "", r.URL.Query().Get("asofdate"))
		w.Header().Set("Content-Type", "application/json")
		_, err := io.WriteString(w, `{"ExchangeRate":{"SyncToken":"0","SourceCurrencyCode":"INR","Rate":65,"TargetCurrencyCode":"USD"}}`)
		require.NoError(t, err)
	}))
	defer server.Close()

	client, err := NewClient(WithHTTPClient(server.Client()), WithBaseURL(server.URL+"/v3/company"))
	require.NoError(t, err)

	er, err := client.Company("123", oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test-token"})).ExchangeRates().Get(context.Background(), "INR", nil)
	require.NoError(t, err)
	assert.Equal(t, "INR", er.SourceCurrencyCode)
	assert.Equal(t, "65", er.Rate.String())
}

func TestExchangeRateServiceListPage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/company/123/query", r.URL.Path)
		assert.Equal(t, "SELECT * FROM ExchangeRate STARTPOSITION 1 MAXRESULTS 5", r.URL.Query().Get("query"))
		w.Header().Set("Content-Type", "application/json")
		_, err := io.WriteString(w, `{"QueryResponse":{"ExchangeRate":[{"SyncToken":"0","SourceCurrencyCode":"EUR","Rate":1.1},{"SyncToken":"0","SourceCurrencyCode":"GBP","Rate":1.3}],"startPosition":1,"maxResults":2}}`)
		require.NoError(t, err)
	}))
	defer server.Close()

	client, err := NewClient(WithHTTPClient(server.Client()), WithBaseURL(server.URL+"/v3/company"))
	require.NoError(t, err)

	ers, err := client.Company("123", oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test-token"})).ExchangeRates().ListPage(context.Background(), 1, 5)
	require.NoError(t, err)
	require.Len(t, ers, 2)
	assert.Equal(t, "EUR", ers[0].SourceCurrencyCode)
	assert.Equal(t, "GBP", ers[1].SourceCurrencyCode)
}

func TestExchangeRateJSONDecodesV75Fields(t *testing.T) {
	const fixture = `{
		"ExchangeRate": {
			"SyncToken": "1",
			"MetaData": {
				"CreateTime": "2015-07-07T10:00:00-07:00",
				"LastUpdatedTime": "2015-07-07T10:00:00-07:00"
			},
			"SourceCurrencyCode": "EUR",
			"TargetCurrencyCode": "USD",
			"Rate": 2.5,
			"AsOfDate": "2015-07-07"
		}
	}`

	var envelope struct {
		ExchangeRate ExchangeRate `json:"ExchangeRate"`
	}
	require.NoError(t, json.Unmarshal([]byte(fixture), &envelope))

	er := envelope.ExchangeRate
	assert.Equal(t, "1", er.SyncToken())
	assert.Equal(t, "EUR", er.SourceCurrencyCode)
	assert.Equal(t, "USD", er.TargetCurrencyCode)
	assert.Equal(t, "2.5", er.Rate.String())
	require.NotNil(t, er.AsOfDate)
	assert.Equal(t, "2015-07-07", er.AsOfDate.String())
	require.NotNil(t, er.MetaData)
	require.NotNil(t, er.MetaData.CreateTime)
}

func TestExchangeRateUpdateMarshalPayload(t *testing.T) {
	var capturedBody []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/company/123/exchangerate", r.URL.Path)
		assert.Equal(t, http.MethodPost, r.Method)
		var err error
		capturedBody, err = io.ReadAll(r.Body)
		require.NoError(t, err)
		w.Header().Set("Content-Type", "application/json")
		_, err = io.WriteString(w, `{"ExchangeRate":{"SyncToken":"1","SourceCurrencyCode":"INR","Rate":7,"AsOfDate":"2015-07-08","TargetCurrencyCode":"USD"}}`)
		require.NoError(t, err)
	}))
	defer server.Close()

	client, err := NewClient(WithHTTPClient(server.Client()), WithBaseURL(server.URL+"/v3/company"))
	require.NoError(t, err)

	rate := ExchangeRate{
		SyncTokenValue:     SyncTokenValue{SyncTokenValue: "0"},
		SourceCurrencyCode: "INR",
		Rate:               json.Number("7"),
		AsOfDate:           &Date{Time: time.Date(2015, 7, 8, 0, 0, 0, 0, time.UTC)},
		TargetCurrencyCode: "USD",
	}

	result, err := client.Company("123", oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test-token"})).ExchangeRates().Update(context.Background(), rate)
	require.NoError(t, err)
	assert.Equal(t, "INR", result.SourceCurrencyCode)
	assert.Equal(t, "7", result.Rate.String())

	assert.Contains(t, string(capturedBody), `"SyncToken":"0"`)
	assert.Contains(t, string(capturedBody), `"SourceCurrencyCode":"INR"`)
	assert.Contains(t, string(capturedBody), `"Rate":7`)
}

func TestExchangeRateCDCExtraction(t *testing.T) {
	changedSince := time.Date(2026, time.March, 10, 12, 0, 0, 0, time.UTC)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/company/123/cdc", r.URL.Path)
		assert.Equal(t, string(EntityExchangeRate), r.URL.Query().Get("entities"))
		assert.Equal(t, changedSince.Format(time.RFC3339), r.URL.Query().Get("changedSince"))
		w.Header().Set("Content-Type", "application/json")
		_, err := io.WriteString(w, `{"CDCResponse":[{"QueryResponse":[{"ExchangeRate":[{"SyncToken":"1","domain":"QBO","AsOfDate":"2015-07-07","SourceCurrencyCode":"EUR","Rate":2.5,"TargetCurrencyCode":"USD","MetaData":{"LastUpdatedTime":"2015-07-08T09:24:02-07:00"}}],"startPosition":1,"maxResults":1}]}],"time":"2026-03-10T12:00:00Z"}`)
		require.NoError(t, err)
	}))
	defer server.Close()

	client, err := NewClient(WithHTTPClient(server.Client()), WithBaseURL(server.URL+"/v3/company"))
	require.NoError(t, err)

	rates, err := client.Company("123", oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test-token"})).ExchangeRates().ChangeDataCapture(context.Background(), changedSince)
	require.NoError(t, err)
	require.Len(t, rates, 1)
	assert.Equal(t, "EUR", rates[0].SourceCurrencyCode)
	assert.Equal(t, "2.5", rates[0].Rate.String())
	assert.Equal(t, "QBO", rates[0].CDCDomain())
	require.NotNil(t, rates[0].AsOfDate)
	assert.Equal(t, "2015-07-07", rates[0].AsOfDate.String())
}

// Compile-time interface compliance checks for ExchangeRate and ExchangeRateService.
// ExchangeRate does not have an Id field, so it does not satisfy the Entity or
// CDCEntity interfaces. It satisfies NamedEntity and Syncronized, and its
// service supports CDC through EntityCDCReader.
var (
	// ExchangeRate entity type compliance — no Identifiable.
	_ NamedEntity = ExchangeRate{}
	_ Syncronized = ExchangeRate{}

	// ExchangeRateService operation compliance.
	// Note: ExchangeRateService does not satisfy Getter because its Get
	// method takes sourceCurrencyCode + asOfDate instead of a single id.
	_ Querier[ExchangeRate]         = (*ExchangeRateService)(nil)
	_ Lister[ExchangeRate]          = (*ExchangeRateService)(nil)
	_ Pager[ExchangeRate]           = (*ExchangeRateService)(nil)
	_ Updater[ExchangeRate]         = (*ExchangeRateService)(nil)
	_ EntityCDCReader[ExchangeRate] = (*ExchangeRateService)(nil)
)
