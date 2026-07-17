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

			CREATE TABLE IF NOT EXISTS research_remix (
				id              INTEGER PRIMARY KEY,
				research_job_id INTEGER NOT NULL REFERENCES research_job(id) ON DELETE CASCADE,
				model           TEXT    NOT NULL,
				direction       TEXT    NOT NULL DEFAULT '',
				html            TEXT    NOT NULL,
				created_at      TEXT    NOT NULL
			);
			CREATE INDEX IF NOT EXISTS idx_research_remix_job_id ON research_remix(research_job_id);

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
		log.Println("store: migrating v8 → v10 (add indexes on message, conversation, session)")
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
		log.Println("store: migration v8 → v10 complete")
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

	if version < 27 {
		log.Println("store: migrating v26 → v27 (add deep_report to research_job)")
		if _, err := s.db.Exec(`ALTER TABLE research_job ADD COLUMN deep_report INTEGER NOT NULL DEFAULT 0`); err != nil {
			return err
		}
		if _, err := s.db.Exec(`INSERT INTO schema_version (version, timestamp) VALUES (27, ?)`, now()); err != nil {
			return err
		}
		version = 27
		log.Println("store: migration v26 → v27 complete")
	}

	if version < 28 {
		log.Println("store: migrating v27 → v28 (add background_attachment_id to conversation)")
		if _, err := s.db.Exec(`ALTER TABLE conversation ADD COLUMN background_attachment_id INTEGER`); err != nil {
			return err
		}
		if _, err := s.db.Exec(`INSERT INTO schema_version (version, timestamp) VALUES (28, ?)`, now()); err != nil {
			return err
		}
		version = 28
		log.Println("store: migration v27 → v28 complete")
	}

	if version < 29 {
		log.Println("store: migrating v28 → v29 (add status and error to attachment)")
		if _, err := s.db.Exec(`ALTER TABLE attachment ADD COLUMN status TEXT NOT NULL DEFAULT 'ready'`); err != nil {
			return err
		}
		if _, err := s.db.Exec(`ALTER TABLE attachment ADD COLUMN error TEXT`); err != nil {
			return err
		}
		if _, err := s.db.Exec(`INSERT INTO schema_version (version, timestamp) VALUES (29, ?)`, now()); err != nil {
			return err
		}
		version = 29
		log.Println("store: migration v28 → v29 complete")
	}

	if version < 30 {
		log.Println("store: migrating v29 → v30 (add pause_reddit_import to research_job)")
		if _, err := s.db.Exec(`ALTER TABLE research_job ADD COLUMN pause_reddit_import INTEGER NOT NULL DEFAULT 0`); err != nil {
			return err
		}
		if _, err := s.db.Exec(`INSERT INTO schema_version (version, timestamp) VALUES (30, ?)`, now()); err != nil {
			return err
		}
		version = 30
		log.Println("store: migration v29 → v30 complete")
	}

	if version < 31 {
		log.Println("store: migrating v30 → v31 (add durable Reddit import state)")
		if err := s.migrateResearchRedditState(); err != nil {
			return err
		}
		version = 31
		log.Println("store: migration v30 → v31 complete")
	}

	if version < 32 {
		log.Println("store: migrating v31 → v32 (add research slug)")
		if _, err := s.db.Exec(`ALTER TABLE research_job ADD COLUMN slug TEXT`); err != nil {
			return err
		}
		if _, err := s.db.Exec(`INSERT INTO schema_version (version, timestamp) VALUES (32, ?)`, now()); err != nil {
			return err
		}
		version = 32
		log.Println("store: migration v31 → v32 complete")
	}

	if version < 33 {
		log.Println("store: migrating v32 → v33 (create research remix table)")
		if _, err := s.db.Exec(`
			CREATE TABLE IF NOT EXISTS research_remix (
				id              INTEGER PRIMARY KEY,
				research_job_id INTEGER NOT NULL REFERENCES research_job(id) ON DELETE CASCADE,
				model           TEXT    NOT NULL,
				direction       TEXT    NOT NULL DEFAULT '',
				html            TEXT    NOT NULL,
				created_at      TEXT    NOT NULL
			);
			CREATE INDEX IF NOT EXISTS idx_research_remix_job_id ON research_remix(research_job_id);
		`); err != nil {
			return err
		}
		if _, err := s.db.Exec(`INSERT INTO schema_version (version, timestamp) VALUES (33, ?)`, now()); err != nil {
			return err
		}
		version = 33
		log.Println("store: migration v32 → v33 complete")
	}
	if version < 34 {
		log.Println("store: migrating v33 → v34 (add research pricing)")
		for _, stmt := range []string{`ALTER TABLE research_job ADD COLUMN price_usd REAL`, `ALTER TABLE research_remix ADD COLUMN price_usd REAL`} {
			if _, err := s.db.Exec(stmt); err != nil {
				return err
			}
		}
		if _, err := s.db.Exec(`INSERT INTO schema_version (version, timestamp) VALUES (34, ?)`, now()); err != nil {
			return err
		}
		version = 34
		log.Println("store: migration v33 → v34 complete")
	}

	if version < 35 {
		log.Println("store: migrating v34 → v35 (create research_report table, back-fill reports and remixes)")
		if _, err := s.db.Exec(`
			CREATE TABLE research_report (
				id              INTEGER PRIMARY KEY,
				research_job_id INTEGER NOT NULL REFERENCES research_job(id) ON DELETE CASCADE,
				markdown        TEXT,
				html            TEXT,
				model           TEXT    NOT NULL DEFAULT '',
				direction       TEXT    NOT NULL DEFAULT '',
				price_usd       REAL,
				is_default      INTEGER NOT NULL DEFAULT 0,
				created_at      TEXT    NOT NULL,
				updated_at      TEXT    NOT NULL
			);
			CREATE INDEX idx_research_report_job_id ON research_report(research_job_id);
		`); err != nil {
			return err
		}
		// Every completed job's final_report becomes its default report.
		def, err := s.db.Exec(`
			INSERT INTO research_report (research_job_id, markdown, html, model, direction, price_usd, is_default, created_at, updated_at)
			SELECT id, final_report, NULL, model, '', NULL, 1, updated_at, updated_at
			FROM research_job
			WHERE final_report IS NOT NULL AND TRIM(final_report) <> ''`)
		if err != nil {
			return err
		}
		nDef, _ := def.RowsAffected()
		log.Printf("store: migrating v34 → v35 — back-filled %d default report(s) from final_report", nDef)
		// Each existing remix becomes a non-default report carrying the master
		// markdown it was rendered from plus its HTML.
		rem, err := s.db.Exec(`
			INSERT INTO research_report (research_job_id, markdown, html, model, direction, price_usd, is_default, created_at, updated_at)
			SELECT rr.research_job_id, j.final_report, rr.html, rr.model, rr.direction, rr.price_usd, 0, rr.created_at, rr.created_at
			FROM research_remix AS rr
			JOIN research_job AS j ON j.id = rr.research_job_id`)
		if err != nil {
			return err
		}
		nRem, _ := rem.RowsAffected()
		log.Printf("store: migrating v34 → v35 — back-filled %d report(s) from research_remix", nRem)
		if _, err := s.db.Exec(`INSERT INTO schema_version (version, timestamp) VALUES (35, ?)`, now()); err != nil {
			return err
		}
		version = 35
		log.Println("store: migration v34 → v35 complete")
	}

	if version < 36 {
		log.Println("store: migrating v35 → v36 (add auto HTML report options to research_job)")
		if _, err := s.db.Exec(`ALTER TABLE research_job ADD COLUMN auto_html_report INTEGER NOT NULL DEFAULT 0`); err != nil {
			return err
		}
		if _, err := s.db.Exec(`ALTER TABLE research_job ADD COLUMN html_report_direction TEXT`); err != nil {
			return err
		}
		if _, err := s.db.Exec(`INSERT INTO schema_version (version, timestamp) VALUES (36, ?)`, now()); err != nil {
			return err
		}
		version = 36
		log.Println("store: migration v35 → v36 complete")
	}

	if version < 37 {
		log.Println("store: migrating v36 → v37 (add HTML report model override to research_job)")
		if _, err := s.db.Exec(`ALTER TABLE research_job ADD COLUMN html_report_model TEXT`); err != nil {
			return err
		}
		if _, err := s.db.Exec(`INSERT INTO schema_version (version, timestamp) VALUES (37, ?)`, now()); err != nil {
			return err
		}
		version = 37
		log.Println("store: migration v36 → v37 complete")
	}

	if version < 38 {
		log.Println("store: migrating v37 → v38 (add worker model override to research_job)")
		if _, err := s.db.Exec(`ALTER TABLE research_job ADD COLUMN worker_model TEXT`); err != nil {
			return err
		}
		if _, err := s.db.Exec(`INSERT INTO schema_version (version, timestamp) VALUES (38, ?)`, now()); err != nil {
			return err
		}
		version = 38
		log.Println("store: migration v37 → v38 complete")
	}

	if version < 39 {
		log.Println("store: migrating v38 → v39 (create model_price table)")
		if _, err := s.db.Exec(`
			CREATE TABLE model_price (
				model_id       TEXT PRIMARY KEY,
				prompt_usd     REAL NOT NULL,
				completion_usd REAL NOT NULL,
				updated_at     TEXT NOT NULL
			);
		`); err != nil {
			return err
		}
		if _, err := s.db.Exec(`INSERT INTO schema_version (version, timestamp) VALUES (39, ?)`, now()); err != nil {
			return err
		}
		version = 39
		log.Println("store: migration v38 → v39 complete")
	}

	if version < 40 {
		log.Println("store: migrating v39 → v40 (research_report.markdown_direction)")
		if _, err := s.db.Exec(`ALTER TABLE research_report ADD COLUMN markdown_direction TEXT NOT NULL DEFAULT ''`); err != nil {
			return err
		}
		if _, err := s.db.Exec(`INSERT INTO schema_version (version, timestamp) VALUES (40, ?)`, now()); err != nil {
			return err
		}
		version = 40
		log.Println("store: migration v39 → v40 complete")
	}

	if version < 41 {
		log.Println("store: migrating v40 → v41 (oauth_token single-row table)")
		if _, err := s.db.Exec(`
			CREATE TABLE oauth_token (
				id            INTEGER PRIMARY KEY CHECK (id = 1),
				provider      TEXT    NOT NULL DEFAULT 'openai',
				access_token  TEXT    NOT NULL,
				refresh_token TEXT    NOT NULL,
				id_token      TEXT    NOT NULL DEFAULT '',
				account_id    TEXT    NOT NULL DEFAULT '',
				expiry        TEXT    NOT NULL,
				updated_at    TEXT    NOT NULL
			)`); err != nil {
			return err
		}
		if _, err := s.db.Exec(`INSERT INTO schema_version (version, timestamp) VALUES (41, ?)`, now()); err != nil {
			return err
		}
		version = 41
		log.Println("store: migration v40 → v41 complete")
	}

	if version < 42 {
		log.Println("store: migrating v41 → v42 (message.cached_tokens)")
		if _, err := s.db.Exec(`ALTER TABLE message ADD COLUMN cached_tokens INTEGER`); err != nil {
			return err
		}
		if _, err := s.db.Exec(`INSERT INTO schema_version (version, timestamp) VALUES (42, ?)`, now()); err != nil {
			return err
		}
		version = 42
		log.Println("store: migration v41 → v42 complete")
	}

	log.Printf("store: schema ready at version %d", version)
	return nil
}

// migrateResearchRedditState rebuilds research_job because SQLite cannot alter
// its status CHECK constraint in place. The copy and table swap are one
// transaction, so a failure leaves the complete v30 table untouched.
func (s *Store) migrateResearchRedditState() (err error) {
	if _, err = s.db.Exec(`PRAGMA foreign_keys = OFF`); err != nil {
		return fmt.Errorf("disable foreign keys for research_job rebuild: %w", err)
	}
	defer func() {
		if _, pragmaErr := s.db.Exec(`PRAGMA foreign_keys = ON`); err == nil && pragmaErr != nil {
			err = fmt.Errorf("restore foreign keys after research_job rebuild: %w", pragmaErr)
		}
	}()

	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err = tx.Exec(`
		CREATE TABLE research_job_v31 (
			id                    INTEGER PRIMARY KEY,
			user_id               INTEGER NOT NULL REFERENCES user(id) ON DELETE CASCADE,
			query                 TEXT    NOT NULL,
			model                 TEXT    NOT NULL,
			status                TEXT    NOT NULL DEFAULT 'pending'
				CHECK (status IN ('pending', 'running', 'awaiting_reddit', 'done', 'error', 'cancelled')),
			phase                 TEXT,
			round                 INTEGER NOT NULL DEFAULT 0,
			empty_rounds          INTEGER NOT NULL DEFAULT 0,
			elapsed_ms            INTEGER NOT NULL DEFAULT 0,
			category              TEXT,
			plan                  TEXT,
			report                TEXT,
			final_report          TEXT,
			findings              TEXT,
			queries_used          TEXT,
			analyzed_urls         TEXT,
			error                 TEXT,
			created_at            TEXT    NOT NULL,
			updated_at            TEXT    NOT NULL,
			title                 TEXT,
			effort                INTEGER NOT NULL DEFAULT 3,
			max_time_seconds      INTEGER NOT NULL DEFAULT 0,
			mode                  TEXT    NOT NULL DEFAULT 'research',
			force_search          INTEGER NOT NULL DEFAULT 0,
			deep_report           INTEGER NOT NULL DEFAULT 0,
			pause_reddit_import   INTEGER NOT NULL DEFAULT 0,
			reddit_request_id     TEXT,
			pending_reddit_round  TEXT,
			reddit_response       TEXT,
			reddit_skipped        INTEGER NOT NULL DEFAULT 0
		);

		INSERT INTO research_job_v31 (
			id, user_id, query, model, status, phase, round, empty_rounds, elapsed_ms,
			category, plan, report, final_report, findings, queries_used, analyzed_urls,
			error, created_at, updated_at, title, effort, max_time_seconds, mode,
			force_search, deep_report, pause_reddit_import
		)
		SELECT
			id, user_id, query, model, status, phase, round, empty_rounds, elapsed_ms,
			category, plan, report, final_report, findings, queries_used, analyzed_urls,
			error, created_at, updated_at, title, effort, max_time_seconds, mode,
			force_search, deep_report, pause_reddit_import
		FROM research_job;

		DROP TABLE research_job;
		ALTER TABLE research_job_v31 RENAME TO research_job;
		CREATE INDEX idx_research_job_user_id ON research_job(user_id);
		CREATE INDEX idx_research_job_status ON research_job(status);
	`); err != nil {
		return fmt.Errorf("rebuild research_job: %w", err)
	}
	if _, err = tx.Exec(`INSERT INTO schema_version (version, timestamp) VALUES (31, ?)`, now()); err != nil {
		return err
	}
	var violations int
	rows, err := tx.Query(`PRAGMA foreign_key_check`)
	if err != nil {
		return fmt.Errorf("check foreign keys after research_job rebuild: %w", err)
	}
	for rows.Next() {
		violations++
	}
	if closeErr := rows.Close(); closeErr != nil {
		return closeErr
	}
	if violations != 0 {
		return fmt.Errorf("research_job migration produced %d foreign key violation(s)", violations)
	}
	return tx.Commit()
}
