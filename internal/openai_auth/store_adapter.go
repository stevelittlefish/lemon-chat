package openai_auth

import "github.com/stevelittlefish/lemon-chat/internal/store"

// StoreAdapter adapts *store.Store to the TokenStore interface, translating
// between this package's Tokens and the store's OAuthToken row.
type StoreAdapter struct {
	Store *store.Store
}

// NewStoreAdapter wraps s so it can back a Provider.
func NewStoreAdapter(s *store.Store) *StoreAdapter {
	return &StoreAdapter{Store: s}
}

// LoadTokens implements TokenStore.
func (a *StoreAdapter) LoadTokens() (Tokens, bool, error) {
	row, ok, err := a.Store.GetOAuthToken()
	if err != nil || !ok {
		return Tokens{}, ok, err
	}
	return Tokens{
		AccessToken:  row.AccessToken,
		RefreshToken: row.RefreshToken,
		IDToken:      row.IDToken,
		AccountID:    row.AccountID,
		Expiry:       row.Expiry,
	}, true, nil
}

// SaveTokens implements TokenStore.
func (a *StoreAdapter) SaveTokens(t Tokens) error {
	return a.Store.UpsertOAuthToken(store.OAuthToken{
		Provider:     "openai",
		AccessToken:  t.AccessToken,
		RefreshToken: t.RefreshToken,
		IDToken:      t.IDToken,
		AccountID:    t.AccountID,
		Expiry:       t.Expiry,
	})
}

// DeleteTokens implements TokenStore.
func (a *StoreAdapter) DeleteTokens() error {
	return a.Store.DeleteOAuthToken()
}
