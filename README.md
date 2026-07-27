# go-quickbooks

A Go client for the [QuickBooks Online v3 API](https://developer.intuit.com/app/developer/qbo/docs/api/accounting/most-commonly-used/account) at minor version 75.

## Installation

```sh
go get github.com/tommyhedley/go-quickbooks
```

## Quick start

```go
import quickbooks "github.com/tommyhedley/go-quickbooks"

// 1. Build the client once for your application.
client, err := quickbooks.NewClient(
    quickbooks.WithOAuthConfig(
        "clientID", "clientSecret",
        "https://your-app.example/callback",
        "", "",                          // empty = use production defaults
        quickbooks.AccountingScope,
    ),
)

// 2. Complete the OAuth flow (see below), then create a token source.
tokenSource, err := client.TokenSource(ctx, token)

// 3. Scope to a company and start calling the API.
company := client.Company(realmID, tokenSource)

invoices, err := company.Invoices().List(ctx)
invoice, err  := company.Invoices().Get(ctx, "42")
```

## OAuth flow

QuickBooks uses OAuth 2.0. The typical server-side flow:

```go
// Step 1 — redirect the user to Intuit to authorise your app.
authURL, err := client.AuthCodeURL("random-state-value")
http.Redirect(w, r, authURL, http.StatusFound)

// Step 2 — handle the callback and exchange the code for a token.
token, err := client.Exchange(ctx, r.URL.Query().Get("code"))

// Step 3 — persist the token, then create a reusing token source.
// The source automatically refreshes the access token when it expires.
tokenSource, err := client.TokenSource(ctx, token)

// Step 4 — extract the realmID from the callback query parameter.
realmID := r.URL.Query().Get("realmId")
company  := client.Company(realmID, tokenSource)
```

`TokenSource` wraps `oauth2.Config.TokenSource`, which handles refresh transparently. If the refresh token has expired, calls through `company` will return an error wrapping the underlying OAuth error.

## Sandbox

```go
doc, err := quickbooks.Discovery(ctx, nil, quickbooks.DiscoveryEndpointSandbox)

client, err := quickbooks.NewClient(
    quickbooks.WithBaseURL(quickbooks.SandboxBaseURL),
    quickbooks.WithOAuthConfig(
        "clientID", "clientSecret",
        "https://your-app.example/callback",
        doc.AuthorizationEndpoint,
        doc.TokenEndpoint,
        quickbooks.AccountingScope,
    ),
)
```

## Company scoping

`CompanyClient` is lightweight — it holds only a realm ID and a token source. You can create one per request or reuse it; both are safe.

```go
// Per-request style (fine for short-lived handlers):
invoice, err := client.Company(realmID, tokenSource).Invoices().Get(ctx, id)

// Reuse style (preferred for long-lived workers):
company := client.Company(realmID, tokenSource)
invoices, _ := company.Invoices().List(ctx)
vendors,  _ := company.Vendors().List(ctx)
```

## Batch operations

Send up to 30 entity operations in a single API call:

```go
req := quickbooks.NewBatchRequest(
    company.Invoices().BatchQueryItem("q1", "SELECT * FROM Invoice WHERE TotalAmt > '100'"),
    company.Vendors().BatchCreateItem("c1", newVendor),
)

resp, err := company.Batch(ctx, req)
if resp.HasFaults() {
    // inspect resp.Faults() for per-item errors
}

invoices, err := company.Invoices().ExtractBatch(resp)
vendors,  err := company.Vendors().ExtractBatch(resp)
```

When you have more than 30 items, split them with `GroupBatchRequests`:

```go
groups := quickbooks.GroupBatchRequests(items, quickbooks.BatchMaxItems)
for _, group := range groups {
    resp, err := company.Batch(ctx, quickbooks.NewBatchRequest(group...))
    // ...
}
```

## Change Data Capture (CDC)

Fetch all changes since a given time across multiple entity types:

```go
since := time.Now().Add(-24 * time.Hour)

// Fetch changes for all CDC-eligible entities at once:
resp, err := company.ChangeDataCapture(ctx, since, quickbooks.CDCEntityNames())

// Extract typed slices from the response:
invoices, err := company.Invoices().ExtractCDC(resp)
customers, err := company.Customers().ExtractCDC(resp)

// Deleted entities are included — check the status field:
for _, inv := range invoices {
    if inv.CDCStatus() == "Deleted" {
        // handle deletion
    }
}
```

Or use the per-entity shortcut which handles the CDC call internally:

```go
invoices, err := company.Invoices().ChangeDataCapture(ctx, since)
```

## Rate limiting

Rate limiting is **on by default**, matching Intuit's published limits:

| Endpoint | Rate | Burst |
|----------|------|-------|
| General  | 500 req/min | 10 |
| Batch    | 40 req/min  | 5  |
| Concurrent per realm | — | 10 |

Disable for testing:

```go
client, _ := quickbooks.NewClient(quickbooks.WithNoRateLimiting())
```

Customise limits:

```go
client, _ := quickbooks.NewClient(
    quickbooks.WithRateLimitConfig(quickbooks.RateLimitConfig{
        GeneralPerMinute: 300,
        WaitTimeout:      5 * time.Second, // return error instead of blocking
    }),
)
```

## Discovery

Fetch live OAuth endpoints from Intuit's OIDC discovery document:

```go
doc, err := quickbooks.Discovery(ctx, nil, quickbooks.DiscoveryEndpointProduction)
// doc.AuthorizationEndpoint, doc.TokenEndpoint, doc.ScopesSupported, ...
```

## Error handling

All API errors are returned as `*quickbooks.APIError`. Use `errors.As` to inspect them:

```go
var apiErr *quickbooks.APIError
if errors.As(err, &apiErr) {
    log.Printf("status %d, intuit_tid %s", apiErr.StatusCode, apiErr.IntuitTID)
    if apiErr.Failure != nil {
        for _, e := range apiErr.Failure.Fault.Error {
            log.Printf("  code %s: %s", e.Code, e.Detail)
        }
    }
}

var rateErr quickbooks.RateLimitError
if errors.As(err, &rateErr) {
    log.Printf("rate limited, retry after %s", rateErr.RetryAfter)
}
```

## Supported entities

See [STATUS_MATRIX.md](STATUS_MATRIX.md) for the full list of supported entities, operations, and implementation status.
