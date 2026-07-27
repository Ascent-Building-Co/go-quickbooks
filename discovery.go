package quickbooks

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

const (
	// DiscoveryEndpointProduction is the Intuit OIDC discovery endpoint for
	// the production environment. Pass it to Discovery to retrieve live
	// authorization, token, and userinfo endpoints.
	DiscoveryEndpointProduction = "https://developer.api.intuit.com/.well-known/openid_configuration"

	// DiscoveryEndpointSandbox is the Intuit OIDC discovery endpoint for the
	// sandbox environment.
	DiscoveryEndpointSandbox = "https://developer.api.intuit.com/.well-known/openid_sandbox_configuration"
)

// DiscoveryDocument holds the Intuit OIDC discovery document returned by the
// well-known configuration endpoint. The most commonly used fields are
// AuthorizationEndpoint and TokenEndpoint, which can be passed directly to
// WithOAuthConfig to ensure endpoints stay current without hardcoding.
type DiscoveryDocument struct {
	Issuer                            string   `json:"issuer"`
	AuthorizationEndpoint             string   `json:"authorization_endpoint"`
	TokenEndpoint                     string   `json:"token_endpoint"`
	UserinfoEndpoint                  string   `json:"userinfo_endpoint"`
	RevocationEndpoint                string   `json:"revocation_endpoint"`
	JwksURI                           string   `json:"jwks_uri"`
	ResponseTypesSupported            []string `json:"response_types_supported"`
	SubjectTypesSupported             []string `json:"subject_types_supported"`
	IDTokenSigningAlgValuesSupported  []string `json:"id_token_signing_alg_values_supported"`
	ScopesSupported                   []string `json:"scopes_supported"`
	TokenEndpointAuthMethodsSupported []string `json:"token_endpoint_auth_methods_supported"`
	ClaimsSupported                   []string `json:"claims_supported"`
}

// Discovery fetches and parses the Intuit OIDC discovery document from
// discoverEndpoint. Use DiscoveryEndpointProduction or
// DiscoveryEndpointSandbox as the endpoint. If client is nil,
// http.DefaultClient is used.
func Discovery(ctx context.Context, client *http.Client, discoverEndpoint string) (*DiscoveryDocument, error) {
	if client == nil {
		client = http.DefaultClient
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, discoverEndpoint, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("discovery endpoint error: status %d", resp.StatusCode)
	}

	var discovery DiscoveryDocument
	if err = json.Unmarshal(body, &discovery); err != nil {
		return nil, err
	}

	return &discovery, nil
}
