package quickbooks

import (
	"context"
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"golang.org/x/oauth2"
)

// DefaultJWKSCacheTTL is the default lifetime for cached JWK sets when no
// override is supplied to NewJWKSCache.
const DefaultJWKSCacheTTL = 12 * time.Hour

// idTokenAlgorithm is the only JWS algorithm Intuit currently signs ID
// tokens with.
const idTokenAlgorithm = "RS256"

// ID token validation errors. Callers should branch on these via errors.Is.
var (
	ErrIDTokenMalformed         = errors.New("id token: malformed JWT")
	ErrIDTokenAlgUnsupported    = errors.New("id token: unsupported algorithm")
	ErrIDTokenKeyNotFound       = errors.New("id token: signing key not found in JWK set")
	ErrIDTokenSignatureMismatch = errors.New("id token: signature mismatch")
	ErrIDTokenIssuerMismatch    = errors.New("id token: issuer mismatch")
	ErrIDTokenAudienceMismatch  = errors.New("id token: audience mismatch")
	ErrIDTokenExpired           = errors.New("id token: expired")
	ErrIDTokenIssuedInFuture    = errors.New("id token: issued in the future")
)

// UserInfo is the response body of the OIDC userinfo endpoint. Only the
// fields documented in the Intuit OpenID Connect spec are decoded.
//
// The Intuit spec requires apps to reject users whose EmailVerified is
// false; that policy is enforced by the caller, not this type.
type UserInfo struct {
	Sub                 string       `json:"sub"`
	Email               string       `json:"email,omitempty"`
	EmailVerified       bool         `json:"emailVerified,omitempty"`
	GivenName           string       `json:"givenName,omitempty"`
	FamilyName          string       `json:"familyName,omitempty"`
	PhoneNumber         string       `json:"phoneNumber,omitempty"`
	PhoneNumberVerified bool         `json:"phoneNumberVerified,omitempty"`
	Address             *UserAddress `json:"address,omitempty"`
}

// UserAddress is the nested address object returned by the OIDC userinfo
// endpoint when the address scope is granted.
type UserAddress struct {
	StreetAddress string `json:"streetAddress,omitempty"`
	Locality      string `json:"locality,omitempty"`
	Region        string `json:"region,omitempty"`
	PostalCode    string `json:"postalCode,omitempty"`
	Country       string `json:"country,omitempty"`
}

// FetchUserInfo calls userinfoEndpoint with accessToken as a bearer
// credential and decodes the response into a UserInfo. If client is nil,
// http.DefaultClient is used.
func FetchUserInfo(ctx context.Context, client *http.Client, userinfoEndpoint, accessToken string) (*UserInfo, error) {
	if userinfoEndpoint == "" {
		return nil, fmt.Errorf("userinfo endpoint is required")
	}
	if accessToken == "" {
		return nil, fmt.Errorf("access token is required")
	}
	if client == nil {
		client = http.DefaultClient
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, userinfoEndpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("build userinfo request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("userinfo request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read userinfo response: %w", err)
	}

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("userinfo endpoint error: status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var info UserInfo
	if err := json.Unmarshal(body, &info); err != nil {
		return nil, fmt.Errorf("decode userinfo response: %w", err)
	}
	return &info, nil
}

// GetUserInfo is a convenience wrapper around FetchUserInfo that uses the
// Client's HTTP client and pulls a fresh access token from tokenSource.
// userinfoEndpoint must come from the OIDC discovery document.
func (c *Client) GetUserInfo(ctx context.Context, tokenSource oauth2.TokenSource, userinfoEndpoint string) (*UserInfo, error) {
	if tokenSource == nil {
		return nil, ErrTokenSourceNil
	}
	tok, err := tokenSource.Token()
	if err != nil {
		return nil, fmt.Errorf("get access token: %w", err)
	}
	return FetchUserInfo(ctx, c.httpClient, userinfoEndpoint, tok.AccessToken)
}

// IDTokenClaims represents the standard claims from an Intuit ID token.
// Only claims documented in the QBO OpenID Connect spec are decoded;
// unknown claims are ignored.
type IDTokenClaims struct {
	Sub      string   `json:"sub"`
	Aud      []string `json:"aud"`
	RealmID  string   `json:"realmId"`
	AuthTime int64    `json:"auth_time"`
	Iss      string   `json:"iss"`
	Exp      int64    `json:"exp"`
	Iat      int64    `json:"iat"`
}

// idTokenAud is an internal helper that decodes the audience claim, which
// the OpenID spec permits as either a string or an array of strings.
type idTokenAud []string

func (a *idTokenAud) UnmarshalJSON(data []byte) error {
	if len(data) == 0 {
		return nil
	}
	if data[0] == '"' {
		var s string
		if err := json.Unmarshal(data, &s); err != nil {
			return err
		}
		*a = []string{s}
		return nil
	}
	var arr []string
	if err := json.Unmarshal(data, &arr); err != nil {
		return err
	}
	*a = arr
	return nil
}

func (c *IDTokenClaims) UnmarshalJSON(data []byte) error {
	type raw struct {
		Sub      string     `json:"sub"`
		Aud      idTokenAud `json:"aud"`
		RealmID  string     `json:"realmId"`
		AuthTime int64      `json:"auth_time"`
		Iss      string     `json:"iss"`
		Exp      int64      `json:"exp"`
		Iat      int64      `json:"iat"`
	}
	var r raw
	if err := json.Unmarshal(data, &r); err != nil {
		return err
	}
	c.Sub = r.Sub
	c.Aud = []string(r.Aud)
	c.RealmID = r.RealmID
	c.AuthTime = r.AuthTime
	c.Iss = r.Iss
	c.Exp = r.Exp
	c.Iat = r.Iat
	return nil
}

// ExtractIDToken returns the id_token string carried in token.Extra("id_token"),
// populated by oauth2.Config.Exchange when the openid scope is granted.
// Returns ("", false) when no ID token is present.
func ExtractIDToken(token *oauth2.Token) (string, bool) {
	if token == nil {
		return "", false
	}
	v, ok := token.Extra("id_token").(string)
	if !ok || v == "" {
		return "", false
	}
	return v, true
}

// idTokenHeader is the JWT header for an ID token. We need only kid and alg.
type idTokenHeader struct {
	Kid string `json:"kid"`
	Alg string `json:"alg"`
	Typ string `json:"typ,omitempty"`
}

// splitJWT splits a compact-serialized JWT into its three segments. Returns
// the raw signing input (header + "." + payload bytes) for signature
// verification.
func splitJWT(token string) (header, payload, sig, signingInput string, err error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return "", "", "", "", ErrIDTokenMalformed
	}
	return parts[0], parts[1], parts[2], parts[0] + "." + parts[1], nil
}

// ParseIDToken parses an unvalidated ID token JWT and returns its claims.
// Use ValidateIDToken when authenticity must be verified; ParseIDToken is
// suitable when the caller has independent assurance the token came from
// Intuit (for example, the token was just returned by Client.Exchange).
func ParseIDToken(idToken string) (IDTokenClaims, error) {
	_, payloadSeg, _, _, err := splitJWT(idToken)
	if err != nil {
		return IDTokenClaims{}, err
	}
	payload, err := base64.RawURLEncoding.DecodeString(payloadSeg)
	if err != nil {
		return IDTokenClaims{}, fmt.Errorf("%w: payload base64 decode: %v", ErrIDTokenMalformed, err)
	}
	var claims IDTokenClaims
	if err := json.Unmarshal(payload, &claims); err != nil {
		return IDTokenClaims{}, fmt.Errorf("%w: payload decode: %v", ErrIDTokenMalformed, err)
	}
	return claims, nil
}

// JWK is one entry from a JSON Web Key Set, restricted to the fields needed
// to verify Intuit ID tokens (RSA keys, RS256 signatures).
type JWK struct {
	Kid string `json:"kid"`
	Kty string `json:"kty"`
	Alg string `json:"alg"`
	N   string `json:"n"`
	E   string `json:"e"`
	Use string `json:"use,omitempty"`
}

// JWKSet is a collection of JWKs returned by a jwks_uri endpoint.
type JWKSet struct {
	Keys []JWK `json:"keys"`
}

// PublicKey returns the RSA public key for the JWK with the given kid, or
// (nil, false) if no matching RSA key is present.
func (s *JWKSet) PublicKey(kid string) (*rsa.PublicKey, bool) {
	if s == nil {
		return nil, false
	}
	for _, k := range s.Keys {
		if k.Kid != kid {
			continue
		}
		if !strings.EqualFold(k.Kty, "RSA") {
			return nil, false
		}
		nBytes, err := base64.RawURLEncoding.DecodeString(k.N)
		if err != nil {
			return nil, false
		}
		eBytes, err := base64.RawURLEncoding.DecodeString(k.E)
		if err != nil {
			return nil, false
		}
		n := new(big.Int).SetBytes(nBytes)
		e := new(big.Int).SetBytes(eBytes)
		if !e.IsInt64() {
			return nil, false
		}
		ev := e.Int64()
		if ev <= 0 {
			return nil, false
		}
		return &rsa.PublicKey{N: n, E: int(ev)}, true
	}
	return nil, false
}

// FetchJWKS retrieves and parses the JWK set at jwksURI. If client is nil,
// http.DefaultClient is used.
func FetchJWKS(ctx context.Context, client *http.Client, jwksURI string) (*JWKSet, error) {
	if jwksURI == "" {
		return nil, fmt.Errorf("jwks uri is required")
	}
	if client == nil {
		client = http.DefaultClient
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, jwksURI, nil)
	if err != nil {
		return nil, fmt.Errorf("build jwks request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("jwks request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read jwks response: %w", err)
	}

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("jwks endpoint error: status %d", resp.StatusCode)
	}

	var set JWKSet
	if err := json.Unmarshal(body, &set); err != nil {
		return nil, fmt.Errorf("decode jwks response: %w", err)
	}
	return &set, nil
}

// JWKSCache memoizes a JWK set fetched from jwksURI for a configurable TTL.
// All methods are safe for concurrent use.
type JWKSCache struct {
	httpClient *http.Client
	jwksURI    string
	ttl        time.Duration

	mu        sync.Mutex
	keys      *JWKSet
	expiresAt time.Time
}

// NewJWKSCache constructs a cache for the JWK set at jwksURI. ttl <= 0
// resolves to DefaultJWKSCacheTTL. If client is nil, http.DefaultClient is
// used at fetch time.
func NewJWKSCache(client *http.Client, jwksURI string, ttl time.Duration) *JWKSCache {
	if ttl <= 0 {
		ttl = DefaultJWKSCacheTTL
	}
	return &JWKSCache{
		httpClient: client,
		jwksURI:    jwksURI,
		ttl:        ttl,
	}
}

// Get returns the cached JWK set, fetching it (or refreshing it after
// expiry) as needed.
func (c *JWKSCache) Get(ctx context.Context) (*JWKSet, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.keys != nil && time.Now().Before(c.expiresAt) {
		return c.keys, nil
	}

	set, err := FetchJWKS(ctx, c.httpClient, c.jwksURI)
	if err != nil {
		return nil, err
	}
	c.keys = set
	c.expiresAt = time.Now().Add(c.ttl)
	return set, nil
}

// Invalidate forces the next Get call to refetch the JWK set.
func (c *JWKSCache) Invalidate() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.keys = nil
	c.expiresAt = time.Time{}
}

// IDTokenValidationOptions controls validation rules for ValidateIDToken.
//
// Issuer is the expected `iss` claim; Audience is the expected entry in the
// `aud` claim (typically the OAuth2 client_id). Both are required. ClockSkew
// adds tolerance to exp/iat checks; <= 0 resolves to 60 seconds.
type IDTokenValidationOptions struct {
	Issuer    string
	Audience  string
	ClockSkew time.Duration
}

// ValidateIDToken parses idToken, looks up its signing key by kid in keys,
// verifies the RS256 signature, and validates the standard claims (iss,
// aud, exp, iat) per OpenID Connect Core. On success, it returns the
// validated claims.
func ValidateIDToken(idToken string, keys *JWKSet, opts IDTokenValidationOptions) (IDTokenClaims, error) {
	if opts.Issuer == "" {
		return IDTokenClaims{}, fmt.Errorf("validation options: issuer required")
	}
	if opts.Audience == "" {
		return IDTokenClaims{}, fmt.Errorf("validation options: audience required")
	}
	skew := opts.ClockSkew
	if skew <= 0 {
		skew = 60 * time.Second
	}

	headerSeg, payloadSeg, sigSeg, signingInput, err := splitJWT(idToken)
	if err != nil {
		return IDTokenClaims{}, err
	}

	headerBytes, err := base64.RawURLEncoding.DecodeString(headerSeg)
	if err != nil {
		return IDTokenClaims{}, fmt.Errorf("%w: header base64: %v", ErrIDTokenMalformed, err)
	}
	var header idTokenHeader
	if err := json.Unmarshal(headerBytes, &header); err != nil {
		return IDTokenClaims{}, fmt.Errorf("%w: header decode: %v", ErrIDTokenMalformed, err)
	}
	if header.Alg != idTokenAlgorithm {
		return IDTokenClaims{}, fmt.Errorf("%w: %s", ErrIDTokenAlgUnsupported, header.Alg)
	}
	if header.Kid == "" {
		return IDTokenClaims{}, fmt.Errorf("%w: missing kid", ErrIDTokenMalformed)
	}

	pub, ok := keys.PublicKey(header.Kid)
	if !ok {
		return IDTokenClaims{}, fmt.Errorf("%w: kid %q", ErrIDTokenKeyNotFound, header.Kid)
	}

	sig, err := base64.RawURLEncoding.DecodeString(sigSeg)
	if err != nil {
		return IDTokenClaims{}, fmt.Errorf("%w: signature base64: %v", ErrIDTokenMalformed, err)
	}

	hashed := sha256.Sum256([]byte(signingInput))
	if err := rsa.VerifyPKCS1v15(pub, crypto.SHA256, hashed[:], sig); err != nil {
		return IDTokenClaims{}, ErrIDTokenSignatureMismatch
	}

	payloadBytes, err := base64.RawURLEncoding.DecodeString(payloadSeg)
	if err != nil {
		return IDTokenClaims{}, fmt.Errorf("%w: payload base64: %v", ErrIDTokenMalformed, err)
	}
	var claims IDTokenClaims
	if err := json.Unmarshal(payloadBytes, &claims); err != nil {
		return IDTokenClaims{}, fmt.Errorf("%w: payload decode: %v", ErrIDTokenMalformed, err)
	}

	if claims.Iss != opts.Issuer {
		return IDTokenClaims{}, fmt.Errorf("%w: got %q, want %q", ErrIDTokenIssuerMismatch, claims.Iss, opts.Issuer)
	}
	audOK := false
	for _, a := range claims.Aud {
		if a == opts.Audience {
			audOK = true
			break
		}
	}
	if !audOK {
		return IDTokenClaims{}, fmt.Errorf("%w: %v does not contain %q", ErrIDTokenAudienceMismatch, claims.Aud, opts.Audience)
	}

	now := time.Now()
	if claims.Exp > 0 && time.Unix(claims.Exp, 0).Add(skew).Before(now) {
		return IDTokenClaims{}, ErrIDTokenExpired
	}
	if claims.Iat > 0 && time.Unix(claims.Iat, 0).Add(-skew).After(now) {
		return IDTokenClaims{}, ErrIDTokenIssuedInFuture
	}

	return claims, nil
}

// ValidateIDToken is a convenience wrapper that fetches the JWK set fresh
// from jwksURI on every call and validates idToken using the Client's
// OAuth2 client_id as the expected audience. Production callers should
// share a JWKSCache and call the package-level ValidateIDToken directly.
func (c *Client) ValidateIDToken(ctx context.Context, idToken, issuer, jwksURI string) (IDTokenClaims, error) {
	if c.oauthConfig == nil {
		return IDTokenClaims{}, ErrOAuthConfigNotSet
	}
	keys, err := FetchJWKS(ctx, c.httpClient, jwksURI)
	if err != nil {
		return IDTokenClaims{}, err
	}
	return ValidateIDToken(idToken, keys, IDTokenValidationOptions{
		Issuer:   issuer,
		Audience: c.oauthConfig.ClientID,
	})
}

// RevokeToken posts to the OIDC revocation endpoint to invalidate an access
// or refresh token. The token's value is sent in the request body and the
// Client's OAuth2 client_id / secret are used as HTTP basic auth, as
// required by the Intuit revocation endpoint.
//
// revocationEndpoint should come from the OIDC discovery document.
// tokenValue can be either an access_token or a refresh_token; revoking a
// refresh_token also revokes its associated access_tokens.
func (c *Client) RevokeToken(ctx context.Context, revocationEndpoint, tokenValue string) error {
	if c.oauthConfig == nil {
		return ErrOAuthConfigNotSet
	}
	if revocationEndpoint == "" {
		return fmt.Errorf("revocation endpoint is required")
	}
	if tokenValue == "" {
		return fmt.Errorf("token value is required")
	}

	form := url.Values{}
	form.Set("token", tokenValue)
	body := strings.NewReader(form.Encode())

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, revocationEndpoint, body)
	if err != nil {
		return fmt.Errorf("build revoke request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth(c.oauthConfig.ClientID, c.oauthConfig.ClientSecret)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("revoke request: %w", err)
	}
	defer resp.Body.Close()

	// Per the spec, 200 indicates success. Some implementations also use
	// 204. Anything else is an error; surface the status and (truncated)
	// body for diagnostics.
	if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusNoContent {
		_, _ = io.Copy(io.Discard, resp.Body)
		return nil
	}
	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
	return fmt.Errorf("revoke endpoint error: status %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
}
