package store

import (
	"database/sql"
	"time"

	_ "modernc.org/sqlite"
)

type Store struct {
	db *sql.DB
}

func Open(path string) (*Store, error) {
	db, err := sql.Open("sqlite", path+"?_foreign_keys=on&_journal_mode=WAL")
	if err != nil {
		return nil, err
	}
	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		db.Close()
		return nil, err
	}
	return s, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

func now() string {
	return time.Now().UTC().Format(time.RFC3339)
}

func (s *Store) migrate() error {
	_, err := s.db.Exec(`
		CREATE TABLE IF NOT EXISTS user (
			id            INTEGER PRIMARY KEY,
			username      TEXT    NOT NULL UNIQUE,
			password_hash TEXT,
			is_admin      INTEGER NOT NULL DEFAULT 0,
			created_at    TEXT    NOT NULL
		);

		CREATE TABLE IF NOT EXISTS conversation (
			id           INTEGER PRIMARY KEY,
			user_id      INTEGER NOT NULL REFERENCES user(id),
			model        TEXT,
			character_id INTEGER REFERENCES character(id),
			title        TEXT,
			created_at   TEXT    NOT NULL,
			updated_at   TEXT    NOT NULL,
			CHECK ((model IS NOT NULL) != (character_id IS NOT NULL))
		);

		CREATE TABLE IF NOT EXISTS message (
			id              INTEGER PRIMARY KEY,
			conversation_id INTEGER NOT NULL REFERENCES conversation(id),
			role            TEXT    NOT NULL,
			content         TEXT    NOT NULL,
			name            TEXT,
			created_at      TEXT    NOT NULL
		);

		CREATE TABLE IF NOT EXISTS session (
			id         TEXT    PRIMARY KEY,
			user_id    INTEGER NOT NULL REFERENCES user(id),
			created_at TEXT    NOT NULL,
			expires_at TEXT    NOT NULL
		);

		CREATE TABLE IF NOT EXISTS character (
			id            INTEGER PRIMARY KEY,
			name          TEXT    NOT NULL,
			model         TEXT    NOT NULL,
			system_prompt TEXT,
			first_message TEXT,
			created_by    INTEGER NOT NULL REFERENCES user(id),
			visibility    TEXT    NOT NULL DEFAULT 'private',
			created_at    TEXT    NOT NULL,
			updated_at    TEXT    NOT NULL
		);

		CREATE TABLE IF NOT EXISTS schema_version (
			version INTEGER NOT NULL
		);

		INSERT INTO schema_version (version)
		SELECT 1 WHERE NOT EXISTS (SELECT 1 FROM schema_version);
	`)
	return err
}
