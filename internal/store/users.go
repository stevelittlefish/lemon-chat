package store

import (
	"database/sql"
	"errors"
	"log"
)

type User struct {
	ID             int64
	Username       string
	DisplayName    *string
	AvatarFilename *string
	PasswordHash   *string
	IsAdmin        bool
	// CanUseRestricted grants access to models marked restricted in the config
	// (typically ones that cost money). Admins have access regardless.
	CanUseRestricted bool
	CreatedAt        string
}

var ErrNotFound = errors.New("not found")

func (s *Store) UserByID(id int64) (*User, error) {
	u := &User{}
	err := s.db.QueryRow(
		`SELECT id, username, display_name, avatar_filename, password_hash, is_admin, can_use_restricted, created_at FROM user WHERE id = ?`, id,
	).Scan(&u.ID, &u.Username, &u.DisplayName, &u.AvatarFilename, &u.PasswordHash, &u.IsAdmin, &u.CanUseRestricted, &u.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return u, err
}

func (s *Store) UserByUsername(username string) (*User, error) {
	u := &User{}
	err := s.db.QueryRow(
		`SELECT id, username, display_name, avatar_filename, password_hash, is_admin, can_use_restricted, created_at FROM user WHERE username = ?`, username,
	).Scan(&u.ID, &u.Username, &u.DisplayName, &u.AvatarFilename, &u.PasswordHash, &u.IsAdmin, &u.CanUseRestricted, &u.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return u, err
}

func (s *Store) CreateUser(username string, passwordHash *string, isAdmin bool, displayName *string) (*User, error) {
	t := now()
	res, err := s.db.Exec(
		`INSERT INTO user (username, display_name, password_hash, is_admin, created_at) VALUES (?, ?, ?, ?, ?)`,
		username, displayName, passwordHash, boolToInt(isAdmin), t,
	)
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	return &User{ID: id, Username: username, DisplayName: displayName, PasswordHash: passwordHash, IsAdmin: isAdmin, CreatedAt: t}, nil
}

func (s *Store) UpdateUser(id int64, username string, passwordHash *string, isAdmin bool, displayName *string) error {
	_, err := s.db.Exec(
		`UPDATE user SET username = ?, display_name = ?, password_hash = ?, is_admin = ? WHERE id = ?`,
		username, displayName, passwordHash, boolToInt(isAdmin), id,
	)
	return err
}

// SetUserCanUseRestricted grants or revokes access to restricted models.
func (s *Store) SetUserCanUseRestricted(id int64, allowed bool) error {
	_, err := s.db.Exec(`UPDATE user SET can_use_restricted = ? WHERE id = ?`, boolToInt(allowed), id)
	return err
}

// DeleteUser removes a user and everything they own: sessions, conversations
// (with their messages and attachments), and the characters they created.
// Other users' conversations that used one of those characters revert to the
// character's model, as with DeleteCharacter. Completions, notes and research
// jobs go via ON DELETE CASCADE.
//
// It returns the attachment disk paths and avatar filenames (the user's and
// their characters') so the caller can remove the files after commit.
func (s *Store) DeleteUser(id int64) (attachmentPaths, avatarFiles []string, err error) {
	tx, err := s.db.Begin()
	if err != nil {
		return nil, nil, err
	}
	defer tx.Rollback()

	var username string
	var avatar *string
	err = tx.QueryRow(`SELECT username, avatar_filename FROM user WHERE id = ?`, id).Scan(&username, &avatar)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil, ErrNotFound
	}
	if err != nil {
		return nil, nil, err
	}
	if avatar != nil {
		avatarFiles = append(avatarFiles, *avatar)
	}
	log.Printf("store: DeleteUser — deleting user id=%d username=%q", id, username)

	res, err := tx.Exec(`DELETE FROM session WHERE user_id = ?`, id)
	if err != nil {
		return nil, nil, err
	}
	n, _ := res.RowsAffected()
	log.Printf("store: DeleteUser — deleted %d session(s)", n)

	convIDs, err := queryIDs(tx, `SELECT id FROM conversation WHERE user_id = ?`, id)
	if err != nil {
		return nil, nil, err
	}
	attachmentPaths, msgs, err := deleteConversationsTx(tx, convIDs)
	if err != nil {
		return nil, nil, err
	}
	log.Printf("store: DeleteUser — deleted %d conversation(s), %d message(s), %d attachment file(s)", len(convIDs), msgs, len(attachmentPaths))

	rows, err := tx.Query(`SELECT id, name, avatar_filename FROM character WHERE created_by = ?`, id)
	if err != nil {
		return nil, nil, err
	}
	type charRow struct {
		id     int64
		name   string
		avatar *string
	}
	var chars []charRow
	for rows.Next() {
		var c charRow
		if err := rows.Scan(&c.id, &c.name, &c.avatar); err != nil {
			rows.Close()
			return nil, nil, err
		}
		chars = append(chars, c)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}
	for _, c := range chars {
		log.Printf("store:   character id=%d name=%q", c.id, c.name)
		if err := deleteCharacterTx(tx, c.id); err != nil {
			return nil, nil, err
		}
		if c.avatar != nil {
			avatarFiles = append(avatarFiles, *c.avatar)
		}
	}
	log.Printf("store: DeleteUser — deleted %d character(s)", len(chars))

	if _, err := tx.Exec(`DELETE FROM user WHERE id = ?`, id); err != nil {
		return nil, nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, nil, err
	}
	log.Printf("store: DeleteUser — deleted user id=%d successfully", id)
	return attachmentPaths, avatarFiles, nil
}

func queryIDs(tx *sql.Tx, query string, args ...any) ([]int64, error) {
	rows, err := tx.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func (s *Store) ListUsers() ([]User, error) {
	rows, err := s.db.Query(`SELECT id, username, display_name, avatar_filename, password_hash, is_admin, can_use_restricted, created_at FROM user ORDER BY username`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var users []User
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.ID, &u.Username, &u.DisplayName, &u.AvatarFilename, &u.PasswordHash, &u.IsAdmin, &u.CanUseRestricted, &u.CreatedAt); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, rows.Err()
}

func (s *Store) SetUserAvatar(id int64, filename string) error {
	_, err := s.db.Exec(`UPDATE user SET avatar_filename = ? WHERE id = ?`, filename, id)
	return err
}

func (s *Store) ClearUserAvatar(id int64) error {
	_, err := s.db.Exec(`UPDATE user SET avatar_filename = NULL WHERE id = ?`, id)
	return err
}

func (s *Store) UserCount() (int, error) {
	var n int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM user`).Scan(&n)
	return n, err
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
