package store

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"time"
)

const sessionTTL = 30 * 24 * time.Hour

func (s *Store) CreateSession(userID int64) (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	id := hex.EncodeToString(b)
	t := now()
	expires := time.Now().UTC().Add(sessionTTL).Format(time.RFC3339)
	_, err := s.db.Exec(
		`INSERT INTO session (id, user_id, created_at, expires_at) VALUES (?, ?, ?, ?)`,
		id, userID, t, expires,
	)
	return id, err
}

func (s *Store) SessionUserID(sessionID string) (int64, error) {
	var userID int64
	var expiresAt string
	err := s.db.QueryRow(
		`SELECT user_id, expires_at FROM session WHERE id = ?`, sessionID,
	).Scan(&userID, &expiresAt)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, ErrNotFound
	}
	if err != nil {
		return 0, err
	}
	exp, err := time.Parse(time.RFC3339, expiresAt)
	if err != nil || time.Now().After(exp) {
		_ = s.DeleteSession(sessionID)
		return 0, ErrNotFound
	}
	return userID, nil
}

func (s *Store) DeleteSession(sessionID string) error {
	_, err := s.db.Exec(`DELETE FROM session WHERE id = ?`, sessionID)
	return err
}

// DeleteExpiredSessions removes every session whose expiry has passed. The
// comparison is lexicographic, which only matches chronological order because
// expires_at is always written as UTC RFC3339 (fixed width, "Z" suffix) — see
// CreateSession and now(). Writing an offset like "+01:00" would silently break
// this sweep.
func (s *Store) DeleteExpiredSessions() (int64, error) {
	result, err := s.db.Exec(`DELETE FROM session WHERE expires_at <= ?`, now())
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}
