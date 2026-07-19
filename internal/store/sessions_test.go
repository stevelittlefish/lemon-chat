package store

import (
	"errors"
	"testing"
	"time"
)

func TestDeleteExpiredSessions(t *testing.T) {
	s := newTestStore(t)
	user, err := s.CreateUser("session-test-user", nil, false, nil)
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	insertSession := func(id string, expiresAt time.Time) {
		t.Helper()
		_, err := s.db.Exec(
			`INSERT INTO session (id, user_id, created_at, expires_at) VALUES (?, ?, ?, ?)`,
			id, user.ID, now(), expiresAt.UTC().Format(time.RFC3339),
		)
		if err != nil {
			t.Fatalf("insert session %q: %v", id, err)
		}
	}

	insertSession("expired", time.Now().Add(-time.Hour))
	insertSession("active", time.Now().Add(time.Hour))

	deleted, err := s.DeleteExpiredSessions()
	if err != nil {
		t.Fatalf("delete expired sessions: %v", err)
	}
	if deleted != 1 {
		t.Fatalf("deleted %d sessions, want 1", deleted)
	}

	if _, err := s.SessionUserID("expired"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expired session lookup: got %v, want ErrNotFound", err)
	}
	if got, err := s.SessionUserID("active"); err != nil || got != user.ID {
		t.Fatalf("active session lookup: got user %d, err %v; want user %d", got, err, user.ID)
	}

	deleted, err = s.DeleteExpiredSessions()
	if err != nil {
		t.Fatalf("repeat cleanup: %v", err)
	}
	if deleted != 0 {
		t.Fatalf("repeat cleanup deleted %d sessions, want 0", deleted)
	}
}
