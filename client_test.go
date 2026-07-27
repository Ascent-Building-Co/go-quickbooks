package quickbooks

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
)

type blockingLimiter struct{}

func (blockingLimiter) Acquire(ctx context.Context, _ RequestScope) (ConcurrentRelease, error) {
	<-ctx.Done()
	return nil, ctx.Err()
}

func gzipString(t *testing.T, body string) []byte {
	t.Helper()

	var buf bytes.Buffer
	writer := gzip.NewWriter(&buf)
	_, err := io.WriteString(writer, body)
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	return buf.Bytes()
}

func TestClientDoJSONUsesTokenSourceAndRequestID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/company/123/invoice/42", r.URL.Path)
		assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))
		assert.Equal(t, "75", r.URL.Query().Get("minorversion"))
		assert.Equal(t, "client-generated-id", r.URL.Query().Get("requestid"))

		w.Header().Set("Content-Type", "application/json")
		require.NoError(t, json.NewEncoder(w).Encode(map[string]string{"status": "ok"}))
	}))
	defer server.Close()

	client, err := NewClient(
		WithHTTPClient(server.Client()),
		WithBaseURL(server.URL+"/v3/company"),
	)
	require.NoError(t, err)

	var out map[string]string
	err = client.doJSON(
		context.Background(),
		http.MethodPost,
		"123",
		"invoice/42",
		oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test-token"}),
		nil,
		map[string]string{"hello": "world"},
		&out,
		WithRequestID("client-generated-id"),
	)
	require.NoError(t, err)
	assert.Equal(t, "ok", out["status"])
}

func TestClientDoJSONGeneratesRequestIDForMutations(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := r.URL.Query().Get("requestid")
		assert.NotEmpty(t, requestID)
		_, err := uuid.Parse(requestID)
		require.NoError(t, err)

		w.Header().Set("Content-Type", "application/json")
		require.NoError(t, json.NewEncoder(w).Encode(map[string]string{"status": "ok"}))
	}))
	defer server.Close()

	client, err := NewClient(WithHTTPClient(server.Client()), WithBaseURL(server.URL+"/v3/company"))
	require.NoError(t, err)

	var out map[string]string
	err = client.doJSON(
		context.Background(),
		http.MethodPost,
		"123",
		"invoice",
		oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test-token"}),
		nil,
		map[string]string{"hello": "world"},
		&out,
	)
	require.NoError(t, err)
	assert.Equal(t, "ok", out["status"])
}

func TestClientDoJSONSuppressesRequestIDWhenOptedOut(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Empty(t, r.URL.Query().Get("requestid"))

		w.Header().Set("Content-Type", "application/json")
		require.NoError(t, json.NewEncoder(w).Encode(map[string]string{"status": "ok"}))
	}))
	defer server.Close()

	client, err := NewClient(WithHTTPClient(server.Client()), WithBaseURL(server.URL+"/v3/company"))
	require.NoError(t, err)

	var out map[string]string
	err = client.doJSON(
		context.Background(),
		http.MethodPost,
		"123",
		"invoice",
		oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test-token"}),
		nil,
		map[string]string{"hello": "world"},
		&out,
		WithRequestID(""), // explicit opt-out
	)
	require.NoError(t, err)
	assert.Equal(t, "ok", out["status"])
}

func TestClientDoJSONRejectsRequestIDTooLong(t *testing.T) {
	client, err := NewClient(WithBaseURL("https://example.com/v3/company"))
	require.NoError(t, err)

	err = client.doJSON(
		context.Background(),
		http.MethodPost,
		"123",
		"invoice",
		oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test-token"}),
		nil,
		map[string]string{"hello": "world"},
		nil,
		WithRequestID(strings.Repeat("x", RequestIDMaxLength+1)),
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "exceeds maximum length of 50")
}

func TestClientDoJSONOmitsRequestIDForReadsByDefault(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Empty(t, r.URL.Query().Get("requestid"))

		w.Header().Set("Content-Type", "application/json")
		require.NoError(t, json.NewEncoder(w).Encode(map[string]string{"id": "1"}))
	}))
	defer server.Close()

	client, err := NewClient(WithHTTPClient(server.Client()), WithBaseURL(server.URL+"/v3/company"))
	require.NoError(t, err)

	var out map[string]string
	err = client.doJSON(
		context.Background(),
		http.MethodGet,
		"123",
		"account/1",
		oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test-token"}),
		nil,
		nil,
		&out,
	)
	require.NoError(t, err)
	assert.Equal(t, "1", out["id"])
}

func TestCompanyClientUsesBoundRealmAndTokenSource(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/company/999/account/1", r.URL.Path)
		assert.Equal(t, "Bearer company-token", r.Header.Get("Authorization"))

		w.Header().Set("Content-Type", "application/json")
		require.NoError(t, json.NewEncoder(w).Encode(map[string]string{"id": "1"}))
	}))
	defer server.Close()

	client, err := NewClient(WithHTTPClient(server.Client()), WithBaseURL(server.URL+"/v3/company"))
	require.NoError(t, err)

	company := client.Company("999", oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "company-token"}))

	var out map[string]string
	err = company.doJSON(context.Background(), http.MethodGet, "account/1", nil, nil, &out)
	require.NoError(t, err)
	assert.Equal(t, "1", out["id"])
}

func TestClientDoJSONReturnsAPIErrorForJSONFault(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("intuit_tid", "tid-json")
		w.WriteHeader(http.StatusBadRequest)
		_, err := io.WriteString(w, `{"Fault":{"Error":[{"Message":"Length exceeds limit","Detail":"Length of the field exceeds 21 chars","code":"2050","element":"firstname"},{"Message":"Illegal number format","Detail":"ZipCode should be a number with at least 5 digits","code":"2080","element":"postalcode"}],"type":"ValidationFault"},"time":"2013-03-29T15:58:08.976-07:00"}`)
		require.NoError(t, err)
	}))
	defer server.Close()

	client, err := NewClient(WithHTTPClient(server.Client()), WithBaseURL(server.URL+"/v3/company"))
	require.NoError(t, err)

	err = client.doJSON(
		context.Background(),
		http.MethodPost,
		"123",
		"customer",
		oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test-token"}),
		nil,
		map[string]string{"name": "x"},
		nil,
	)

	var apiErr *APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, http.StatusBadRequest, apiErr.StatusCode)
	assert.Equal(t, "tid-json", apiErr.IntuitTID)
	require.NotNil(t, apiErr.Failure)
	assert.Equal(t, "ValidationFault", apiErr.Failure.Fault.Type)
	require.Len(t, apiErr.Failure.Fault.Error, 2)
	assert.Equal(t, "2050", apiErr.Failure.Fault.Error[0].Code)
	assert.Contains(t, apiErr.Error(), "Length of the field exceeds 21 chars")
	assert.Contains(t, apiErr.Error(), "ZipCode should be a number with at least 5 digits")
}

func TestClientDoJSONReturnsAPIErrorForXMLFault(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/xml")
		w.Header().Set("intuit_tid", "tid-xml")
		w.WriteHeader(http.StatusUnauthorized)
		_, err := io.WriteString(w, `<?xml version="1.0" encoding="UTF-8" standalone="yes"?><IntuitResponse xmlns="http://schema.intuit.com/finance/v3" time="2020-08-19T17:31:26.785-07:00"><Fault type="AuthenticationFault"><Error code="3200"><Message>message=AuthenticationFailed; errorCode=003200; statusCode=401</Message></Error></Fault></IntuitResponse>`)
		require.NoError(t, err)
	}))
	defer server.Close()

	client, err := NewClient(WithHTTPClient(server.Client()), WithBaseURL(server.URL+"/v3/company"))
	require.NoError(t, err)

	err = client.doJSON(
		context.Background(),
		http.MethodGet,
		"123",
		"customer/1",
		oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test-token"}),
		nil,
		nil,
		nil,
	)

	var apiErr *APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, http.StatusUnauthorized, apiErr.StatusCode)
	assert.Equal(t, "tid-xml", apiErr.IntuitTID)
	require.NotNil(t, apiErr.Failure)
	assert.Equal(t, "AuthenticationFault", apiErr.Failure.Fault.Type)
	require.Len(t, apiErr.Failure.Fault.Error, 1)
	assert.Equal(t, "3200", apiErr.Failure.Fault.Error[0].Code)
	assert.Contains(t, apiErr.Error(), "AuthenticationFailed")
}

func TestClientDoJSONReturnsRateLimitError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Content-Encoding", "gzip")
		w.Header().Set("Retry-After", "60")
		w.Header().Set("intuit_tid", "tid-rate")
		w.WriteHeader(http.StatusTooManyRequests)
		_, err := w.Write(gzipString(t, `{"Fault":{"Error":[{"Message":"ThrottleExceeded","Detail":"Too many requests, please retry later","code":"3001"}],"type":"SystemFault"},"time":"2026-03-10T10:00:00.000-07:00"}`))
		require.NoError(t, err)
	}))
	defer server.Close()

	client, err := NewClient(
		WithHTTPClient(server.Client()),
		WithBaseURL(server.URL+"/v3/company"),
	)
	require.NoError(t, err)

	err = client.doJSON(
		context.Background(),
		http.MethodPost,
		"123",
		"invoice",
		oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test-token"}),
		nil,
		map[string]string{"name": "x"},
		nil,
		WithRequestID("req-123"),
	)

	var rateErr RateLimitError
	require.ErrorAs(t, err, &rateErr)
	assert.Equal(t, http.StatusTooManyRequests, rateErr.StatusCode)
	assert.Equal(t, "123", rateErr.RealmID)
	assert.Equal(t, "req-123", rateErr.RequestID)
	assert.Equal(t, "60", rateErr.RetryAfter)
	assert.Equal(t, "tid-rate", rateErr.IntuitTID)
	require.NotNil(t, rateErr.Failure)
	assert.Equal(t, "3001", rateErr.Failure.Fault.Error[0].Code)
	assert.Contains(t, rateErr.Error(), "retry after 60")
}

func TestClientDoJSONTreats200FaultAsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, err := io.WriteString(w, `{"Fault":{"Error":[{"Message":"Duplicate Request ID","Detail":"The request ID already exists","code":"600"}],"type":"ValidationFault"},"time":"2026-03-10T10:00:00.000-07:00"}`)
		require.NoError(t, err)
	}))
	defer server.Close()

	client, err := NewClient(WithHTTPClient(server.Client()), WithBaseURL(server.URL+"/v3/company"))
	require.NoError(t, err)

	var out map[string]string
	err = client.doJSON(
		context.Background(),
		http.MethodPost,
		"123",
		"invoice",
		oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test-token"}),
		nil,
		map[string]string{"name": "x"},
		&out,
	)

	var apiErr *APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, http.StatusOK, apiErr.StatusCode)
	require.NotNil(t, apiErr.Failure)
	assert.Equal(t, "600", apiErr.Failure.Fault.Error[0].Code)
}

func TestClientRespectsRateLimitWaitTimeout(t *testing.T) {
	client, err := NewClient(
		WithRateLimiter(blockingLimiter{}),
		WithRateLimitConfig(RateLimitConfig{WaitTimeout: 20 * time.Millisecond}),
	)
	require.NoError(t, err)

	err = client.doJSON(
		context.Background(),
		http.MethodGet,
		"123",
		"account/1",
		oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test-token"}),
		nil,
		nil,
		nil,
	)
	require.Error(t, err)
	assert.True(t, errors.Is(err, context.DeadlineExceeded))
}

func TestOAuthHelpersRequireConfig(t *testing.T) {
	client, err := NewClient()
	require.NoError(t, err)

	_, err = client.AuthCodeURL("state")
	require.ErrorIs(t, err, ErrOAuthConfigNotSet)

	_, err = client.TokenSource(context.Background(), &oauth2.Token{})
	require.ErrorIs(t, err, ErrOAuthConfigNotSet)

	_, err = client.ExchangeWithRedirect(context.Background(), "code", "https://example.com/callback")
	require.ErrorIs(t, err, ErrOAuthConfigNotSet)
}

func TestExchangeWithRedirectUsesProvidedRedirectURL(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.NoError(t, r.ParseForm())
		assert.Equal(t, "authorization_code", r.Form.Get("grant_type"))
		assert.Equal(t, "test-code", r.Form.Get("code"))
		assert.Equal(t, "https://dynamic.example/callback", r.Form.Get("redirect_uri"))

		w.Header().Set("Content-Type", "application/json")
		_, err := io.WriteString(w, `{"access_token":"access-token","token_type":"Bearer","refresh_token":"refresh-token","expires_in":3600}`)
		require.NoError(t, err)
	}))
	defer server.Close()

	client, err := NewClient(
		WithHTTPClient(server.Client()),
		WithOAuthConfig("client-id", "client-secret", "https://default.example/callback", "", "", AccountingScope),
	)
	require.NoError(t, err)

	client.oauthConfig.Endpoint.TokenURL = server.URL

	token, err := client.ExchangeWithRedirect(context.Background(), "test-code", "https://dynamic.example/callback")
	require.NoError(t, err)
	assert.Equal(t, "access-token", token.AccessToken)
	assert.Equal(t, "refresh-token", token.RefreshToken)
	assert.Equal(t, "Bearer", token.TokenType)
}

func TestAuthCodeURLWithRedirectUsesProvidedRedirectURL(t *testing.T) {
	client, err := NewClient(
		WithOAuthConfig("client-id", "client-secret", "https://default.example/callback", "", "", AccountingScope),
	)
	require.NoError(t, err)

	authURL, err := client.AuthCodeURLWithRedirect("state", "https://dynamic.example/callback")
	require.NoError(t, err)

	parsed, err := url.Parse(authURL)
	require.NoError(t, err)
	assert.Equal(t, "https://dynamic.example/callback", parsed.Query().Get("redirect_uri"))
	assert.True(t, strings.Contains(parsed.Query().Get("scope"), AccountingScope))
}

func TestWithIncludeAppendsQueryParam(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "invoiceLink", r.URL.Query().Get("include"))
		w.Header().Set("Content-Type", "application/json")
		require.NoError(t, json.NewEncoder(w).Encode(map[string]string{"Id": "42"}))
	}))
	defer server.Close()

	client, err := NewClient(WithHTTPClient(server.Client()), WithBaseURL(server.URL+"/v3/company"))
	require.NoError(t, err)

	var out map[string]string
	err = client.doJSON(
		context.Background(),
		http.MethodGet,
		"123",
		"invoice/42",
		oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test-token"}),
		nil,
		nil,
		&out,
		WithInclude(IncludeInvoiceLink),
	)
	require.NoError(t, err)
}

func TestWithIncludeMultipleValuesJoined(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "invoiceLink,updateaccountontxns", r.URL.Query().Get("include"))
		w.Header().Set("Content-Type", "application/json")
		require.NoError(t, json.NewEncoder(w).Encode(map[string]string{"Id": "1"}))
	}))
	defer server.Close()

	client, err := NewClient(WithHTTPClient(server.Client()), WithBaseURL(server.URL+"/v3/company"))
	require.NoError(t, err)

	var out map[string]string
	err = client.doJSON(
		context.Background(),
		http.MethodGet,
		"123",
		"invoice/1",
		oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test-token"}),
		nil,
		nil,
		&out,
		WithInclude(IncludeInvoiceLink, IncludeUpdateAccountOnTxns),
	)
	require.NoError(t, err)
}

func TestClientDoBytesReturnsBinaryResponse(t *testing.T) {
	pdfPayload := []byte("%PDF-1.4 fake pdf content")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/company/123/creditmemo/73/pdf", r.URL.Path)
		assert.Equal(t, "application/pdf", r.Header.Get("Accept"))
		assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))
		assert.Equal(t, "75", r.URL.Query().Get("minorversion"))

		w.Header().Set("Content-Type", "application/pdf")
		_, err := w.Write(pdfPayload)
		require.NoError(t, err)
	}))
	defer server.Close()

	client, err := NewClient(WithHTTPClient(server.Client()), WithBaseURL(server.URL+"/v3/company"))
	require.NoError(t, err)

	got, err := client.Company("123", oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test-token"})).
		doBytes(context.Background(), http.MethodGet, "creditmemo/73/pdf", nil, "application/pdf")
	require.NoError(t, err)
	assert.Equal(t, pdfPayload, got)
}

func TestClientDoBytesReturnsAPIErrorOnFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_, err := io.WriteString(w, `{"Fault":{"Error":[{"Message":"Object Not Found","Detail":"Object not found","code":"610"}],"type":"ValidationFault"},"time":"2026-03-10T10:00:00.000-07:00"}`)
		require.NoError(t, err)
	}))
	defer server.Close()

	client, err := NewClient(WithHTTPClient(server.Client()), WithBaseURL(server.URL+"/v3/company"))
	require.NoError(t, err)

	_, err = client.Company("123", oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test-token"})).
		doBytes(context.Background(), http.MethodGet, "creditmemo/999/pdf", nil, "application/pdf")

	var apiErr *APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, http.StatusNotFound, apiErr.StatusCode)
}

func TestClientDoBytesHandlesGzipResponse(t *testing.T) {
	pdfPayload := []byte("%PDF-1.4 gzipped pdf content")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/pdf")
		w.Header().Set("Content-Encoding", "gzip")
		_, err := w.Write(func() []byte {
			var buf bytes.Buffer
			gz := gzip.NewWriter(&buf)
			_, _ = gz.Write(pdfPayload)
			_ = gz.Close()
			return buf.Bytes()
		}())
		require.NoError(t, err)
	}))
	defer server.Close()

	client, err := NewClient(WithHTTPClient(server.Client()), WithBaseURL(server.URL+"/v3/company"))
	require.NoError(t, err)

	got, err := client.Company("123", oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test-token"})).
		doBytes(context.Background(), http.MethodGet, "creditmemo/73/pdf", nil, "application/pdf")
	require.NoError(t, err)
	assert.Equal(t, pdfPayload, got)
}

func TestClientDoStreamReturnsReadCloser(t *testing.T) {
	pdfPayload := []byte("%PDF-1.4 fake pdf stream content")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/company/123/invoice/42/pdf", r.URL.Path)
		assert.Equal(t, "application/pdf", r.Header.Get("Accept"))
		assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))

		w.Header().Set("Content-Type", "application/pdf")
		_, err := w.Write(pdfPayload)
		require.NoError(t, err)
	}))
	defer server.Close()

	client, err := NewClient(WithHTTPClient(server.Client()), WithBaseURL(server.URL+"/v3/company"))
	require.NoError(t, err)

	rc, err := client.Company("123", oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test-token"})).
		doStream(context.Background(), http.MethodGet, "invoice/42/pdf", nil, "application/pdf")
	require.NoError(t, err)
	require.NotNil(t, rc)
	defer rc.Close()

	got, err := io.ReadAll(rc)
	require.NoError(t, err)
	assert.Equal(t, pdfPayload, got)
}

func TestClientDoStreamReturnsAPIErrorOnFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_, err := io.WriteString(w, `{"Fault":{"Error":[{"Message":"Object Not Found","Detail":"Object not found","code":"610"}],"type":"ValidationFault"},"time":"2026-03-10T10:00:00.000-07:00"}`)
		require.NoError(t, err)
	}))
	defer server.Close()

	client, err := NewClient(WithHTTPClient(server.Client()), WithBaseURL(server.URL+"/v3/company"))
	require.NoError(t, err)

	rc, err := client.Company("123", oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test-token"})).
		doStream(context.Background(), http.MethodGet, "invoice/999/pdf", nil, "application/pdf")
	require.Nil(t, rc)

	var apiErr *APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, http.StatusNotFound, apiErr.StatusCode)
}

func TestClientDoStreamHandlesGzipResponse(t *testing.T) {
	pdfPayload := []byte("%PDF-1.4 gzipped pdf stream content")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/pdf")
		w.Header().Set("Content-Encoding", "gzip")
		_, err := w.Write(func() []byte {
			var buf bytes.Buffer
			gz := gzip.NewWriter(&buf)
			_, _ = gz.Write(pdfPayload)
			_ = gz.Close()
			return buf.Bytes()
		}())
		require.NoError(t, err)
	}))
	defer server.Close()

	client, err := NewClient(WithHTTPClient(server.Client()), WithBaseURL(server.URL+"/v3/company"))
	require.NoError(t, err)

	rc, err := client.Company("123", oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test-token"})).
		doStream(context.Background(), http.MethodGet, "creditmemo/73/pdf", nil, "application/pdf")
	require.NoError(t, err)
	require.NotNil(t, rc)
	defer rc.Close()

	got, err := io.ReadAll(rc)
	require.NoError(t, err)
	assert.Equal(t, pdfPayload, got)
}

func TestClientDoStreamReleasesRateLimiterOnClose(t *testing.T) {
	released := make(chan struct{}, 1)

	limiter := &trackingLimiter{released: released}

	pdfPayload := []byte("%PDF-1.4 rate limiter test")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/pdf")
		_, err := w.Write(pdfPayload)
		require.NoError(t, err)
	}))
	defer server.Close()

	client, err := NewClient(
		WithHTTPClient(server.Client()),
		WithBaseURL(server.URL+"/v3/company"),
		WithRateLimiter(limiter),
	)
	require.NoError(t, err)

	rc, err := client.Company("123", oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test-token"})).
		doStream(context.Background(), http.MethodGet, "invoice/1/pdf", nil, "application/pdf")
	require.NoError(t, err)

	// Rate limiter slot should not yet be released.
	select {
	case <-released:
		t.Fatal("rate limiter released before Close()")
	default:
	}

	require.NoError(t, rc.Close())

	select {
	case <-released:
		// expected
	default:
		t.Fatal("rate limiter not released after Close()")
	}
}

func TestClientDoMultipartSendsCorrectContentType(t *testing.T) {
	var capturedContentType string
	var capturedBody []byte

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/company/123/upload", r.URL.Path)
		assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))
		capturedContentType = r.Header.Get("Content-Type")
		var err error
		capturedBody, err = io.ReadAll(r.Body)
		require.NoError(t, err)
		w.Header().Set("Content-Type", "application/json")
		_, err = io.WriteString(w, `{"AttachableResponse":[{"Attachable":{"Id":"42","SyncToken":"0"}}]}`)
		require.NoError(t, err)
	}))
	defer server.Close()

	// Build a minimal multipart body.
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	fw, err := mw.CreateFormFile("file", "receipt.jpg")
	require.NoError(t, err)
	_, err = fw.Write([]byte("fake-image-bytes"))
	require.NoError(t, err)
	require.NoError(t, mw.Close())

	client, err := NewClient(WithHTTPClient(server.Client()), WithBaseURL(server.URL+"/v3/company"))
	require.NoError(t, err)

	raw, err := client.Company("123", oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test-token"})).
		doMultipart(context.Background(), "upload", nil, &buf, mw.FormDataContentType())
	require.NoError(t, err)

	assert.Contains(t, capturedContentType, "multipart/form-data")
	assert.Contains(t, capturedContentType, "boundary=")
	assert.Contains(t, string(capturedBody), "fake-image-bytes")
	assert.Contains(t, string(raw), `"Id":"42"`)
}

func TestClientDoMultipartReturnsAPIErrorOnFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, err := io.WriteString(w, `{"Fault":{"Error":[{"Message":"Bad Request","Detail":"Invalid multipart","code":"400"}],"type":"ValidationFault"},"time":"2026-03-10T10:00:00.000-07:00"}`)
		require.NoError(t, err)
	}))
	defer server.Close()

	client, err := NewClient(WithHTTPClient(server.Client()), WithBaseURL(server.URL+"/v3/company"))
	require.NoError(t, err)

	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	require.NoError(t, mw.Close())

	_, err = client.Company("123", oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test-token"})).
		doMultipart(context.Background(), "upload", nil, &buf, mw.FormDataContentType())

	var apiErr *APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, http.StatusBadRequest, apiErr.StatusCode)
}

// TestDoMultipartOutlastsClientTimeout verifies uploads run on the transfer
// client, which drops the short client-wide Timeout in favour of the transfer
// deadline — so a slow upload that would trip http.Client.Timeout still
// succeeds.
func TestDoMultipartOutlastsClientTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(120 * time.Millisecond)
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"AttachableResponse":[{"Attachable":{"Id":"1","SyncToken":"0"}}]}`)
	}))
	defer server.Close()

	hc := server.Client()
	hc.Timeout = 30 * time.Millisecond // too short for the handler

	client, err := NewClient(
		WithHTTPClient(hc),
		WithBaseURL(server.URL+"/v3/company"),
		WithTransferTimeout(5*time.Second),
	)
	require.NoError(t, err)

	company := client.Company("123", oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test-token"}))

	// A plain JSON request on the short-Timeout client must fail...
	_, getErr := company.Attachables().Get(context.Background(), "1")
	require.Error(t, getErr)

	// ...but the multipart upload runs on the transfer client and succeeds.
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	require.NoError(t, mw.Close())

	raw, err := company.doMultipart(context.Background(), "upload", nil, &buf, mw.FormDataContentType())
	require.NoError(t, err)
	assert.Contains(t, string(raw), `"Id":"1"`)
}

type trackingLimiter struct {
	released chan struct{}
}

func (l *trackingLimiter) Acquire(_ context.Context, _ RequestScope) (ConcurrentRelease, error) {
	return func() { l.released <- struct{}{} }, nil
}
