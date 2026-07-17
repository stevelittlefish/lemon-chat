package store

import (
	"database/sql"
	"time"
)

// OAuthToken is the single shared OAuth credential lemon-chat uses for all
// users' requests to a provider (currently OpenAI). Only one row ever exists
// (id = 1); it is rotated in place on refresh.
type OAuthToken struct {
	Provider     string
	AccessToken  string
	RefreshToken string
	IDToken      string
	AccountID    string
	Expiry       time.Time
	UpdatedAt    time.Time
}

// GetOAuthToken returns the stored token and true, or a zero value and false if
// no login has been performed yet.
func (s *Store) GetOAuthToken() (OAuthToken, bool, error) {
	var (
		t                 OAuthToken
		expiry, updatedAt string
	)
	err := s.db.QueryRow(
		`SELECT provider, access_token, refresh_token, id_token, account_id, expiry, updated_at
		 FROM oauth_token WHERE id = 1`,
	).Scan(&t.Provider, &t.AccessToken, &t.RefreshToken, &t.IDToken, &t.AccountID, &expiry, &updatedAt)
	if err == sql.ErrNoRows {
		return OAuthToken{}, false, nil
	}
	if err != nil {
		return OAuthToken{}, false, err
	}
	t.Expiry, _ = time.Parse(time.RFC3339, expiry)
	t.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
	return t, true, nil
}

// UpsertOAuthToken inserts or replaces the single-row token.
func (s *Store) UpsertOAuthToken(t OAuthToken) error {
	if t.Provider == "" {
		t.Provider = "openai"
	}
	_, err := s.db.Exec(
		`INSERT INTO oauth_token (id, provider, access_token, refresh_token, id_token, account_id, expiry, updated_at)
		 VALUES (1, ?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT(id) DO UPDATE SET
		   provider = excluded.provider,
		   access_token = excluded.access_token,
		   refresh_token = excluded.refresh_token,
		   id_token = excluded.id_token,
		   account_id = excluded.account_id,
		   expiry = excluded.expiry,
		   updated_at = excluded.updated_at`,
		t.Provider, t.AccessToken, t.RefreshToken, t.IDToken, t.AccountID,
		t.Expiry.UTC().Format(time.RFC3339), now())
	return err
}

// DeleteOAuthToken removes the stored token (unlinking the account).
func (s *Store) DeleteOAuthToken() error {
	_, err := s.db.Exec(`DELETE FROM oauth_token WHERE id = 1`)
	return err
}
