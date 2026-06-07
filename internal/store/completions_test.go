package store

import (
	"errors"
	"slices"
	"testing"
	"time"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()
	s, err := Open(":memory:")
	if err != nil {
		t.Fatalf("open test store: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func TestGetCompletionErrors(t *testing.T) {
	s := newTestStore(t)
	user, err := s.CreateUser("testuser", nil, false, nil)
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	// Missing row → ErrNotFound
	_, err = s.GetCompletion(999, user.ID)
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("missing row: got %v, want ErrNotFound", err)
	}

	// Closed DB → real DB error, not ErrNotFound
	s.db.Close()
	_, err = s.GetCompletion(999, user.ID)
	if errors.Is(err, ErrNotFound) {
		t.Fatal("closed DB: got ErrNotFound, want a real DB error")
	}
	if err == nil {
		t.Fatal("closed DB: got nil, want error")
	}
}

func TestListUntitledEligibleCompletions(t *testing.T) {
	s := newTestStore(t)

	user, err := s.CreateUser("testuser", nil, false, nil)
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	old := time.Now().UTC().Add(-10 * time.Minute).Format(time.RFC3339)
	recent := time.Now().UTC().Format(time.RFC3339)

	ip := func(v int64) *int64 { return &v }
	sp := func(v string) *string { return &v }

	insert := func(title *string, promptTokens, completionTokens *int64, updatedAt string) int64 {
		t.Helper()
		res, err := s.db.Exec(
			`INSERT INTO completion (user_id, model, title, prompt_tokens, completion_tokens, created_at, updated_at)
			 VALUES (?, 'test-model', ?, ?, ?, ?, ?)`,
			user.ID, title, promptTokens, completionTokens, old, updatedAt,
		)
		if err != nil {
			t.Fatalf("insert completion: %v", err)
		}
		id, _ := res.LastInsertId()
		return id
	}

	// eligible: old, enough total tokens, no title
	eligible1 := insert(nil, ip(100), ip(50), old)
	// eligible: prompt + completion together cross the threshold
	eligible2 := insert(nil, ip(49), ip(1), old)
	// excluded: updated too recently
	insert(nil, ip(100), ip(50), recent)
	// excluded: total tokens below threshold (49 + 0 = 49)
	insert(nil, ip(49), nil, old)
	// excluded: already has a title
	insert(sp("existing title"), ip(100), ip(50), old)
	// excluded: no token data at all
	insert(nil, nil, nil, old)

	ids, err := s.ListUntitledEligibleCompletions(50)
	if err != nil {
		t.Fatalf("ListUntitledEligibleCompletions: %v", err)
	}

	want := []int64{eligible1, eligible2}
	slices.Sort(ids)
	slices.Sort(want)
	if len(ids) != len(want) {
		t.Fatalf("got ids %v, want %v", ids, want)
	}
	for i := range want {
		if ids[i] != want[i] {
			t.Fatalf("got ids %v, want %v", ids, want)
		}
	}
}
