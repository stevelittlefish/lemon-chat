package store

import (
	"testing"
	"time"
)

func TestOAuthTokenRoundTrip(t *testing.T) {
	s := newTestStore(t)

	if _, ok, err := s.GetOAuthToken(); err != nil || ok {
		t.Fatalf("expected no token initially, got ok=%v err=%v", ok, err)
	}

	exp := time.Now().Add(time.Hour).UTC().Truncate(time.Second)
	tok := OAuthToken{
		Provider:     "openai",
		AccessToken:  "at",
		RefreshToken: "rt",
		IDToken:      "idt",
		AccountID:    "acct_1",
		Expiry:       exp,
	}
	if err := s.UpsertOAuthToken(tok); err != nil {
		t.Fatalf("upsert: %v", err)
	}

	got, ok, err := s.GetOAuthToken()
	if err != nil || !ok {
		t.Fatalf("get after upsert: ok=%v err=%v", ok, err)
	}
	if got.AccessToken != "at" || got.RefreshToken != "rt" || got.AccountID != "acct_1" {
		t.Errorf("unexpected token: %+v", got)
	}
	if !got.Expiry.Equal(exp) {
		t.Errorf("expiry = %v, want %v", got.Expiry, exp)
	}

	// Upsert again to confirm single-row replace.
	if err := s.UpsertOAuthToken(OAuthToken{AccessToken: "at2", RefreshToken: "rt2", Expiry: exp}); err != nil {
		t.Fatalf("second upsert: %v", err)
	}
	got, _, _ = s.GetOAuthToken()
	if got.AccessToken != "at2" {
		t.Errorf("access token = %q, want at2", got.AccessToken)
	}

	if err := s.DeleteOAuthToken(); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, ok, _ := s.GetOAuthToken(); ok {
		t.Error("token should be gone after delete")
	}
}
