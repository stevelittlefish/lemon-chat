package store

import (
	"database/sql"
	"errors"
)

// ResearchReport is a report belonging to a research job. Each job has one
// default report — the master markdown document produced by the engine — plus
// any number of additional reports. A report always has a markdown master;
// HTML (today produced by the "remix" flow) is optional.
type ResearchReport struct {
	ID            int64    `json:"id"`
	ResearchJobID int64    `json:"research_job_id"`
	Markdown      *string  `json:"markdown"`
	HTML          string   `json:"html,omitempty"`
	Model         string   `json:"model"`
	Direction     string   `json:"direction"`
	PriceUSD      *float64 `json:"price_usd"`
	IsDefault     bool     `json:"is_default"`
	CreatedAt     string   `json:"created_at"`
	UpdatedAt     string   `json:"updated_at"`
}

const researchReportCols = `id, research_job_id, markdown, html, model, direction, price_usd, is_default, created_at, updated_at`

func scanResearchReport(row interface{ Scan(...any) error }) (*ResearchReport, error) {
	var rep ResearchReport
	var html sql.NullString
	err := row.Scan(&rep.ID, &rep.ResearchJobID, &rep.Markdown, &html, &rep.Model, &rep.Direction, &rep.PriceUSD, &rep.IsDefault, &rep.CreatedAt, &rep.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	rep.HTML = html.String
	return &rep, nil
}

// CreateResearchReport inserts a report for a job.
func (s *Store) CreateResearchReport(jobID int64, markdown *string, html, model, direction string, priceUSD *float64, isDefault bool) (*ResearchReport, error) {
	t := now()
	var htmlVal any
	if html != "" {
		htmlVal = html
	}
	res, err := s.db.Exec(
		`INSERT INTO research_report (research_job_id, markdown, html, model, direction, price_usd, is_default, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		jobID, markdown, htmlVal, model, direction, priceUSD, isDefault, t, t)
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	return &ResearchReport{ID: id, ResearchJobID: jobID, Markdown: markdown, HTML: html, Model: model,
		Direction: direction, PriceUSD: priceUSD, IsDefault: isDefault, CreatedAt: t, UpdatedAt: t}, nil
}

// UpsertDefaultResearchReport records the engine's master markdown document as
// the job's default report, creating it on first completion and updating it if
// the job is re-run. Called when a job finishes successfully.
func (s *Store) UpsertDefaultResearchReport(jobID int64, markdown, model string) error {
	t := now()
	res, err := s.db.Exec(
		`UPDATE research_report SET markdown = ?, model = ?, updated_at = ? WHERE research_job_id = ? AND is_default = 1`,
		markdown, model, t, jobID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n > 0 {
		return nil
	}
	_, err = s.db.Exec(
		`INSERT INTO research_report (research_job_id, markdown, html, model, direction, price_usd, is_default, created_at, updated_at)
		 VALUES (?, ?, NULL, ?, '', NULL, 1, ?, ?)`,
		jobID, markdown, model, t, t)
	return err
}

// SetDefaultResearchReportHTML attaches a designed HTML rendering (and the
// style direction and cost that produced it) to the job's default report.
func (s *Store) SetDefaultResearchReportHTML(jobID int64, html, direction string, priceUSD *float64) error {
	res, err := s.db.Exec(
		`UPDATE research_report SET html = ?, direction = ?, price_usd = ?, updated_at = ? WHERE research_job_id = ? AND is_default = 1`,
		html, direction, priceUSD, now(), jobID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) GetResearchReport(id, jobID int64) (*ResearchReport, error) {
	return scanResearchReport(s.db.QueryRow(
		`SELECT `+researchReportCols+` FROM research_report WHERE id = ? AND research_job_id = ?`, id, jobID))
}

func (s *Store) GetDefaultResearchReport(jobID int64) (*ResearchReport, error) {
	return scanResearchReport(s.db.QueryRow(
		`SELECT `+researchReportCols+` FROM research_report WHERE research_job_id = ? AND is_default = 1`, jobID))
}

// ListResearchReports returns every report for a job, default first.
func (s *Store) ListResearchReports(jobID int64) ([]ResearchReport, error) {
	rows, err := s.db.Query(
		`SELECT `+researchReportCols+` FROM research_report WHERE research_job_id = ?
		 ORDER BY is_default DESC, created_at DESC, id DESC`, jobID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var reports []ResearchReport
	for rows.Next() {
		rep, err := scanResearchReport(rows)
		if err != nil {
			return nil, err
		}
		reports = append(reports, *rep)
	}
	return reports, rows.Err()
}
