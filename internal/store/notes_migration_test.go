package store

import (
	"slices"
	"testing"
)

func TestMigrateRemoveNotes(t *testing.T) {
	s := newTestStore(t)

	// A fresh store is already past v47: the note table is gone.
	var n int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = 'note'`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Fatal("note table exists on a fresh store")
	}

	// Recreate the pre-v47 state: a note and characters using the notes tools.
	if _, err := s.db.Exec(`CREATE TABLE note (id INTEGER PRIMARY KEY, key TEXT NOT NULL, value TEXT NOT NULL DEFAULT '')`); err != nil {
		t.Fatal(err)
	}
	if _, err := s.db.Exec(`INSERT INTO note (key, value) VALUES ('u.memories', 'x')`); err != nil {
		t.Fatal(err)
	}
	user, err := s.CreateUser("u", nil, false, nil)
	if err != nil {
		t.Fatal(err)
	}
	withNotes, err := s.CreateCharacter("with notes", "m", nil, nil, nil, user.ID, "public", false,
		[]string{"get_time", "notes", "note_to_self", "note_load"})
	if err != nil {
		t.Fatal(err)
	}
	without, err := s.CreateCharacter("without", "m", nil, nil, nil, user.ID, "public", false,
		[]string{"roll_dice"})
	if err != nil {
		t.Fatal(err)
	}

	if err := s.migrateRemoveNotes(); err != nil {
		t.Fatalf("migrateRemoveNotes: %v", err)
	}

	if err := s.db.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = 'note'`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Error("note table still exists")
	}
	for _, tc := range []struct {
		id   int64
		want []string
	}{
		{withNotes.ID, []string{"get_time", "note_to_self"}},
		{without.ID, []string{"roll_dice"}},
	} {
		c, err := s.GetCharacter(tc.id)
		if err != nil {
			t.Fatal(err)
		}
		if !slices.Equal(c.Tools, tc.want) {
			t.Errorf("character %d tools = %v, want %v", tc.id, c.Tools, tc.want)
		}
	}
}
