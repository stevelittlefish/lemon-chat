package store

import (
	"database/sql"
	"errors"
)

type User struct {
	ID             int64
	Username       string
	DisplayName    *string
	AvatarFilename *string
	PasswordHash   *string
	IsAdmin        bool
	CreatedAt      string
}

var ErrNotFound = errors.New("not found")

func (s *Store) UserByID(id int64) (*User, error) {
	u := &User{}
	err := s.db.QueryRow(
		`SELECT id, username, display_name, avatar_filename, password_hash, is_admin, created_at FROM user WHERE id = ?`, id,
	).Scan(&u.ID, &u.Username, &u.DisplayName, &u.AvatarFilename, &u.PasswordHash, &u.IsAdmin, &u.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return u, err
}

func (s *Store) UserByUsername(username string) (*User, error) {
	u := &User{}
	err := s.db.QueryRow(
		`SELECT id, username, display_name, avatar_filename, password_hash, is_admin, created_at FROM user WHERE username = ?`, username,
	).Scan(&u.ID, &u.Username, &u.DisplayName, &u.AvatarFilename, &u.PasswordHash, &u.IsAdmin, &u.CreatedAt)
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

func (s *Store) DeleteUser(id int64) error {
	_, err := s.db.Exec(`DELETE FROM user WHERE id = ?`, id)
	return err
}

func (s *Store) ListUsers() ([]User, error) {
	rows, err := s.db.Query(`SELECT id, username, display_name, avatar_filename, password_hash, is_admin, created_at FROM user ORDER BY username`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var users []User
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.ID, &u.Username, &u.DisplayName, &u.AvatarFilename, &u.PasswordHash, &u.IsAdmin, &u.CreatedAt); err != nil {
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
