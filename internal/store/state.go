package store

import (
	"database/sql"
	"fmt"
	"strconv"
	"strings"
)

func (s *Store) SetState(conversationID int64, key, value string) error {
	_, err := s.db.Exec(
		`INSERT INTO conversation_state (conversation_id, key, value, updated_at)
		 VALUES (?, ?, ?, ?)
		 ON CONFLICT(conversation_id, key) DO UPDATE SET value = excluded.value, updated_at = excluded.updated_at`,
		conversationID, key, value, now(),
	)
	return err
}

func (s *Store) ModifyState(conversationID int64, key string, delta float64) (string, error) {
	var current string
	err := s.db.QueryRow(
		`SELECT value FROM conversation_state WHERE conversation_id = ? AND key = ?`,
		conversationID, key,
	).Scan(&current)
	if err == sql.ErrNoRows {
		return "", fmt.Errorf("key %q is not set — use state_set to create it first", key)
	}
	if err != nil {
		return "", err
	}

	f, parseErr := strconv.ParseFloat(strings.TrimSpace(current), 64)
	if parseErr != nil {
		return "", fmt.Errorf("value %q for key %q is not numeric and cannot be modified with a delta", current, key)
	}

	newVal := f + delta
	var newStr string
	if newVal == float64(int64(newVal)) {
		newStr = strconv.FormatInt(int64(newVal), 10)
	} else {
		newStr = strconv.FormatFloat(newVal, 'f', -1, 64)
	}

	if _, err := s.db.Exec(
		`UPDATE conversation_state SET value = ?, updated_at = ? WHERE conversation_id = ? AND key = ?`,
		newStr, now(), conversationID, key,
	); err != nil {
		return "", err
	}
	return newStr, nil
}

func (s *Store) UnsetState(conversationID int64, key string) error {
	result, err := s.db.Exec(
		`DELETE FROM conversation_state WHERE conversation_id = ? AND key = ?`,
		conversationID, key,
	)
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return fmt.Errorf("key %q is not set", key)
	}
	return nil
}

func (s *Store) ClearState(conversationID int64) (int64, error) {
	result, err := s.db.Exec(
		`DELETE FROM conversation_state WHERE conversation_id = ?`,
		conversationID,
	)
	if err != nil {
		return 0, err
	}
	n, err := result.RowsAffected()
	return n, err
}

func (s *Store) ListState(conversationID int64) (string, error) {
	rows, err := s.db.Query(
		`SELECT key, value FROM conversation_state WHERE conversation_id = ? ORDER BY id`,
		conversationID,
	)
	if err != nil {
		return "", err
	}
	defer rows.Close()

	var sb strings.Builder
	n := 0
	for rows.Next() {
		var key, value string
		if err := rows.Scan(&key, &value); err != nil {
			return "", err
		}
		fmt.Fprintf(&sb, "%s: %s\n", key, value)
		n++
	}
	if err := rows.Err(); err != nil {
		return "", err
	}
	if n == 0 {
		return "No state stored.", nil
	}
	return strings.TrimRight(sb.String(), "\n"), nil
}
