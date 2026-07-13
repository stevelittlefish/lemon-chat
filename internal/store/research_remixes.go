package store

import (
	"database/sql"
	"errors"
)

// ResearchRemix is the wire/DTO view of an HTML report variant. A remix is a
// non-default research_report that carries an HTML rendering of the job's
// master markdown. It is kept as a distinct type for the current remix UI;
// the underlying storage is the research_report table.
type ResearchRemix struct {
	ID            int64    `json:"id"`
	ResearchJobID int64    `json:"research_job_id"`
	Model         string   `json:"model"`
	Direction     string   `json:"direction"`
	HTML          string   `json:"html,omitempty"`
	PriceUSD      *float64 `json:"price_usd"`
	CreatedAt     string   `json:"created_at"`
}

// CreateResearchRemix stores an HTML rendering as a non-default report. It
// copies the job's current default markdown so the report is self-contained.
func (s *Store) CreateResearchRemix(jobID int64, model, direction, html string, priceUSD *float64) (*ResearchRemix, error) {
	var markdown *string
	if def, err := s.GetDefaultResearchReport(jobID); err == nil {
		markdown = def.Markdown
	} else if !errors.Is(err, ErrNotFound) {
		return nil, err
	}
	rep, err := s.CreateResearchReport(jobID, markdown, html, model, direction, priceUSD, false)
	if err != nil {
		return nil, err
	}
	return &ResearchRemix{ID: rep.ID, ResearchJobID: jobID, Model: model, Direction: direction, HTML: html, PriceUSD: priceUSD, CreatedAt: rep.CreatedAt}, nil
}

func (s *Store) ListResearchRemixes(jobID int64) ([]ResearchRemix, error) {
	rows, err := s.db.Query(`SELECT id, research_job_id, model, direction, price_usd, created_at
		FROM research_report
		WHERE research_job_id = ? AND is_default = 0 AND html IS NOT NULL AND html <> ''
		ORDER BY created_at DESC, id DESC`, jobID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var remixes []ResearchRemix
	for rows.Next() {
		var remix ResearchRemix
		if err := rows.Scan(&remix.ID, &remix.ResearchJobID, &remix.Model, &remix.Direction, &remix.PriceUSD, &remix.CreatedAt); err != nil {
			return nil, err
		}
		remixes = append(remixes, remix)
	}
	return remixes, rows.Err()
}

// ListResearchRemixCounts returns the number of HTML report variants for each
// of a user's research jobs. Jobs without any are omitted from the result.
func (s *Store) ListResearchRemixCounts(userID int64) (map[int64]int, error) {
	rows, err := s.db.Query(`
		SELECT rep.research_job_id, COUNT(*)
		FROM research_report AS rep
		JOIN research_job AS job ON job.id = rep.research_job_id
		WHERE job.user_id = ? AND rep.is_default = 0 AND rep.html IS NOT NULL AND rep.html <> ''
		GROUP BY rep.research_job_id`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	counts := make(map[int64]int)
	for rows.Next() {
		var jobID int64
		var count int
		if err := rows.Scan(&jobID, &count); err != nil {
			return nil, err
		}
		counts[jobID] = count
	}
	return counts, rows.Err()
}

func (s *Store) GetResearchRemix(id, jobID int64) (*ResearchRemix, error) {
	var remix ResearchRemix
	var html sql.NullString
	err := s.db.QueryRow(`SELECT id, research_job_id, model, direction, html, price_usd, created_at
		FROM research_report WHERE id = ? AND research_job_id = ? AND is_default = 0`, id, jobID).
		Scan(&remix.ID, &remix.ResearchJobID, &remix.Model, &remix.Direction, &html, &remix.PriceUSD, &remix.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	remix.HTML = html.String
	return &remix, nil
}
