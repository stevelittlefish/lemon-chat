package store

import (
	"database/sql"
	"errors"
	"log"
)

// ResearchJob statuses. A job that is pending or running when the server
// stops is resumed from its last checkpoint on the next startup.
const (
	ResearchStatusPending   = "pending"
	ResearchStatusRunning   = "running"
	ResearchStatusDone      = "done"
	ResearchStatusError     = "error"
	ResearchStatusCancelled = "cancelled"
)

type ResearchJob struct {
	ID          int64   `json:"id"`
	UserID      int64   `json:"user_id"`
	Title       *string `json:"title"`
	Query       string  `json:"query"`
	Model       string  `json:"model"`
	// Mode is "research" (web-search-driven report) or "brainstorm"
	// (ideation-driven design doc, search used only when the model decides it
	// needs to look something up).
	Mode        string  `json:"mode"`
	Status      string  `json:"status"`
	Phase       *string `json:"phase"`
	// Effort is the 1–5 effort level chosen for the job; MaxTimeSeconds is the
	// per-job wall-clock budget (0 means use the configured default).
	Effort         int   `json:"effort"`
	MaxTimeSeconds int   `json:"max_time_seconds"`
	Round          int   `json:"round"`
	EmptyRounds    int   `json:"empty_rounds"`
	ElapsedMS      int64 `json:"elapsed_ms"`
	Category    *string `json:"category"`
	Plan        *string `json:"plan"`
	Report      *string `json:"report"`
	FinalReport *string `json:"final_report"`
	// Findings, QueriesUsed, and AnalyzedURLs are JSON-encoded arrays —
	// checkpoint state for resuming an interrupted job.
	Findings     *string `json:"findings"`
	QueriesUsed  *string `json:"queries_used"`
	AnalyzedURLs *string `json:"analyzed_urls"`
	Error        *string `json:"error"`
	CreatedAt    string  `json:"created_at"`
	UpdatedAt    string  `json:"updated_at"`
}

const researchJobCols = `id, user_id, title, query, model, mode, status, phase, effort, max_time_seconds, round, empty_rounds, elapsed_ms,
	category, plan, report, final_report, findings, queries_used, analyzed_urls, error, created_at, updated_at`

func scanResearchJob(row interface{ Scan(...any) error }) (*ResearchJob, error) {
	var j ResearchJob
	err := row.Scan(&j.ID, &j.UserID, &j.Title, &j.Query, &j.Model, &j.Mode, &j.Status, &j.Phase, &j.Effort, &j.MaxTimeSeconds, &j.Round, &j.EmptyRounds, &j.ElapsedMS,
		&j.Category, &j.Plan, &j.Report, &j.FinalReport, &j.Findings, &j.QueriesUsed, &j.AnalyzedURLs, &j.Error, &j.CreatedAt, &j.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &j, nil
}

func (s *Store) CreateResearchJob(userID int64, title, query, model, mode string, effort, maxTimeSeconds int) (*ResearchJob, error) {
	t := now()
	var titlePtr *string
	if title != "" {
		titlePtr = &title
	}
	res, err := s.db.Exec(
		`INSERT INTO research_job (user_id, title, query, model, mode, status, effort, max_time_seconds, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, 'pending', ?, ?, ?, ?)`,
		userID, titlePtr, query, model, mode, effort, maxTimeSeconds, t, t,
	)
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	return &ResearchJob{ID: id, UserID: userID, Title: titlePtr, Query: query, Model: model, Mode: mode, Status: ResearchStatusPending,
		Effort: effort, MaxTimeSeconds: maxTimeSeconds, CreatedAt: t, UpdatedAt: t}, nil
}

func (s *Store) GetResearchJob(id, userID int64) (*ResearchJob, error) {
	return scanResearchJob(s.db.QueryRow(
		`SELECT `+researchJobCols+` FROM research_job WHERE id = ? AND user_id = ?`, id, userID))
}

func (s *Store) ListResearchJobs(userID int64) ([]ResearchJob, error) {
	rows, err := s.db.Query(
		`SELECT `+researchJobCols+` FROM research_job WHERE user_id = ? ORDER BY created_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []ResearchJob
	for rows.Next() {
		j, err := scanResearchJob(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, *j)
	}
	return items, rows.Err()
}

// ListResumableResearchJobs returns jobs that were in flight when the server
// last stopped. Called once at startup; logs each job it finds.
func (s *Store) ListResumableResearchJobs() ([]ResearchJob, error) {
	rows, err := s.db.Query(
		`SELECT ` + researchJobCols + ` FROM research_job WHERE status IN ('pending', 'running') ORDER BY created_at ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []ResearchJob
	for rows.Next() {
		j, err := scanResearchJob(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, *j)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	log.Printf("store: ListResumableResearchJobs — found %d interrupted job(s)", len(items))
	for _, j := range items {
		log.Printf("store:   research_job id=%d user_id=%d status=%s round=%d query=%q", j.ID, j.UserID, j.Status, j.Round, j.Query)
	}
	return items, nil
}

// UpdateResearchJobPhase records a status/phase transition.
func (s *Store) UpdateResearchJobPhase(id int64, status, phase string) error {
	_, err := s.db.Exec(
		`UPDATE research_job SET status = ?, phase = ?, updated_at = ? WHERE id = ?`,
		status, phase, now(), id)
	return err
}

// CheckpointResearchJob persists the full engine state so the job can be
// resumed from this point if the server stops. Called after planning,
// classification, and at the end of every round.
func (s *Store) CheckpointResearchJob(id int64, round, emptyRounds int, elapsedMS int64, category, plan, report, findings, queriesUsed, analyzedURLs string) error {
	_, err := s.db.Exec(
		`UPDATE research_job SET round = ?, empty_rounds = ?, elapsed_ms = ?, category = ?, plan = ?,
		 report = ?, findings = ?, queries_used = ?, analyzed_urls = ?, updated_at = ? WHERE id = ?`,
		round, emptyRounds, elapsedMS, category, plan, report, findings, queriesUsed, analyzedURLs, now(), id)
	return err
}

// FinishResearchJob marks a job done, errored, or cancelled.
func (s *Store) FinishResearchJob(id int64, status string, finalReport, errMsg *string, elapsedMS int64) error {
	_, err := s.db.Exec(
		`UPDATE research_job SET status = ?, phase = NULL, final_report = ?, error = ?, elapsed_ms = ?, updated_at = ? WHERE id = ?`,
		status, finalReport, errMsg, elapsedMS, now(), id)
	return err
}

func (s *Store) DeleteResearchJob(id, userID int64) error {
	res, err := s.db.Exec(`DELETE FROM research_job WHERE id = ? AND user_id = ?`, id, userID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}
