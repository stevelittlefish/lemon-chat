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
		`INSERT INTO sessions (id, user_id, created_at, expires_at) VALUES (?, ?, ?, ?)`,
		id, userID, t, expires,
	)
	return id, err
}

func (s *Store) SessionUserID(sessionID string) (int64, error) {
	var userID int64
	var expiresAt string
	err := s.db.QueryRow(
		`SELECT user_id, expires_at FROM sessions WHERE id = ?`, sessionID,
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
	_, err := s.db.Exec(`DELETE FROM sessions WHERE id = ?`, sessionID)
	return err
}
