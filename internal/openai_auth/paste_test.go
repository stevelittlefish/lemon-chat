package openai_auth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestParsePasted(t *testing.T) {
	cases := []struct {
		name      string
		in        string
		wantCode  string
		wantState string
		wantErr   bool
	}{
		{"bare code", "abc123", "abc123", "", false},
		{"full url", "http://localhost:1455/auth/callback?code=xyz&state=st1", "xyz", "st1", false},
		{"raw query", "code=xyz&state=st1", "xyz", "st1", false},
		{"error redirect", "http://localhost:1455/auth/callback?error=access_denied&error_description=nope", "", "", true},
		{"empty", "  ", "", "", true},
		{"url no code", "http://localhost:1455/auth/callback?state=st1", "", "", true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			code, state, err := ParsePasted(c.in)
			if (err != nil) != c.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, c.wantErr)
			}
			if err == nil && (code != c.wantCode || state != c.wantState) {
				t.Errorf("got code=%q state=%q, want code=%q state=%q", code, state, c.wantCode, c.wantState)
			}
		})
	}
}

func TestBeginProducesAuthorizeURLAndPending(t *testing.T) {
	authURL, pending, err := Begin()
	if err != nil {
		t.Fatalf("Begin: %v", err)
	}
	if pending.Verifier == "" || pending.State == "" {
		t.Fatal("expected verifier and state in pending login")
	}
	u, err := url.Parse(authURL)
	if err != nil {
		t.Fatalf("parse authorize URL: %v", err)
	}
	q := u.Query()
	if q.Get("state") != pending.State {
		t.Error("authorize URL state must match pending state")
	}
	if q.Get("redirect_uri") != redirectURI() {
		t.Errorf("redirect_uri = %q, want fixed loopback %q", q.Get("redirect_uri"), redirectURI())
	}
	if q.Get("originator") != originator || q.Get("codex_cli_simplified_flow") != "true" {
		t.Error("missing codex-flow authorize params")
	}
}

func TestCompleteExchangesPastedCode(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		if r.Form.Get("code") != "the-code" || r.Form.Get("code_verifier") != "ver" {
			t.Errorf("unexpected exchange form: %v", r.Form)
		}
		json.NewEncoder(w).Encode(tokenResponse{AccessToken: "at", RefreshToken: "rt", ExpiresIn: 3600})
	}))
	defer srv.Close()
	client := srv.Client()
	client.Transport = rewriteHost(srv.URL)

	p := PendingLogin{Verifier: "ver", State: "st1"}
	toks, err := Complete(context.Background(), client, "  the-code  ", "st1", p)
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if toks.AccessToken != "at" {
		t.Errorf("access token = %q", toks.AccessToken)
	}
}

func TestCompleteRejectsStateMismatch(t *testing.T) {
	p := PendingLogin{Verifier: "ver", State: "st1"}
	_, err := Complete(context.Background(), nil, "code", "different", p)
	if err == nil || !strings.Contains(err.Error(), "state mismatch") {
		t.Fatalf("expected state mismatch error, got %v", err)
	}
}
