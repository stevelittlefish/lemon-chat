package openai_auth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// memStore is an in-memory TokenStore for tests.
type memStore struct {
	tok   Tokens
	ok    bool
	saves int
}

func (m *memStore) LoadTokens() (Tokens, bool, error) { return m.tok, m.ok, nil }
func (m *memStore) SaveTokens(t Tokens) error {
	m.tok, m.ok, m.saves = t, true, m.saves+1
	return nil
}
func (m *memStore) DeleteTokens() error {
	m.tok, m.ok = Tokens{}, false
	return nil
}

func TestProviderReturnsValidToken(t *testing.T) {
	store := &memStore{tok: Tokens{AccessToken: "live", Expiry: time.Now().Add(time.Hour)}, ok: true}
	p := NewProvider(store, nil)

	got, err := p.Token(context.Background())
	if err != nil {
		t.Fatalf("Token: %v", err)
	}
	if got != "live" {
		t.Errorf("token = %q, want live", got)
	}
	if store.saves != 0 {
		t.Errorf("expected no refresh/save, got %d saves", store.saves)
	}
}

func TestProviderRefreshesExpiredToken(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(tokenResponse{AccessToken: "fresh", RefreshToken: "rt2", ExpiresIn: 3600})
	}))
	defer srv.Close()
	client := srv.Client()
	client.Transport = rewriteHost(srv.URL)

	store := &memStore{
		tok: Tokens{AccessToken: "stale", RefreshToken: "rt1", AccountID: "acct_x", Expiry: time.Now().Add(-time.Hour)},
		ok:  true,
	}
	p := NewProvider(store, client)

	got, err := p.Token(context.Background())
	if err != nil {
		t.Fatalf("Token: %v", err)
	}
	if got != "fresh" {
		t.Errorf("token = %q, want fresh", got)
	}
	if store.saves != 1 {
		t.Errorf("expected 1 persisted refresh, got %d", store.saves)
	}
	if store.tok.AccountID != "acct_x" {
		t.Errorf("account id should survive refresh, got %q", store.tok.AccountID)
	}
}

func TestProviderErrorsWhenUnlinked(t *testing.T) {
	p := NewProvider(&memStore{}, nil)
	if _, err := p.Token(context.Background()); err == nil {
		t.Fatal("expected error when no account linked")
	}
	linked, err := p.Linked()
	if err != nil {
		t.Fatal(err)
	}
	if linked {
		t.Error("Linked should be false with no token")
	}
}

func TestProviderSetTokens(t *testing.T) {
	store := &memStore{}
	p := NewProvider(store, nil)
	if err := p.SetTokens(Tokens{AccessToken: "a", AccountID: "acct_1", Expiry: time.Now().Add(time.Hour)}); err != nil {
		t.Fatal(err)
	}
	if store.saves != 1 || !store.ok {
		t.Error("SetTokens should persist")
	}
	acct, _ := p.AccountID()
	if acct != "acct_1" {
		t.Errorf("account id = %q, want acct_1", acct)
	}
}
