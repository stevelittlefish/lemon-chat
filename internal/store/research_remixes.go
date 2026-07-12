package store

import (
	"database/sql"
	"errors"
)

type ResearchRemix struct {
	ID            int64  `json:"id"`
	ResearchJobID int64  `json:"research_job_id"`
	Model         string `json:"model"`
	Direction     string `json:"direction"`
	HTML          string `json:"html,omitempty"`
	CreatedAt     string `json:"created_at"`
}

func (s *Store) CreateResearchRemix(jobID int64, model, direction, html string) (*ResearchRemix, error) {
	createdAt := now()
	res, err := s.db.Exec(`INSERT INTO research_remix (research_job_id, model, direction, html, created_at) VALUES (?, ?, ?, ?, ?)`,
		jobID, model, direction, html, createdAt)
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	return &ResearchRemix{ID: id, ResearchJobID: jobID, Model: model, Direction: direction, HTML: html, CreatedAt: createdAt}, nil
}

func (s *Store) ListResearchRemixes(jobID int64) ([]ResearchRemix, error) {
	rows, err := s.db.Query(`SELECT id, research_job_id, model, direction, created_at FROM research_remix WHERE research_job_id = ? ORDER BY created_at DESC, id DESC`, jobID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var remixes []ResearchRemix
	for rows.Next() {
		var remix ResearchRemix
		if err := rows.Scan(&remix.ID, &remix.ResearchJobID, &remix.Model, &remix.Direction, &remix.CreatedAt); err != nil {
			return nil, err
		}
		remixes = append(remixes, remix)
	}
	return remixes, rows.Err()
}

func (s *Store) GetResearchRemix(id, jobID int64) (*ResearchRemix, error) {
	var remix ResearchRemix
	err := s.db.QueryRow(`SELECT id, research_job_id, model, direction, html, created_at FROM research_remix WHERE id = ? AND research_job_id = ?`, id, jobID).
		Scan(&remix.ID, &remix.ResearchJobID, &remix.Model, &remix.Direction, &remix.HTML, &remix.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &remix, nil
}
