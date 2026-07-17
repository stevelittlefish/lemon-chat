package openai_auth

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"
)

// TokenStore persists the single shared token. internal/store.Store satisfies
// this via a thin adapter; the interface keeps this package free of a store
// dependency (and an import cycle).
type TokenStore interface {
	LoadTokens() (Tokens, bool, error)
	SaveTokens(Tokens) error
	DeleteTokens() error
}

// Provider hands out a live access token, refreshing and persisting it when it
// nears expiry. It is safe for concurrent use by all request handlers.
type Provider struct {
	store  TokenStore
	client *http.Client

	mu     sync.Mutex
	cached Tokens
	loaded bool
}

// NewProvider builds a Provider backed by store. client may be nil.
func NewProvider(store TokenStore, client *http.Client) *Provider {
	if client == nil {
		client = http.DefaultClient
	}
	return &Provider{store: store, client: client}
}

// SetTokens persists a freshly-obtained login result (e.g. from Login) and
// primes the cache.
func (p *Provider) SetTokens(t Tokens) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if err := p.store.SaveTokens(t); err != nil {
		return err
	}
	p.cached = t
	p.loaded = true
	return nil
}

// Linked reports whether a token is present (an account has been connected).
func (p *Provider) Linked() (bool, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if err := p.ensureLoadedLocked(); err != nil {
		return false, err
	}
	return p.cached.AccessToken != "", nil
}

// AccountID returns the linked ChatGPT account id, or "".
func (p *Provider) AccountID() (string, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if err := p.ensureLoadedLocked(); err != nil {
		return "", err
	}
	return p.cached.AccountID, nil
}

// Status reports whether an account is linked, its account id, and the access
// token's expiry (for display). A zero expiry means unknown.
func (p *Provider) Status() (linked bool, accountID string, expiry time.Time, err error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if err = p.ensureLoadedLocked(); err != nil {
		return false, "", time.Time{}, err
	}
	return p.cached.AccessToken != "", p.cached.AccountID, p.cached.Expiry, nil
}

// Unlink deletes the stored token and clears the cache (the "disconnect" action).
func (p *Provider) Unlink() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if err := p.store.DeleteTokens(); err != nil {
		return err
	}
	p.cached = Tokens{}
	p.loaded = true
	return nil
}

// Token returns a non-expired access token, refreshing if necessary. It returns
// an error if no account is linked or a refresh fails.
func (p *Provider) Token(ctx context.Context) (string, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if err := p.ensureLoadedLocked(); err != nil {
		return "", err
	}
	if p.cached.AccessToken == "" {
		return "", fmt.Errorf("openai_auth: no OpenAI account linked; run login first")
	}
	if !p.cached.Expired() {
		return p.cached.AccessToken, nil
	}
	if p.cached.RefreshToken == "" {
		return "", fmt.Errorf("openai_auth: token expired and no refresh token available; re-link required")
	}
	log.Printf("openai_auth: refreshing expired access token")
	refreshed, err := refresh(ctx, p.client, p.cached.RefreshToken)
	if err != nil {
		return "", fmt.Errorf("openai_auth: refreshing token: %w", err)
	}
	// Preserve account id across refresh if the new id_token omitted it.
	if refreshed.AccountID == "" {
		refreshed.AccountID = p.cached.AccountID
	}
	if err := p.store.SaveTokens(refreshed); err != nil {
		return "", fmt.Errorf("openai_auth: persisting refreshed token: %w", err)
	}
	p.cached = refreshed
	return p.cached.AccessToken, nil
}

// ensureLoadedLocked lazily loads the token from the store on first use. Caller
// must hold p.mu.
func (p *Provider) ensureLoadedLocked() error {
	if p.loaded {
		return nil
	}
	t, ok, err := p.store.LoadTokens()
	if err != nil {
		return err
	}
	if ok {
		p.cached = t
	}
	p.loaded = true
	return nil
}
