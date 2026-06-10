package store

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
)

var noteKeyRegex = regexp.MustCompile(`^(g|u|c)\.([a-z0-9_-]+\.)*[a-z0-9_-]+$`)

func ValidateNoteKey(key string) error {
	if !noteKeyRegex.MatchString(key) {
		return fmt.Errorf("invalid key %q: must start with g., u., or c. followed by lowercase letters, digits, underscores, hyphens, and dots (no consecutive dots, no trailing dot)", key)
	}
	return nil
}

type Note struct {
	ID        int64  `json:"id"`
	Key       string `json:"key"`
	Value     string `json:"value"`
	ReadOnly  bool   `json:"read_only"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

type NoteRow struct {
	ID        int64  `json:"id"`
	Key       string `json:"key"`
	Excerpt   string `json:"excerpt"`
	ReadOnly  bool   `json:"read_only"`
	UpdatedAt string `json:"updated_at"`
}

// noteContextFKs derives user_id / conversation_id based on the key's scope prefix.
func noteContextFKs(key string, userID, convID int64) (*int64, *int64) {
	switch key[0] {
	case 'u':
		return &userID, nil
	case 'c':
		return nil, &convID
	default: // 'g'
		return nil, nil
	}
}

func scanNoteRow(rows *sql.Rows) (NoteRow, error) {
	var r NoteRow
	var readOnly int
	if err := rows.Scan(&r.ID, &r.Key, &r.Excerpt, &readOnly, &r.UpdatedAt); err != nil {
		return NoteRow{}, err
	}
	r.ReadOnly = readOnly != 0
	return r, nil
}

// SaveNote creates or replaces a note. Fails if the note exists and is read-only.
func (s *Store) SaveNote(key, value string, userID, convID int64) error {
	if err := ValidateNoteKey(key); err != nil {
		return err
	}
	if key[0] == 'c' && convID == 0 {
		return fmt.Errorf("no active conversation: c. scope requires an active conversation")
	}
	value = strings.TrimSpace(value)
	uid, cid := noteContextFKs(key, userID, convID)
	ts := now()

	var existingID int64
	var existingReadOnly int
	err := s.db.QueryRow(`
		SELECT id, read_only FROM note
		WHERE key = ?
		  AND COALESCE(user_id, 0)         = COALESCE(?, 0)
		  AND COALESCE(conversation_id, 0) = COALESCE(?, 0)
	`, key, uid, cid).Scan(&existingID, &existingReadOnly)

	if err == nil {
		if existingReadOnly != 0 {
			return fmt.Errorf("note is read-only")
		}
		_, err = s.db.Exec(`UPDATE note SET value = ?, updated_at = ? WHERE id = ?`, value, ts, existingID)
		return err
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return err
	}

	_, err = s.db.Exec(`
		INSERT INTO note (key, value, user_id, conversation_id, read_only, created_at, updated_at)
		VALUES (?, ?, ?, ?, 0, ?, ?)
	`, key, value, uid, cid, ts, ts)
	return err
}

// LoadNote fetches a single note by exact key. Returns ErrNotFound if absent or not visible.
func (s *Store) LoadNote(key string, userID, convID int64) (Note, error) {
	if err := ValidateNoteKey(key); err != nil {
		return Note{}, err
	}
	if key[0] == 'c' && convID == 0 {
		return Note{}, fmt.Errorf("no active conversation: c. scope requires an active conversation")
	}
	uid, cid := noteContextFKs(key, userID, convID)

	var n Note
	var readOnly int
	err := s.db.QueryRow(`
		SELECT id, key, value, read_only, created_at, updated_at
		FROM note
		WHERE key = ?
		  AND COALESCE(user_id, 0)         = COALESCE(?, 0)
		  AND COALESCE(conversation_id, 0) = COALESCE(?, 0)
	`, key, uid, cid).Scan(&n.ID, &n.Key, &n.Value, &readOnly, &n.CreatedAt, &n.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return Note{}, ErrNotFound
	}
	if err != nil {
		return Note{}, err
	}
	n.ReadOnly = readOnly != 0
	return n, nil
}

// ListNotes returns matching notes visible to the caller. prefix may be:
//   - empty: all accessible notes
//   - a scope letter alone ("g", "g.", "u", "u.", "c", "c."): all notes in that scope
//   - a scoped path like "g.eldoria": global notes at that key or under it
//   - a bare term like "eldoria": prefix match under each scope (g.eldoria*, u.eldoria*, c.eldoria*)
func (s *Store) ListNotes(prefix string, userID, convID int64) (string, error) {
	var whereClauses []string
	var args []any

	addGlobal := func(exactKey, likeKey string) {
		if exactKey == "" {
			whereClauses = append(whereClauses, "(user_id IS NULL AND conversation_id IS NULL)")
		} else {
			whereClauses = append(whereClauses, "(user_id IS NULL AND conversation_id IS NULL AND (key = ? OR key LIKE ?))")
			args = append(args, exactKey, likeKey)
		}
	}
	addUser := func(exactKey, likeKey string) {
		if exactKey == "" {
			whereClauses = append(whereClauses, "(user_id = ? AND conversation_id IS NULL)")
			args = append(args, userID)
		} else {
			whereClauses = append(whereClauses, "(user_id = ? AND conversation_id IS NULL AND (key = ? OR key LIKE ?))")
			args = append(args, userID, exactKey, likeKey)
		}
	}
	addConv := func(exactKey, likeKey string) {
		if convID == 0 {
			return
		}
		if exactKey == "" {
			whereClauses = append(whereClauses, "(conversation_id = ? AND user_id IS NULL)")
			args = append(args, convID)
		} else {
			whereClauses = append(whereClauses, "(conversation_id = ? AND user_id IS NULL AND (key = ? OR key LIKE ?))")
			args = append(args, convID, exactKey, likeKey)
		}
	}

	if prefix == "" {
		addGlobal("", "")
		addUser("", "")
		addConv("", "")
	} else {
		scopeLetter := prefix[0] == 'g' || prefix[0] == 'u' || prefix[0] == 'c'
		scopeOnly := scopeLetter && (len(prefix) == 1 || (len(prefix) == 2 && prefix[1] == '.'))
		hasScope := scopeLetter && !scopeOnly && len(prefix) >= 3 && prefix[1] == '.'

		if scopeOnly {
			switch prefix[0] {
			case 'g':
				addGlobal("", "")
			case 'u':
				addUser("", "")
			case 'c':
				addConv("", "")
			}
		} else if hasScope {
			exact := prefix
			like := prefix + "%"
			switch prefix[0] {
			case 'g':
				addGlobal(exact, like)
			case 'u':
				addUser(exact, like)
			case 'c':
				addConv(exact, like)
			}
		} else {
			// Bare term: prefix match under each scope (e.g. "foo" finds g.foo*, u.foo*, c.foo*).
			addGlobal("g."+prefix, "g."+prefix+"%")
			addUser("u."+prefix, "u."+prefix+"%")
			addConv("c."+prefix, "c."+prefix+"%")
		}
	}

	if len(whereClauses) == 0 {
		return "[]", nil
	}

	const limit = 50

	query := fmt.Sprintf(`
		SELECT id, key, SUBSTR(value, 1, 120), read_only, updated_at
		FROM note
		WHERE %s
		ORDER BY key
		LIMIT %d
	`, strings.Join(whereClauses, " OR "), limit+1)

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return "", err
	}
	defer rows.Close()

	var result []NoteRow
	for rows.Next() {
		r, err := scanNoteRow(rows)
		if err != nil {
			return "", err
		}
		result = append(result, r)
	}
	if err := rows.Err(); err != nil {
		return "", err
	}

	truncated := len(result) > limit
	if truncated {
		result = result[:limit]
	}

	type listResponse struct {
		Notes   []NoteRow `json:"notes"`
		Message string    `json:"message,omitempty"`
	}
	resp := listResponse{Notes: result}
	if result == nil {
		resp.Notes = []NoteRow{}
	}
	if truncated {
		resp.Message = fmt.Sprintf("Showing first %d results. Use a more specific prefix to narrow the list.", limit)
	}
	b, err := json.Marshal(resp)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// DeleteNote removes a note. Fails if not found, not visible, or read-only.
func (s *Store) DeleteNote(key string, userID, convID int64) error {
	if err := ValidateNoteKey(key); err != nil {
		return err
	}
	if key[0] == 'c' && convID == 0 {
		return fmt.Errorf("no active conversation: c. scope requires an active conversation")
	}
	uid, cid := noteContextFKs(key, userID, convID)

	var existingID int64
	var existingReadOnly int
	err := s.db.QueryRow(`
		SELECT id, read_only FROM note
		WHERE key = ?
		  AND COALESCE(user_id, 0)         = COALESCE(?, 0)
		  AND COALESCE(conversation_id, 0) = COALESCE(?, 0)
	`, key, uid, cid).Scan(&existingID, &existingReadOnly)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	if existingReadOnly != 0 {
		return fmt.Errorf("note is read-only")
	}

	_, err = s.db.Exec(`DELETE FROM note WHERE id = ?`, existingID)
	return err
}

// AppendNote appends text to an existing note, creating it if absent.
// Returns "created" or "appended".
func (s *Store) AppendNote(key, text string, userID, convID int64) (string, error) {
	if err := ValidateNoteKey(key); err != nil {
		return "", err
	}
	if key[0] == 'c' && convID == 0 {
		return "", fmt.Errorf("no active conversation: c. scope requires an active conversation")
	}
	text = strings.TrimSpace(text)
	uid, cid := noteContextFKs(key, userID, convID)
	ts := now()

	var existingID int64
	var existingValue string
	var existingReadOnly int
	err := s.db.QueryRow(`
		SELECT id, value, read_only FROM note
		WHERE key = ?
		  AND COALESCE(user_id, 0)         = COALESCE(?, 0)
		  AND COALESCE(conversation_id, 0) = COALESCE(?, 0)
	`, key, uid, cid).Scan(&existingID, &existingValue, &existingReadOnly)

	if errors.Is(err, sql.ErrNoRows) {
		_, err = s.db.Exec(`
			INSERT INTO note (key, value, user_id, conversation_id, read_only, created_at, updated_at)
			VALUES (?, ?, ?, ?, 0, ?, ?)
		`, key, text, uid, cid, ts, ts)
		if err != nil {
			return "", err
		}
		return "created", nil
	}
	if err != nil {
		return "", err
	}
	if existingReadOnly != 0 {
		return "", fmt.Errorf("note is read-only")
	}

	newValue := existingValue + "\n\n" + text
	_, err = s.db.Exec(`UPDATE note SET value = ?, updated_at = ? WHERE id = ?`, newValue, ts, existingID)
	if err != nil {
		return "", err
	}
	return "appended", nil
}

// ListUserVisibleNotes lists all notes visible to a user for the settings UI:
// global notes, the user's own notes, and notes from the user's conversations.
func (s *Store) ListUserVisibleNotes(prefix string, userID int64) ([]NoteRow, error) {
	var whereClauses []string
	var args []any

	buildFilter := func(exactKey, likeKey string) string {
		if exactKey == "" {
			return ""
		}
		return "(key = ? OR key LIKE ?)"
	}
	appendFilter := func(clause, exactKey, likeKey string) {
		if exactKey == "" {
			whereClauses = append(whereClauses, clause)
		} else {
			whereClauses = append(whereClauses, fmt.Sprintf("(%s AND %s)", clause, buildFilter(exactKey, likeKey)))
			args = append(args, exactKey, likeKey)
		}
	}

	globalClause := "(user_id IS NULL AND conversation_id IS NULL)"
	userClause := fmt.Sprintf("(user_id = %d AND conversation_id IS NULL)", userID)
	convClause := fmt.Sprintf("(user_id IS NULL AND conversation_id IN (SELECT id FROM conversation WHERE user_id = %d))", userID)

	if prefix == "" {
		whereClauses = append(whereClauses, globalClause)
		whereClauses = append(whereClauses, userClause)
		whereClauses = append(whereClauses, convClause)
	} else {
		hasScope := len(prefix) >= 3 && prefix[1] == '.' && (prefix[0] == 'g' || prefix[0] == 'u' || prefix[0] == 'c')
		if hasScope {
			exact := prefix
			like := prefix + ".%"
			switch prefix[0] {
			case 'g':
				appendFilter(globalClause, exact, like)
			case 'u':
				appendFilter(userClause, exact, like)
			case 'c':
				appendFilter(convClause, exact, like)
			}
		} else {
			appendFilter(globalClause, "g."+prefix, "g."+prefix+".%")
			appendFilter(userClause, "u."+prefix, "u."+prefix+".%")
			appendFilter(convClause, "c."+prefix, "c."+prefix+".%")
		}
	}

	if len(whereClauses) == 0 {
		return []NoteRow{}, nil
	}

	query := fmt.Sprintf(`
		SELECT id, key, SUBSTR(value, 1, 120), read_only, updated_at
		FROM note
		WHERE %s
		ORDER BY key
	`, strings.Join(whereClauses, " OR "))

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []NoteRow
	for rows.Next() {
		r, err := scanNoteRow(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, r)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if result == nil {
		result = []NoteRow{}
	}
	return result, nil
}

// GetNoteByID fetches a note by its DB id (for the settings REST API).
func (s *Store) GetNoteByID(id int64) (Note, error) {
	var n Note
	var readOnly int
	err := s.db.QueryRow(`
		SELECT id, key, value, read_only, created_at, updated_at FROM note WHERE id = ?
	`, id).Scan(&n.ID, &n.Key, &n.Value, &readOnly, &n.CreatedAt, &n.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return Note{}, ErrNotFound
	}
	if err != nil {
		return Note{}, err
	}
	n.ReadOnly = readOnly != 0
	return n, nil
}

// UpsertNoteAdmin creates or replaces a note from the settings UI.
// readOnly nil means preserve the existing value (or default to false on create).
// Does not enforce the read-only restriction (the handler has already checked permissions).
func (s *Store) UpsertNoteAdmin(key, value string, readOnly *bool, userID *int64, convID *int64) (Note, error) {
	if err := ValidateNoteKey(key); err != nil {
		return Note{}, err
	}
	value = strings.TrimSpace(value)
	ts := now()

	var existingID int64
	var existingReadOnly int
	err := s.db.QueryRow(`
		SELECT id, read_only FROM note
		WHERE key = ?
		  AND COALESCE(user_id, 0)         = COALESCE(?, 0)
		  AND COALESCE(conversation_id, 0) = COALESCE(?, 0)
	`, key, userID, convID).Scan(&existingID, &existingReadOnly)

	if err == nil {
		// Update existing — preserve read_only if not specified.
		ro := existingReadOnly
		if readOnly != nil {
			if *readOnly {
				ro = 1
			} else {
				ro = 0
			}
		}
		_, err = s.db.Exec(`UPDATE note SET value = ?, read_only = ?, updated_at = ? WHERE id = ?`, value, ro, ts, existingID)
		if err != nil {
			return Note{}, err
		}
	} else if errors.Is(err, sql.ErrNoRows) {
		ro := 0
		if readOnly != nil && *readOnly {
			ro = 1
		}
		result, err := s.db.Exec(`
			INSERT INTO note (key, value, user_id, conversation_id, read_only, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?)
		`, key, value, userID, convID, ro, ts, ts)
		if err != nil {
			return Note{}, err
		}
		existingID, _ = result.LastInsertId()
	} else {
		return Note{}, err
	}

	return s.GetNoteByID(existingID)
}

// SetNoteReadOnly toggles the read_only flag on a note.
func (s *Store) SetNoteReadOnly(id int64, readOnly bool) error {
	ro := 0
	if readOnly {
		ro = 1
	}
	result, err := s.db.Exec(`UPDATE note SET read_only = ?, updated_at = ? WHERE id = ?`, ro, now(), id)
	if err != nil {
		return err
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// DeleteNoteByID removes a note by DB id (for the settings REST API).
func (s *Store) DeleteNoteByID(id int64) error {
	result, err := s.db.Exec(`DELETE FROM note WHERE id = ?`, id)
	if err != nil {
		return err
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}
