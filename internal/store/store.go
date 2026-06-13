package store

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"

	_ "modernc.org/sqlite"
)

type Store struct {
	db *sql.DB
}

func Open(path string) (*Store, error) {
	db, err := sql.Open("sqlite", path+"?_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)")
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
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

			CREATE TABLE IF NOT EXISTS attachment (
				id              INTEGER PRIMARY KEY AUTOINCREMENT,
				tool_call_id    TEXT    NOT NULL,
				conversation_id INTEGER NOT NULL REFERENCES conversation(id),
				title           TEXT    NOT NULL,
				filename        TEXT    NOT NULL,
				mime_type       TEXT    NOT NULL,
				disk_path       TEXT    NOT NULL,
				created_at      TEXT    NOT NULL
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

	if version < 6 {
		log.Println("store: migrating v5 → v6 (add avatar_filename to user and character)")
		for _, stmt := range []string{
			`ALTER TABLE user ADD COLUMN avatar_filename TEXT`,
			`ALTER TABLE character ADD COLUMN avatar_filename TEXT`,
		} {
			if _, err := s.db.Exec(stmt); err != nil {
				return err
			}
		}
		if _, err := s.db.Exec(`INSERT INTO schema_version (version, timestamp) VALUES (6, ?)`, now()); err != nil {
			return err
		}
		version = 6
		log.Println("store: migration v5 → v6 complete")
	}

	if version < 7 {
		log.Println("store: migrating v6 → v7 (add character_id to message)")
		if _, err := s.db.Exec(`ALTER TABLE message ADD COLUMN character_id INTEGER REFERENCES character(id)`); err != nil {
			return err
		}
		if _, err := s.db.Exec(`INSERT INTO schema_version (version, timestamp) VALUES (7, ?)`, now()); err != nil {
			return err
		}
		version = 7
		log.Println("store: migration v6 → v7 complete")
	}

	if version < 8 {
		log.Println("store: migrating v7 → v8 (add title_prompt to character)")
		if _, err := s.db.Exec(`ALTER TABLE character ADD COLUMN title_prompt TEXT`); err != nil {
			return err
		}
		if _, err := s.db.Exec(`INSERT INTO schema_version (version, timestamp) VALUES (8, ?)`, now()); err != nil {
			return err
		}
		version = 8
		log.Println("store: migration v7 → v8 complete")
	}

	if version < 10 {
		log.Println("store: migrating v9 → v10 (add indexes on message, conversation, session)")
		for _, stmt := range []string{
			`CREATE INDEX IF NOT EXISTS idx_message_conversation_id ON message(conversation_id)`,
			`CREATE INDEX IF NOT EXISTS idx_conversation_user_id    ON conversation(user_id)`,
			`CREATE INDEX IF NOT EXISTS idx_conversation_created_at ON conversation(created_at)`,
			`CREATE INDEX IF NOT EXISTS idx_session_expires_at      ON session(expires_at)`,
		} {
			if _, err := s.db.Exec(stmt); err != nil {
				return err
			}
		}
		if _, err := s.db.Exec(`INSERT INTO schema_version (version, timestamp) VALUES (10, ?)`, now()); err != nil {
			return err
		}
		version = 10
		log.Println("store: migration v9 → v10 complete")
	}

	if version < 11 {
		log.Println("store: migrating v10 → v11 (create completion table)")
		if _, err := s.db.Exec(`
			CREATE TABLE IF NOT EXISTS completion (
				id         INTEGER PRIMARY KEY,
				user_id    INTEGER NOT NULL REFERENCES user(id) ON DELETE CASCADE,
				model      TEXT    NOT NULL,
				title      TEXT,
				created_at TEXT    NOT NULL,
				updated_at TEXT    NOT NULL
			);
			CREATE INDEX IF NOT EXISTS idx_completion_user_id ON completion(user_id);
		`); err != nil {
			return err
		}
		if _, err := s.db.Exec(`INSERT INTO schema_version (version, timestamp) VALUES (11, ?)`, now()); err != nil {
			return err
		}
		version = 11
		log.Println("store: migration v10 → v11 complete")
	}

	if version < 12 {
		log.Println("store: migrating v11 → v12 (add content to completion)")
		if _, err := s.db.Exec(`ALTER TABLE completion ADD COLUMN content TEXT`); err != nil {
			return err
		}
		if _, err := s.db.Exec(`INSERT INTO schema_version (version, timestamp) VALUES (12, ?)`, now()); err != nil {
			return err
		}
		version = 12
		log.Println("store: migration v11 → v12 complete")
	}

	if version < 13 {
		log.Println("store: migrating v12 → v13 (add prev_content and token counts to completion)")
		for _, stmt := range []string{
			`ALTER TABLE completion ADD COLUMN prev_content      TEXT`,
			`ALTER TABLE completion ADD COLUMN prompt_tokens     INTEGER`,
			`ALTER TABLE completion ADD COLUMN completion_tokens INTEGER`,
		} {
			if _, err := s.db.Exec(stmt); err != nil {
				return err
			}
		}
		if _, err := s.db.Exec(`INSERT INTO schema_version (version, timestamp) VALUES (13, ?)`, now()); err != nil {
			return err
		}
		version = 13
		log.Println("store: migration v12 → v13 complete")
	}

	if version < 14 {
		log.Println("store: migrating v13 → v14 (add undone flag to completion)")
		if _, err := s.db.Exec(`ALTER TABLE completion ADD COLUMN undone INTEGER NOT NULL DEFAULT 0`); err != nil {
			return err
		}
		if _, err := s.db.Exec(`INSERT INTO schema_version (version, timestamp) VALUES (14, ?)`, now()); err != nil {
			return err
		}
		version = 14
		log.Println("store: migration v13 → v14 complete")
	}

	if version < 15 {
		log.Println("store: migrating v14 → v15 (add tools to character, tool_calls and tool_call_id to message)")
		for _, stmt := range []string{
			`ALTER TABLE character ADD COLUMN tools TEXT`,
			`ALTER TABLE message ADD COLUMN tool_calls TEXT`,
			`ALTER TABLE message ADD COLUMN tool_call_id TEXT`,
		} {
			if _, err := s.db.Exec(stmt); err != nil {
				return err
			}
		}
		if _, err := s.db.Exec(`INSERT INTO schema_version (version, timestamp) VALUES (15, ?)`, now()); err != nil {
			return err
		}
		version = 15
		log.Println("store: migration v14 → v15 complete")
	}

	if version < 16 {
		log.Println("store: migrating v15 → v16 (create attachment table)")
		if _, err := s.db.Exec(`
			CREATE TABLE IF NOT EXISTS attachment (
				id              INTEGER PRIMARY KEY AUTOINCREMENT,
				tool_call_id    TEXT    NOT NULL,
				conversation_id INTEGER NOT NULL REFERENCES conversation(id),
				title           TEXT    NOT NULL,
				filename        TEXT    NOT NULL,
				mime_type       TEXT    NOT NULL,
				disk_path       TEXT    NOT NULL,
				created_at      TEXT    NOT NULL
			)
		`); err != nil {
			return err
		}
		if _, err := s.db.Exec(`INSERT INTO schema_version (version, timestamp) VALUES (16, ?)`, now()); err != nil {
			return err
		}
		version = 16
		log.Println("store: migration v15 → v16 complete")
	}

	if version < 17 {
		log.Println("store: migrating v16 → v17 (rename web_search to searxng in character tools)")
		rows, err := s.db.Query(`SELECT id, tools FROM character WHERE tools IS NOT NULL AND tools LIKE '%web_search%'`)
		if err != nil {
			return err
		}
		type row struct {
			id    int64
			tools string
		}
		var toUpdate []row
		for rows.Next() {
			var r row
			if err := rows.Scan(&r.id, &r.tools); err != nil {
				rows.Close()
				return err
			}
			toUpdate = append(toUpdate, r)
		}
		rows.Close()
		log.Printf("store: migrating v16 → v17 — found %d character(s) with web_search in tools", len(toUpdate))
		for _, r := range toUpdate {
			var tools []string
			if err := json.Unmarshal([]byte(r.tools), &tools); err != nil {
				return fmt.Errorf("store: character id=%d has invalid tools JSON: %w", r.id, err)
			}
			for i, t := range tools {
				if t == "web_search" {
					tools[i] = "searxng"
				}
			}
			b, _ := json.Marshal(tools)
			if _, err := s.db.Exec(`UPDATE character SET tools = ? WHERE id = ?`, string(b), r.id); err != nil {
				return err
			}
			log.Printf("store:   character id=%d tools updated to %s", r.id, string(b))
		}
		if _, err := s.db.Exec(`INSERT INTO schema_version (version, timestamp) VALUES (17, ?)`, now()); err != nil {
			return err
		}
		version = 17
		log.Println("store: migration v16 → v17 complete")
	}

	if version < 18 {
		log.Println("store: migrating v17 → v18 (add is_default to character)")
		if _, err := s.db.Exec(`ALTER TABLE character ADD COLUMN is_default INTEGER NOT NULL DEFAULT 0`); err != nil {
			return err
		}
		if _, err := s.db.Exec(`CREATE UNIQUE INDEX idx_character_one_default ON character (is_default) WHERE is_default = 1`); err != nil {
			return err
		}
		if _, err := s.db.Exec(`INSERT INTO schema_version (version, timestamp) VALUES (18, ?)`, now()); err != nil {
			return err
		}
		version = 18
		log.Println("store: migration v17 → v18 complete")
	}

	if version < 19 {
		log.Println("store: migrating v18 → v19 (rename generate_image to generate_image_sdxl in character tools)")
		rows, err := s.db.Query(`SELECT id, tools FROM character WHERE tools IS NOT NULL AND tools LIKE '%generate_image%'`)
		if err != nil {
			return err
		}
		type row struct {
			id    int64
			tools string
		}
		var toUpdate []row
		for rows.Next() {
			var r row
			if err := rows.Scan(&r.id, &r.tools); err != nil {
				rows.Close()
				return err
			}
			toUpdate = append(toUpdate, r)
		}
		rows.Close()
		log.Printf("store: migrating v18 → v19 — found %d character(s) with generate_image in tools", len(toUpdate))
		for _, r := range toUpdate {
			var tools []string
			if err := json.Unmarshal([]byte(r.tools), &tools); err != nil {
				return fmt.Errorf("store: character id=%d has invalid tools JSON: %w", r.id, err)
			}
			for i, t := range tools {
				if t == "generate_image" {
					tools[i] = "generate_image_sdxl"
				}
			}
			b, _ := json.Marshal(tools)
			if _, err := s.db.Exec(`UPDATE character SET tools = ? WHERE id = ?`, string(b), r.id); err != nil {
				return err
			}
			log.Printf("store:   character id=%d tools updated to %s", r.id, string(b))
		}
		if _, err := s.db.Exec(`INSERT INTO schema_version (version, timestamp) VALUES (19, ?)`, now()); err != nil {
			return err
		}
		version = 19
		log.Println("store: migration v18 → v19 complete")
	}

	if version < 20 {
		log.Println("store: migrating v19 → v20 (create conversation_state table)")
		if _, err := s.db.Exec(`
			CREATE TABLE conversation_state (
				id              INTEGER PRIMARY KEY,
				conversation_id INTEGER NOT NULL REFERENCES conversation(id) ON DELETE CASCADE,
				key             TEXT    NOT NULL,
				value           TEXT    NOT NULL,
				updated_at      TEXT    NOT NULL,
				UNIQUE(conversation_id, key)
			);
			CREATE INDEX conversation_state_conversation_id
				ON conversation_state(conversation_id);
		`); err != nil {
			return err
		}
		if _, err := s.db.Exec(`INSERT INTO schema_version (version, timestamp) VALUES (20, ?)`, now()); err != nil {
			return err
		}
		version = 20
		log.Println("store: migration v19 → v20 complete")
	}

	if version < 21 {
		log.Println("store: migrating v20 → v21 (create note table)")
		if _, err := s.db.Exec(`
			CREATE TABLE note (
				id              INTEGER PRIMARY KEY,
				key             TEXT    NOT NULL,
				value           TEXT    NOT NULL DEFAULT '',
				user_id         INTEGER REFERENCES user(id) ON DELETE CASCADE,
				conversation_id INTEGER REFERENCES conversation(id) ON DELETE CASCADE,
				read_only       INTEGER NOT NULL DEFAULT 0,
				created_at      TEXT    NOT NULL,
				updated_at      TEXT    NOT NULL
			);
			CREATE UNIQUE INDEX note_key_scope
				ON note(key, COALESCE(user_id, 0), COALESCE(conversation_id, 0));
			CREATE INDEX note_key_prefix ON note(key);
		`); err != nil {
			return err
		}
		if _, err := s.db.Exec(`INSERT INTO schema_version (version, timestamp) VALUES (21, ?)`, now()); err != nil {
			return err
		}
		version = 21
		log.Println("store: migration v20 → v21 complete")
	}

	if version < 22 {
		log.Println("store: migrating v21 → v22 (create research_job table)")
		if _, err := s.db.Exec(`
			CREATE TABLE research_job (
				id            INTEGER PRIMARY KEY,
				user_id       INTEGER NOT NULL REFERENCES user(id) ON DELETE CASCADE,
				query         TEXT    NOT NULL,
				model         TEXT    NOT NULL,
				status        TEXT    NOT NULL DEFAULT 'pending'
					CHECK (status IN ('pending', 'running', 'done', 'error', 'cancelled')),
				phase         TEXT,
				round         INTEGER NOT NULL DEFAULT 0,
				empty_rounds  INTEGER NOT NULL DEFAULT 0,
				elapsed_ms    INTEGER NOT NULL DEFAULT 0,
				category      TEXT,
				plan          TEXT,
				report        TEXT,
				final_report  TEXT,
				findings      TEXT,
				queries_used  TEXT,
				analyzed_urls TEXT,
				error         TEXT,
				created_at    TEXT    NOT NULL,
				updated_at    TEXT    NOT NULL
			);
			CREATE INDEX idx_research_job_user_id ON research_job(user_id);
			CREATE INDEX idx_research_job_status  ON research_job(status);
		`); err != nil {
			return err
		}
		if _, err := s.db.Exec(`INSERT INTO schema_version (version, timestamp) VALUES (22, ?)`, now()); err != nil {
			return err
		}
		version = 22
		log.Println("store: migration v21 → v22 complete")
	}

	if version < 23 {
		log.Println("store: migrating v22 → v23 (add title to research_job)")
		if _, err := s.db.Exec(`ALTER TABLE research_job ADD COLUMN title TEXT`); err != nil {
			return err
		}
		if _, err := s.db.Exec(`INSERT INTO schema_version (version, timestamp) VALUES (23, ?)`, now()); err != nil {
			return err
		}
		version = 23
		log.Println("store: migration v22 → v23 complete")
	}

	if version < 24 {
		log.Println("store: migrating v23 → v24 (add effort and max_time_seconds to research_job)")
		if _, err := s.db.Exec(`ALTER TABLE research_job ADD COLUMN effort INTEGER NOT NULL DEFAULT 3`); err != nil {
			return err
		}
		if _, err := s.db.Exec(`ALTER TABLE research_job ADD COLUMN max_time_seconds INTEGER NOT NULL DEFAULT 0`); err != nil {
			return err
		}
		if _, err := s.db.Exec(`INSERT INTO schema_version (version, timestamp) VALUES (24, ?)`, now()); err != nil {
			return err
		}
		version = 24
		log.Println("store: migration v23 → v24 complete")
	}

	if version < 25 {
		log.Println("store: migrating v24 → v25 (add mode to research_job)")
		if _, err := s.db.Exec(`ALTER TABLE research_job ADD COLUMN mode TEXT NOT NULL DEFAULT 'research'`); err != nil {
			return err
		}
		if _, err := s.db.Exec(`INSERT INTO schema_version (version, timestamp) VALUES (25, ?)`, now()); err != nil {
			return err
		}
		version = 25
		log.Println("store: migration v24 → v25 complete")
	}

	if version < 26 {
		log.Println("store: migrating v25 → v26 (add force_search to research_job)")
		if _, err := s.db.Exec(`ALTER TABLE research_job ADD COLUMN force_search INTEGER NOT NULL DEFAULT 0`); err != nil {
			return err
		}
		if _, err := s.db.Exec(`INSERT INTO schema_version (version, timestamp) VALUES (26, ?)`, now()); err != nil {
			return err
		}
		version = 26
		log.Println("store: migration v25 → v26 complete")
	}

	log.Printf("store: schema ready at version %d", version)
	return nil
}
