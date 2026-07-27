// Package quickbooks provides a Go client for the QuickBooks Online (QBO) v3
// API at minor version 75.
//
// # Getting started
//
// Construct a Client with NewClient, configure OAuth via WithOAuthConfig, then
// obtain a CompanyClient for a specific realm:
//
//	client, err := quickbooks.NewClient(
//	    quickbooks.WithOAuthConfig("clientID", "clientSecret", "https://redirect.example/callback", "", "", quickbooks.AccountingScope),
//	)
//
//	// After completing the OAuth flow and obtaining a token:
//	tokenSource, err := client.TokenSource(ctx, token)
//	company := client.Company(realmID, tokenSource)
//
//	// Make API calls through the company client:
//	invoice, err := company.Invoices().Get(ctx, "123")
//	invoices, err := company.Invoices().List(ctx)
//
// # OAuth flow
//
// Use AuthCodeURL to generate the authorization URL, redirect the user, then
// call Exchange with the returned code to obtain a token. Store the token and
// use TokenSource to create an auto-refreshing source for subsequent calls.
//
// # Sandbox
//
// Point at the sandbox API with WithBaseURL(quickbooks.SandboxBaseURL) and
// use DiscoveryEndpointSandbox with Discovery to fetch sandbox OAuth endpoints.
//
// # Rate limiting
//
// Rate limiting is enabled by default using token-bucket limiters per realm,
// matching the QBO API limits for general (500/min) and batch (40/min)
// endpoints. Disable with WithNoRateLimiting or customise with
// WithRateLimitConfig.
//
// # Batch operations
//
// Multiple entity operations can be sent in a single API call using
// CompanyClient.Batch. Use the BatchXxxItem methods on each service to build
// items, add them to a BatchRequest, and call GroupBatchRequests if you have
// more than 30 items.
//
// # Change Data Capture (CDC)
//
// Fetch all changes since a given time across multiple entities in one call:
//
//	resp, err := company.ChangeDataCapture(ctx, since, quickbooks.CDCEntityNames())
//	invoices, err := company.Invoices().ExtractCDC(resp)
//
// # Entity support
//
// See STATUS_MATRIX.md for the full list of supported entities and operations.
package quickbooks
