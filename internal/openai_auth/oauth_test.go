package openai_auth

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestNewPKCEChallengeMatchesVerifier(t *testing.T) {
	pk, err := newPKCE()
	if err != nil {
		t.Fatalf("newPKCE: %v", err)
	}
	if pk.Verifier == "" || pk.Challenge == "" {
		t.Fatal("expected non-empty verifier and challenge")
	}
	sum := sha256.Sum256([]byte(pk.Verifier))
	want := base64.RawURLEncoding.EncodeToString(sum[:])
	if pk.Challenge != want {
		t.Errorf("challenge = %q, want S256 of verifier %q", pk.Challenge, want)
	}
	if strings.ContainsAny(pk.Challenge, "+/=") {
		t.Errorf("challenge %q is not base64url", pk.Challenge)
	}
}

func TestAuthorizeURL(t *testing.T) {
	raw := authorizeURL("chal123", "state456")
	u, err := url.Parse(raw)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	q := u.Query()
	checks := map[string]string{
		"response_type":         "code",
		"client_id":             clientID,
		"code_challenge":        "chal123",
		"code_challenge_method": "S256",
		"state":                 "state456",
		"redirect_uri":          redirectURI(),
	}
	for k, want := range checks {
		if got := q.Get(k); got != want {
			t.Errorf("query %s = %q, want %q", k, got, want)
		}
	}
}

// makeIDToken builds an unsigned JWT with the given claims payload.
func makeIDToken(t *testing.T, claims map[string]any) string {
	t.Helper()
	payload, err := json.Marshal(claims)
	if err != nil {
		t.Fatal(err)
	}
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"none"}`))
	body := base64.RawURLEncoding.EncodeToString(payload)
	return header + "." + body + ".sig"
}

func TestAccountIDFromIDToken(t *testing.T) {
	tok := makeIDToken(t, map[string]any{
		"https://api.openai.com/auth": map[string]any{"chatgpt_account_id": "acct_789"},
	})
	if got := accountIDFromIDToken(tok); got != "acct_789" {
		t.Errorf("accountID = %q, want acct_789", got)
	}
	if got := accountIDFromIDToken("not-a-jwt"); got != "" {
		t.Errorf("expected empty account id for malformed token, got %q", got)
	}
}

func TestExchangeCode(t *testing.T) {
	var gotForm url.Values
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != tokenPath {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		r.ParseForm()
		gotForm = r.Form
		json.NewEncoder(w).Encode(tokenResponse{
			AccessToken:  "at",
			RefreshToken: "rt",
			IDToken:      makeIDToken(t, map[string]any{"chatgpt_account_id": "acct_1"}),
			ExpiresIn:    3600,
		})
	}))
	defer srv.Close()

	client := srv.Client()
	// Point the exchange at the test server by overriding the base via a custom
	// round-trip: rewrite the issuer host.
	client.Transport = rewriteHost(srv.URL)

	toks, err := exchangeCode(context.Background(), client, "the-code", "the-verifier")
	if err != nil {
		t.Fatalf("exchangeCode: %v", err)
	}
	if toks.AccessToken != "at" || toks.RefreshToken != "rt" || toks.AccountID != "acct_1" {
		t.Errorf("unexpected tokens: %+v", toks)
	}
	if toks.Expired() {
		t.Error("fresh token should not be expired")
	}
	if gotForm.Get("grant_type") != "authorization_code" || gotForm.Get("code") != "the-code" || gotForm.Get("code_verifier") != "the-verifier" {
		t.Errorf("unexpected form: %v", gotForm)
	}
}

func TestRefreshPreservesRefreshTokenWhenOmitted(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(tokenResponse{AccessToken: "at2", ExpiresIn: 3600})
	}))
	defer srv.Close()
	client := srv.Client()
	client.Transport = rewriteHost(srv.URL)

	toks, err := refresh(context.Background(), client, "old-refresh")
	if err != nil {
		t.Fatalf("refresh: %v", err)
	}
	if toks.RefreshToken != "old-refresh" {
		t.Errorf("refresh token = %q, want carried-over old-refresh", toks.RefreshToken)
	}
	if toks.AccessToken != "at2" {
		t.Errorf("access token = %q, want at2", toks.AccessToken)
	}
}

func TestTokenEndpointError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(tokenResponse{Error: "invalid_grant", ErrorDesc: "bad code"})
	}))
	defer srv.Close()
	client := srv.Client()
	client.Transport = rewriteHost(srv.URL)

	if _, err := exchangeCode(context.Background(), client, "x", "y"); err == nil {
		t.Fatal("expected error from token endpoint")
	}
}

func TestExpired(t *testing.T) {
	if !(Tokens{}).Expired() {
		t.Error("zero-expiry token should be expired")
	}
	if (Tokens{Expiry: time.Now().Add(time.Hour)}).Expired() {
		t.Error("token expiring in an hour should not be expired")
	}
	if !(Tokens{Expiry: time.Now().Add(10 * time.Second)}).Expired() {
		t.Error("token within refresh skew should be treated as expired")
	}
}

// rewriteHost returns a RoundTripper that redirects requests aimed at the
// hardcoded issuer to the test server URL.
func rewriteHost(testURL string) http.RoundTripper {
	base, _ := url.Parse(testURL)
	return roundTripFunc(func(req *http.Request) (*http.Response, error) {
		req.URL.Scheme = base.Scheme
		req.URL.Host = base.Host
		return http.DefaultTransport.RoundTrip(req)
	})
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
