package store

import (
	"database/sql"
	"errors"
)

// ResearchEvent is one immutable entry in a research job's structured debug
// trace. Data is JSON text whose shape is determined by EventType.
type ResearchEvent struct {
	ID            int64  `json:"id"`
	ResearchJobID int64  `json:"research_job_id"`
	Sequence      int64  `json:"sequence"`
	EventType     string `json:"event_type"`
	Phase         string `json:"phase"`
	Round         int    `json:"round"`
	Message       string `json:"message"`
	Data          string `json:"data"`
	CreatedAt     string `json:"created_at"`
}

// AppendResearchEvent adds the next per-job sequence entry. SQLite serializes
// the INSERT ... SELECT statement, including concurrent extraction events.
func (s *Store) AppendResearchEvent(jobID int64, eventType, phase string, round int, message, data string) (*ResearchEvent, error) {
	if data == "" {
		data = "{}"
	}
	createdAt := now()
	res, err := s.db.Exec(`
		INSERT INTO research_event (research_job_id, sequence, event_type, phase, round, message, data, created_at)
		SELECT ?, COALESCE(MAX(sequence), 0) + 1, ?, ?, ?, ?, ?, ?
		FROM research_event WHERE research_job_id = ?`,
		jobID, eventType, phase, round, message, data, createdAt, jobID)
	if err != nil {
		return nil, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, err
	}
	return scanResearchEvent(s.db.QueryRow(`
		SELECT id, research_job_id, sequence, event_type, phase, round, message, data, created_at
		FROM research_event WHERE id = ?`, id))
}

func scanResearchEvent(row interface{ Scan(...any) error }) (*ResearchEvent, error) {
	var event ResearchEvent
	err := row.Scan(&event.ID, &event.ResearchJobID, &event.Sequence, &event.EventType, &event.Phase,
		&event.Round, &event.Message, &event.Data, &event.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &event, nil
}

func (s *Store) ListResearchEvents(jobID int64) ([]ResearchEvent, error) {
	rows, err := s.db.Query(`
		SELECT id, research_job_id, sequence, event_type, phase, round, message, data, created_at
		FROM research_event WHERE research_job_id = ? ORDER BY sequence`, jobID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var events []ResearchEvent
	for rows.Next() {
		event, err := scanResearchEvent(rows)
		if err != nil {
			return nil, err
		}
		events = append(events, *event)
	}
	return events, rows.Err()
}
