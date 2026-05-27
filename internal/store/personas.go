package store

import (
	"database/sql"
	"errors"
)

type Persona struct {
	ID           int64   `json:"id"`
	Name         string  `json:"name"`
	Description  *string `json:"description"`
	SystemPrompt string  `json:"system_prompt"`
	CreatedBy    *int64  `json:"created_by"`
	IsGlobal     bool    `json:"is_global"`
	CreatedAt    string  `json:"created_at"`
	UpdatedAt    string  `json:"updated_at"`
}

func (s *Store) ListPersonas(userID int64) ([]Persona, error) {
	rows, err := s.db.Query(
		`SELECT id, name, description, system_prompt, created_by, is_global, created_at, updated_at
		 FROM persona
		 WHERE is_global = 1 OR created_by = ?
		 ORDER BY name`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var personas []Persona
	for rows.Next() {
		var p Persona
		if err := rows.Scan(&p.ID, &p.Name, &p.Description, &p.SystemPrompt, &p.CreatedBy, &p.IsGlobal, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		personas = append(personas, p)
	}
	return personas, rows.Err()
}

func (s *Store) GetPersona(id int64) (*Persona, error) {
	p := &Persona{}
	err := s.db.QueryRow(
		`SELECT id, name, description, system_prompt, created_by, is_global, created_at, updated_at
		 FROM persona WHERE id = ?`, id,
	).Scan(&p.ID, &p.Name, &p.Description, &p.SystemPrompt, &p.CreatedBy, &p.IsGlobal, &p.CreatedAt, &p.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return p, err
}

func (s *Store) CreatePersona(name string, description *string, systemPrompt string, createdBy int64, isGlobal bool) (*Persona, error) {
	t := now()
	res, err := s.db.Exec(
		`INSERT INTO persona (name, description, system_prompt, created_by, is_global, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		name, description, systemPrompt, createdBy, boolToInt(isGlobal), t, t,
	)
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	return &Persona{
		ID: id, Name: name, Description: description, SystemPrompt: systemPrompt,
		CreatedBy: &createdBy, IsGlobal: isGlobal, CreatedAt: t, UpdatedAt: t,
	}, nil
}

func (s *Store) UpdatePersona(id int64, name string, description *string, systemPrompt string, isGlobal bool) error {
	_, err := s.db.Exec(
		`UPDATE persona SET name = ?, description = ?, system_prompt = ?, is_global = ?, updated_at = ? WHERE id = ?`,
		name, description, systemPrompt, boolToInt(isGlobal), now(), id,
	)
	return err
}

func (s *Store) DeletePersona(id int64) error {
	_, err := s.db.Exec(`DELETE FROM persona WHERE id = ?`, id)
	return err
}
