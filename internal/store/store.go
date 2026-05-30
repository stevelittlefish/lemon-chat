package store

import (
	"database/sql"
	"log"
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
	version := 0

	var tableName string
	err := s.db.QueryRow(`SELECT name FROM sqlite_master WHERE type='table' AND name='schema_version'`).Scan(&tableName)
	if err == nil {
		if err := s.db.QueryRow(`SELECT COALESCE(MAX(version), 0) FROM schema_version`).Scan(&version); err != nil {
			return err
		}
	}
	log.Printf("store: schema version is %d", version)

	if version < 1 {
		log.Println("store: migrating v0 → v1 (create schema)")
		if _, err := s.db.Exec(`
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

			CREATE TABLE IF NOT EXISTS character_hidden_message (
				id           INTEGER PRIMARY KEY,
				character_id INTEGER NOT NULL REFERENCES character(id) ON DELETE CASCADE,
				role         TEXT    NOT NULL CHECK (role IN ('user', 'assistant')),
				content      TEXT    NOT NULL,
				sort_order   INTEGER NOT NULL DEFAULT 0,
				created_at   TEXT    NOT NULL
			);

			CREATE TABLE IF NOT EXISTS schema_version (
				version INTEGER NOT NULL
			);

			INSERT INTO schema_version (version) VALUES (1);
		`); err != nil {
			return err
		}
		version = 1
		log.Println("store: migration v0 → v1 complete")
	}

	if version < 2 {
		log.Println("store: migrating v1 → v2 (add timestamp to schema_version)")
		if _, err := s.db.Exec(`ALTER TABLE schema_version ADD COLUMN timestamp TEXT`); err != nil {
			return err
		}
		if _, err := s.db.Exec(`INSERT INTO schema_version (version, timestamp) VALUES (2, ?)`, now()); err != nil {
			return err
		}
		version = 2
		log.Println("store: migration v1 → v2 complete")
	}

	if version < 3 {
		log.Println("store: migrating v2 → v3 (add token and timing stats to message)")
		for _, stmt := range []string{
			`ALTER TABLE message ADD COLUMN prompt_tokens     INTEGER`,
			`ALTER TABLE message ADD COLUMN completion_tokens INTEGER`,
			`ALTER TABLE message ADD COLUMN total_time_ms     INTEGER`,
		} {
			if _, err := s.db.Exec(stmt); err != nil {
				return err
			}
		}
		if _, err := s.db.Exec(`INSERT INTO schema_version (version, timestamp) VALUES (3, ?)`, now()); err != nil {
			return err
		}
		version = 3
		log.Println("store: migration v2 → v3 complete")
	}

	if version < 4 {
		log.Println("store: migrating v3 → v4 (add auto_title to character)")
		if _, err := s.db.Exec(`ALTER TABLE character ADD COLUMN auto_title INTEGER NOT NULL DEFAULT 0`); err != nil {
			return err
		}
		if _, err := s.db.Exec(`INSERT INTO schema_version (version, timestamp) VALUES (4, ?)`, now()); err != nil {
			return err
		}
		version = 4
		log.Println("store: migration v3 → v4 complete")
	}

	if version < 5 {
		log.Println("store: migrating v4 → v5 (add display_name to user)")
		if _, err := s.db.Exec(`ALTER TABLE user ADD COLUMN display_name TEXT`); err != nil {
			return err
		}
		if _, err := s.db.Exec(`INSERT INTO schema_version (version, timestamp) VALUES (5, ?)`, now()); err != nil {
			return err
		}
		version = 5
		log.Println("store: migration v4 → v5 complete")
	}

	log.Printf("store: schema ready at version %d", version)
	return nil
}
