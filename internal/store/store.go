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

		CREATE TABLE IF NOT EXISTS persona (
			id            INTEGER PRIMARY KEY,
			name          TEXT    NOT NULL,
			description   TEXT,
			system_prompt TEXT    NOT NULL,
			created_by    INTEGER REFERENCES user(id),
			is_global     INTEGER NOT NULL DEFAULT 0,
			created_at    TEXT    NOT NULL,
			updated_at    TEXT    NOT NULL
		);

		CREATE TABLE IF NOT EXISTS conversation (
			id         INTEGER PRIMARY KEY,
			user_id    INTEGER NOT NULL REFERENCES user(id),
			persona_id INTEGER REFERENCES persona(id),
			title      TEXT,
			created_at TEXT    NOT NULL,
			updated_at TEXT    NOT NULL
		);

		CREATE TABLE IF NOT EXISTS message (
			id              INTEGER PRIMARY KEY,
			conversation_id INTEGER NOT NULL REFERENCES conversation(id),
			role            TEXT    NOT NULL,
			content         TEXT    NOT NULL,
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
			allow_editing INTEGER NOT NULL DEFAULT 0,
			created_at    TEXT    NOT NULL,
			updated_at    TEXT    NOT NULL
		);
	`)
	return err
}
