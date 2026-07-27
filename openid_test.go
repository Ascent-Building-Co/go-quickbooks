package quickbooks

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
)

// signedToken is a helper that builds and signs an RS256 JWT with the given
// key, kid, and claim payload (which must JSON-marshal). Returns the
// compact-serialized token.
func signedToken(t *testing.T, key *rsa.PrivateKey, kid string, claims any) string {
	t.Helper()
	header := idTokenHeader{Kid: kid, Alg: idTokenAlgorithm, Typ: "JWT"}
	headerBytes, err := json.Marshal(header)
	require.NoError(t, err)

	payloadBytes, err := json.Marshal(claims)
	require.NoError(t, err)

	signingInput := base64.RawURLEncoding.EncodeToString(headerBytes) + "." +
		base64.RawURLEncoding.EncodeToString(payloadBytes)

	hashed := sha256.Sum256([]byte(signingInput))
	sig, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, hashed[:])
	require.NoError(t, err)

	return signingInput + "." + base64.RawURLEncoding.EncodeToString(sig)
}

// jwkFromKey builds a JWK entry for the given RSA public key under kid.
func jwkFromKey(t *testing.T, kid string, pub *rsa.PublicKey) JWK {
	t.Helper()
	eBytes := big.NewInt(int64(pub.E)).Bytes()
	return JWK{
		Kid: kid,
		Kty: "RSA",
		Alg: idTokenAlgorithm,
		Use: "sig",
		N:   base64.RawURLEncoding.EncodeToString(pub.N.Bytes()),
		E:   base64.RawURLEncoding.EncodeToString(eBytes),
	}
}

func TestExtractIDToken(t *testing.T) {
	t.Run("nil token", func(t *testing.T) {
		v, ok := ExtractIDToken(nil)
		require.False(t, ok)
		require.Empty(t, v)
	})

	t.Run("missing extra", func(t *testing.T) {
		tok := &oauth2.Token{AccessToken: "a"}
		v, ok := ExtractIDToken(tok)
		require.False(t, ok)
		require.Empty(t, v)
	})

	t.Run("present", func(t *testing.T) {
		tok := (&oauth2.Token{AccessToken: "a"}).WithExtra(map[string]any{
			"id_token": "header.payload.sig",
		})
		v, ok := ExtractIDToken(tok)
		require.True(t, ok)
		require.Equal(t, "header.payload.sig", v)
	})

	t.Run("empty string", func(t *testing.T) {
		tok := (&oauth2.Token{}).WithExtra(map[string]any{"id_token": ""})
		v, ok := ExtractIDToken(tok)
		require.False(t, ok)
		require.Empty(t, v)
	})
}

func TestParseIDToken(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	claims := map[string]any{
		"sub":     "user-1",
		"aud":     "audience-1",
		"realmId": "12345",
		"iss":     "https://example.test",
		"iat":     time.Now().Unix(),
		"exp":     time.Now().Add(1 * time.Hour).Unix(),
	}
	token := signedToken(t, key, "kid-1", claims)

	parsed, err := ParseIDToken(token)
	require.NoError(t, err)
	assert.Equal(t, "user-1", parsed.Sub)
	assert.Equal(t, []string{"audience-1"}, parsed.Aud)
	assert.Equal(t, "12345", parsed.RealmID)
	assert.Equal(t, "https://example.test", parsed.Iss)

	t.Run("malformed", func(t *testing.T) {
		_, err := ParseIDToken("not-a-jwt")
		require.ErrorIs(t, err, ErrIDTokenMalformed)
	})

	t.Run("audience as array", func(t *testing.T) {
		claims := map[string]any{
			"sub": "u",
			"aud": []string{"a", "b"},
			"iss": "https://x",
			"exp": time.Now().Add(1 * time.Hour).Unix(),
			"iat": time.Now().Unix(),
		}
		tok := signedToken(t, key, "kid-1", claims)
		parsed, err := ParseIDToken(tok)
		require.NoError(t, err)
		assert.Equal(t, []string{"a", "b"}, parsed.Aud)
	})

	t.Run("bad payload base64", func(t *testing.T) {
		_, err := ParseIDToken("aGVhZGVy.!notbase64.c2ln")
		require.ErrorIs(t, err, ErrIDTokenMalformed)
	})
}

func TestValidateIDToken(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	const (
		kid      = "kid-1"
		issuer   = "https://oauth.platform.intuit.com/op/v1"
		audience = "client-id-xyz"
	)
	keys := &JWKSet{Keys: []JWK{jwkFromKey(t, kid, &key.PublicKey)}}

	now := time.Now()
	baseClaims := map[string]any{
		"sub":     "user-1",
		"aud":     []string{audience},
		"realmId": "12345",
		"iss":     issuer,
		"iat":     now.Unix(),
		"exp":     now.Add(1 * time.Hour).Unix(),
	}

	t.Run("happy path with array audience", func(t *testing.T) {
		tok := signedToken(t, key, kid, baseClaims)
		claims, err := ValidateIDToken(tok, keys, IDTokenValidationOptions{
			Issuer: issuer, Audience: audience,
		})
		require.NoError(t, err)
		assert.Equal(t, "user-1", claims.Sub)
		assert.Equal(t, "12345", claims.RealmID)
	})

	t.Run("happy path with string audience", func(t *testing.T) {
		c := cloneClaims(baseClaims)
		c["aud"] = audience
		tok := signedToken(t, key, kid, c)
		_, err := ValidateIDToken(tok, keys, IDTokenValidationOptions{
			Issuer: issuer, Audience: audience,
		})
		require.NoError(t, err)
	})

	t.Run("unknown kid", func(t *testing.T) {
		tok := signedToken(t, key, "no-such-kid", baseClaims)
		_, err := ValidateIDToken(tok, keys, IDTokenValidationOptions{
			Issuer: issuer, Audience: audience,
		})
		require.ErrorIs(t, err, ErrIDTokenKeyNotFound)
	})

	t.Run("signature mismatch (mutated payload)", func(t *testing.T) {
		tok := signedToken(t, key, kid, baseClaims)
		// Replace the payload segment with an alternative valid JSON of the
		// same length-ish; the signature was over the original bytes.
		evil := cloneClaims(baseClaims)
		evil["sub"] = "attacker"
		evilPayload, _ := json.Marshal(evil)
		parts := strings.Split(tok, ".")
		parts[1] = base64.RawURLEncoding.EncodeToString(evilPayload)
		mutated := strings.Join(parts, ".")
		_, err := ValidateIDToken(mutated, keys, IDTokenValidationOptions{
			Issuer: issuer, Audience: audience,
		})
		require.ErrorIs(t, err, ErrIDTokenSignatureMismatch)
	})

	t.Run("expired", func(t *testing.T) {
		c := cloneClaims(baseClaims)
		c["iat"] = now.Add(-2 * time.Hour).Unix()
		c["exp"] = now.Add(-1 * time.Hour).Unix()
		tok := signedToken(t, key, kid, c)
		_, err := ValidateIDToken(tok, keys, IDTokenValidationOptions{
			Issuer: issuer, Audience: audience,
		})
		require.ErrorIs(t, err, ErrIDTokenExpired)
	})

	t.Run("issued in future", func(t *testing.T) {
		c := cloneClaims(baseClaims)
		c["iat"] = now.Add(10 * time.Minute).Unix()
		c["exp"] = now.Add(11 * time.Hour).Unix()
		tok := signedToken(t, key, kid, c)
		_, err := ValidateIDToken(tok, keys, IDTokenValidationOptions{
			Issuer: issuer, Audience: audience,
			ClockSkew: 30 * time.Second,
		})
		require.ErrorIs(t, err, ErrIDTokenIssuedInFuture)
	})

	t.Run("wrong issuer", func(t *testing.T) {
		c := cloneClaims(baseClaims)
		c["iss"] = "https://attacker.test"
		tok := signedToken(t, key, kid, c)
		_, err := ValidateIDToken(tok, keys, IDTokenValidationOptions{
			Issuer: issuer, Audience: audience,
		})
		require.ErrorIs(t, err, ErrIDTokenIssuerMismatch)
	})

	t.Run("wrong audience", func(t *testing.T) {
		c := cloneClaims(baseClaims)
		c["aud"] = []string{"someone-else"}
		tok := signedToken(t, key, kid, c)
		_, err := ValidateIDToken(tok, keys, IDTokenValidationOptions{
			Issuer: issuer, Audience: audience,
		})
		require.ErrorIs(t, err, ErrIDTokenAudienceMismatch)
	})

	t.Run("unsupported alg", func(t *testing.T) {
		// Build a token with alg=HS256 manually (no signing actually
		// performed; validation should reject before any verification).
		header := map[string]string{"kid": kid, "alg": "HS256"}
		hb, _ := json.Marshal(header)
		pb, _ := json.Marshal(baseClaims)
		tok := base64.RawURLEncoding.EncodeToString(hb) + "." +
			base64.RawURLEncoding.EncodeToString(pb) + "." +
			base64.RawURLEncoding.EncodeToString([]byte("sig"))
		_, err := ValidateIDToken(tok, keys, IDTokenValidationOptions{
			Issuer: issuer, Audience: audience,
		})
		require.ErrorIs(t, err, ErrIDTokenAlgUnsupported)
	})

	t.Run("malformed jwt", func(t *testing.T) {
		_, err := ValidateIDToken("only.two", keys, IDTokenValidationOptions{
			Issuer: issuer, Audience: audience,
		})
		require.ErrorIs(t, err, ErrIDTokenMalformed)
	})

	t.Run("missing kid", func(t *testing.T) {
		header := map[string]string{"alg": idTokenAlgorithm}
		hb, _ := json.Marshal(header)
		pb, _ := json.Marshal(baseClaims)
		tok := base64.RawURLEncoding.EncodeToString(hb) + "." +
			base64.RawURLEncoding.EncodeToString(pb) + "." +
			base64.RawURLEncoding.EncodeToString([]byte("sig"))
		_, err := ValidateIDToken(tok, keys, IDTokenValidationOptions{
			Issuer: issuer, Audience: audience,
		})
		require.ErrorIs(t, err, ErrIDTokenMalformed)
	})

	t.Run("missing options", func(t *testing.T) {
		tok := signedToken(t, key, kid, baseClaims)
		_, err := ValidateIDToken(tok, keys, IDTokenValidationOptions{Audience: audience})
		require.Error(t, err)
		_, err = ValidateIDToken(tok, keys, IDTokenValidationOptions{Issuer: issuer})
		require.Error(t, err)
	})

	t.Run("clock skew tolerance for exp", func(t *testing.T) {
		c := cloneClaims(baseClaims)
		c["iat"] = now.Add(-2 * time.Hour).Unix()
		c["exp"] = now.Add(-30 * time.Second).Unix()
		tok := signedToken(t, key, kid, c)
		// 60s skew default should accept a 30s-old expiry.
		_, err := ValidateIDToken(tok, keys, IDTokenValidationOptions{
			Issuer: issuer, Audience: audience,
		})
		require.NoError(t, err)
	})
}

func cloneClaims(in map[string]any) map[string]any {
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func TestFetchJWKS(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	jwk := jwkFromKey(t, "kid-1", &key.PublicKey)
	set := JWKSet{Keys: []JWK{jwk}}

	t.Run("happy path", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			require.Equal(t, http.MethodGet, r.Method)
			require.Equal(t, "application/json", r.Header.Get("Accept"))
			require.NoError(t, json.NewEncoder(w).Encode(set))
		}))
		defer srv.Close()

		got, err := FetchJWKS(context.Background(), srv.Client(), srv.URL)
		require.NoError(t, err)
		require.Len(t, got.Keys, 1)
		assert.Equal(t, "kid-1", got.Keys[0].Kid)

		pub, ok := got.PublicKey("kid-1")
		require.True(t, ok)
		require.NotNil(t, pub)
		assert.Equal(t, key.PublicKey.N, pub.N)
		assert.Equal(t, key.PublicKey.E, pub.E)
	})

	t.Run("non-2xx", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, "boom", http.StatusInternalServerError)
		}))
		defer srv.Close()

		_, err := FetchJWKS(context.Background(), srv.Client(), srv.URL)
		require.Error(t, err)
	})

	t.Run("empty uri", func(t *testing.T) {
		_, err := FetchJWKS(context.Background(), nil, "")
		require.Error(t, err)
	})

	t.Run("malformed json", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = io.WriteString(w, "{not-json")
		}))
		defer srv.Close()

		_, err := FetchJWKS(context.Background(), srv.Client(), srv.URL)
		require.Error(t, err)
	})
}

func TestJWKSetPublicKey(t *testing.T) {
	t.Run("missing kid", func(t *testing.T) {
		set := &JWKSet{}
		_, ok := set.PublicKey("nope")
		require.False(t, ok)
	})

	t.Run("non-rsa kty", func(t *testing.T) {
		set := &JWKSet{Keys: []JWK{{Kid: "k", Kty: "EC"}}}
		_, ok := set.PublicKey("k")
		require.False(t, ok)
	})

	t.Run("invalid base64", func(t *testing.T) {
		set := &JWKSet{Keys: []JWK{{Kid: "k", Kty: "RSA", N: "!", E: "AQAB"}}}
		_, ok := set.PublicKey("k")
		require.False(t, ok)
	})

	t.Run("nil receiver", func(t *testing.T) {
		var set *JWKSet
		_, ok := set.PublicKey("k")
		require.False(t, ok)
	})
}

func TestJWKSCacheBehavior(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	set := JWKSet{Keys: []JWK{jwkFromKey(t, "kid-1", &key.PublicKey)}}

	var hits atomic.Int64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hits.Add(1)
		_ = json.NewEncoder(w).Encode(set)
	}))
	defer srv.Close()

	t.Run("caches within ttl", func(t *testing.T) {
		hits.Store(0)
		cache := NewJWKSCache(srv.Client(), srv.URL, 1*time.Hour)

		_, err := cache.Get(context.Background())
		require.NoError(t, err)
		_, err = cache.Get(context.Background())
		require.NoError(t, err)
		require.Equal(t, int64(1), hits.Load(), "expected single fetch within ttl")
	})

	t.Run("invalidate forces refetch", func(t *testing.T) {
		hits.Store(0)
		cache := NewJWKSCache(srv.Client(), srv.URL, 1*time.Hour)

		_, err := cache.Get(context.Background())
		require.NoError(t, err)
		cache.Invalidate()
		_, err = cache.Get(context.Background())
		require.NoError(t, err)
		require.Equal(t, int64(2), hits.Load())
	})

	t.Run("zero ttl resolves to default", func(t *testing.T) {
		cache := NewJWKSCache(srv.Client(), srv.URL, 0)
		require.Equal(t, DefaultJWKSCacheTTL, cache.ttl)
	})
}

func TestFetchUserInfo(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		const tokenValue = "atok-123"
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			require.Equal(t, http.MethodGet, r.Method)
			require.Equal(t, "Bearer "+tokenValue, r.Header.Get("Authorization"))
			require.Equal(t, "application/json", r.Header.Get("Accept"))
			_, _ = io.WriteString(w, `{
				"sub": "1182d6ec-2a1f-4aa3-af3f-bb3b95db45af",
				"email": "john@doe.com",
				"emailVerified": true,
				"givenName": "John",
				"familyName": "Doe",
				"phoneNumber": "+1 6305555555",
				"phoneNumberVerified": false,
				"address": {
					"streetAddress": "2007 saint julien ct",
					"locality": "mountain view",
					"region": "CA",
					"postalCode": "94043",
					"country": "US"
				}
			}`)
		}))
		defer srv.Close()

		info, err := FetchUserInfo(context.Background(), srv.Client(), srv.URL, tokenValue)
		require.NoError(t, err)
		assert.Equal(t, "1182d6ec-2a1f-4aa3-af3f-bb3b95db45af", info.Sub)
		assert.Equal(t, "john@doe.com", info.Email)
		assert.True(t, info.EmailVerified)
		assert.Equal(t, "John", info.GivenName)
		assert.Equal(t, "Doe", info.FamilyName)
		require.NotNil(t, info.Address)
		assert.Equal(t, "94043", info.Address.PostalCode)
	})

	t.Run("unauthorized", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, "nope", http.StatusUnauthorized)
		}))
		defer srv.Close()
		_, err := FetchUserInfo(context.Background(), srv.Client(), srv.URL, "tok")
		require.Error(t, err)
	})

	t.Run("missing args", func(t *testing.T) {
		_, err := FetchUserInfo(context.Background(), nil, "", "tok")
		require.Error(t, err)
		_, err = FetchUserInfo(context.Background(), nil, "https://x", "")
		require.Error(t, err)
	})

	t.Run("malformed json", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = io.WriteString(w, "{nope")
		}))
		defer srv.Close()
		_, err := FetchUserInfo(context.Background(), srv.Client(), srv.URL, "tok")
		require.Error(t, err)
	})
}

// staticTokenSource implements oauth2.TokenSource for testing without
// pulling oauth2.StaticTokenSource into the closure-of-tests namespace.
type staticTokenSource struct {
	token *oauth2.Token
	err   error
}

func (s staticTokenSource) Token() (*oauth2.Token, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.token, nil
}

func TestClientGetUserInfo(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "Bearer my-access", r.Header.Get("Authorization"))
		_, _ = io.WriteString(w, `{"sub":"u","email":"a@b.com","emailVerified":true}`)
	}))
	defer srv.Close()

	c, err := NewClient()
	require.NoError(t, err)

	t.Run("nil token source", func(t *testing.T) {
		_, err := c.GetUserInfo(context.Background(), nil, srv.URL)
		require.ErrorIs(t, err, ErrTokenSourceNil)
	})

	t.Run("token error", func(t *testing.T) {
		ts := staticTokenSource{err: errors.New("boom")}
		_, err := c.GetUserInfo(context.Background(), ts, srv.URL)
		require.Error(t, err)
	})

	t.Run("happy path", func(t *testing.T) {
		ts := staticTokenSource{token: &oauth2.Token{AccessToken: "my-access"}}
		info, err := c.GetUserInfo(context.Background(), ts, srv.URL)
		require.NoError(t, err)
		assert.Equal(t, "u", info.Sub)
		assert.True(t, info.EmailVerified)
	})
}

func TestClientValidateIDToken(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	const (
		kid      = "kid-validate"
		issuer   = "https://oauth.platform.intuit.com/op/v1"
		clientID = "client-id-test"
	)
	set := JWKSet{Keys: []JWK{jwkFromKey(t, kid, &key.PublicKey)}}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(set)
	}))
	defer srv.Close()

	c, err := NewClient(WithOAuthConfig(clientID, "secret", "https://cb", "", "", AccountingScope))
	require.NoError(t, err)

	now := time.Now()
	tok := signedToken(t, key, kid, map[string]any{
		"sub":     "user",
		"aud":     []string{clientID},
		"realmId": "12345",
		"iss":     issuer,
		"iat":     now.Unix(),
		"exp":     now.Add(1 * time.Hour).Unix(),
	})

	claims, err := c.ValidateIDToken(context.Background(), tok, issuer, srv.URL)
	require.NoError(t, err)
	assert.Equal(t, "user", claims.Sub)
	assert.Equal(t, "12345", claims.RealmID)

	t.Run("oauth not configured", func(t *testing.T) {
		bare, err := NewClient()
		require.NoError(t, err)
		_, err = bare.ValidateIDToken(context.Background(), tok, issuer, srv.URL)
		require.ErrorIs(t, err, ErrOAuthConfigNotSet)
	})
}

func TestRevokeToken(t *testing.T) {
	const (
		clientID     = "id-1"
		clientSecret = "secret"
		tokenValue   = "rfsh-abc"
	)

	t.Run("happy path with form body and basic auth", func(t *testing.T) {
		var (
			gotMethod   string
			gotAuthUser string
			gotAuthPass string
			gotForm     url.Values
		)
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gotMethod = r.Method
			user, pass, _ := r.BasicAuth()
			gotAuthUser, gotAuthPass = user, pass
			require.NoError(t, r.ParseForm())
			gotForm = r.PostForm
			w.WriteHeader(http.StatusOK)
		}))
		defer srv.Close()

		c, err := NewClient(WithOAuthConfig(clientID, clientSecret, "https://cb", "", "", AccountingScope))
		require.NoError(t, err)

		require.NoError(t, c.RevokeToken(context.Background(), srv.URL, tokenValue))
		assert.Equal(t, http.MethodPost, gotMethod)
		assert.Equal(t, clientID, gotAuthUser)
		assert.Equal(t, clientSecret, gotAuthPass)
		assert.Equal(t, tokenValue, gotForm.Get("token"))
	})

	t.Run("non-success surfaces status", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, "bad token", http.StatusBadRequest)
		}))
		defer srv.Close()

		c, err := NewClient(WithOAuthConfig(clientID, clientSecret, "https://cb", "", "", AccountingScope))
		require.NoError(t, err)
		err = c.RevokeToken(context.Background(), srv.URL, tokenValue)
		require.Error(t, err)
		require.Contains(t, err.Error(), "400")
	})

	t.Run("oauth not configured", func(t *testing.T) {
		c, err := NewClient()
		require.NoError(t, err)
		err = c.RevokeToken(context.Background(), "https://example", tokenValue)
		require.ErrorIs(t, err, ErrOAuthConfigNotSet)
	})

	t.Run("missing args", func(t *testing.T) {
		c, err := NewClient(WithOAuthConfig(clientID, clientSecret, "https://cb", "", "", AccountingScope))
		require.NoError(t, err)
		require.Error(t, c.RevokeToken(context.Background(), "", tokenValue))
		require.Error(t, c.RevokeToken(context.Background(), "https://x", ""))
	})

	t.Run("204 also accepted", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		}))
		defer srv.Close()
		c, err := NewClient(WithOAuthConfig(clientID, clientSecret, "https://cb", "", "", AccountingScope))
		require.NoError(t, err)
		require.NoError(t, c.RevokeToken(context.Background(), srv.URL, tokenValue))
	})
}

// Sanity: confirm the JWT helper produces a token the validator accepts.
// (Guards against base64 padding regressions in the test helper.)
func TestSignedTokenRoundtrip(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	const aud = "audience"
	tok := signedToken(t, key, "k", map[string]any{
		"sub": "u",
		"aud": []string{aud},
		"iss": "issuer",
		"iat": time.Now().Unix(),
		"exp": time.Now().Add(time.Hour).Unix(),
	})
	keys := &JWKSet{Keys: []JWK{jwkFromKey(t, "k", &key.PublicKey)}}
	_, err = ValidateIDToken(tok, keys, IDTokenValidationOptions{
		Issuer: "issuer", Audience: aud,
	})
	require.NoError(t, err)
	_ = fmt.Sprintf // keep fmt referenced even if no error formatting in this test
}
