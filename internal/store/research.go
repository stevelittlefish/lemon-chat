package store

import (
	"database/sql"
	"errors"
	"log"
	"strings"
)

// ResearchJob statuses. A job that is pending or running when the server
// stops is resumed from its last checkpoint on the next startup.
const (
	ResearchStatusPending        = "pending"
	ResearchStatusRunning        = "running"
	ResearchStatusAwaitingReddit = "awaiting_reddit"
	ResearchStatusDone           = "done"
	ResearchStatusError          = "error"
	ResearchStatusCancelled      = "cancelled"
)

type ResearchJob struct {
	ID     int64   `json:"id"`
	UserID int64   `json:"user_id"`
	Title  *string `json:"title"`
	Query  string  `json:"query"`
	Model  string  `json:"model"`
	// Mode is "research" (web-search-driven report) or "brainstorm"
	// (ideation-driven design doc, search used only when the model decides it
	// needs to look something up).
	Mode string `json:"mode"`
	// ForceSearch (brainstorm mode only) requires at least one web search on the
	// first round instead of letting the model decide whether to search at all.
	ForceSearch bool `json:"force_search"`
	// DeepReport builds the final report section by section from the raw findings
	// (outline → per-section write → glue) instead of in one summarising pass.
	DeepReport bool `json:"deep_report"`
	// AutoHTMLReport requests a designed HTML rendering of the final report once
	// the markdown report is done; HTMLReportDirection carries optional stylistic
	// guidance. Persisted so a resumed job still honours the request.
	AutoHTMLReport      bool    `json:"auto_html_report"`
	HTMLReportDirection *string `json:"html_report_direction"`
	// HTMLReportModel overrides the model used for the HTML report step; nil means
	// reuse the job's own Model.
	HTMLReportModel *string `json:"html_report_model"`
	// WorkerModel overrides the model used for the worker tier (extraction plus the
	// mechanical slug/classify/query-gen/decide calls); nil means reuse the job's
	// own Model.
	WorkerModel *string `json:"worker_model"`
	// PauseRedditImport enables user-assisted capture when a search round finds
	// Reddit threads. It is opt-in for each job.
	PauseRedditImport bool    `json:"pause_reddit_import"`
	Status            string  `json:"status"`
	Phase             *string `json:"phase"`
	// Effort is the 1–5 effort level chosen for the job; MaxTimeSeconds is the
	// per-job wall-clock budget (0 means use the configured default).
	Effort         int      `json:"effort"`
	MaxTimeSeconds int      `json:"max_time_seconds"`
	Round          int      `json:"round"`
	EmptyRounds    int      `json:"empty_rounds"`
	ElapsedMS      int64    `json:"elapsed_ms"`
	PriceUSD       *float64 `json:"price_usd"`
	Category       *string  `json:"category"`
	Slug           *string  `json:"slug"`
	Plan           *string  `json:"plan"`
	Report         *string  `json:"report"`
	FinalReport    *string  `json:"final_report"`
	// Findings, QueriesUsed, and AnalyzedURLs are JSON-encoded arrays —
	// checkpoint state for resuming an interrupted job.
	Findings     *string `json:"findings"`
	QueriesUsed  *string `json:"queries_used"`
	AnalyzedURLs *string `json:"analyzed_urls"`
	// RedditRequestID and PendingRedditRound hold the durable browser handoff
	// and complete pending-round checkpoint. RedditResponse or RedditSkipped is
	// set atomically before the job becomes resumable again.
	RedditRequestID    *string `json:"-"`
	PendingRedditRound *string `json:"-"`
	RedditResponse     *string `json:"-"`
	RedditSkipped      bool    `json:"-"`
	Error              *string `json:"error"`
	CreatedAt          string  `json:"created_at"`
	UpdatedAt          string  `json:"updated_at"`
}

const researchJobCols = `id, user_id, title, query, model, mode, force_search, deep_report, auto_html_report, html_report_direction, html_report_model, worker_model, pause_reddit_import, status, phase, effort, max_time_seconds, round, empty_rounds, elapsed_ms, price_usd,
	category, slug, plan, report, final_report, findings, queries_used, analyzed_urls, reddit_request_id, pending_reddit_round, reddit_response, reddit_skipped, error, created_at, updated_at`

func scanResearchJob(row interface{ Scan(...any) error }) (*ResearchJob, error) {
	var j ResearchJob
	err := row.Scan(&j.ID, &j.UserID, &j.Title, &j.Query, &j.Model, &j.Mode, &j.ForceSearch, &j.DeepReport, &j.AutoHTMLReport, &j.HTMLReportDirection, &j.HTMLReportModel, &j.WorkerModel, &j.PauseRedditImport, &j.Status, &j.Phase, &j.Effort, &j.MaxTimeSeconds, &j.Round, &j.EmptyRounds, &j.ElapsedMS, &j.PriceUSD,
		&j.Category, &j.Slug, &j.Plan, &j.Report, &j.FinalReport, &j.Findings, &j.QueriesUsed, &j.AnalyzedURLs, &j.RedditRequestID, &j.PendingRedditRound, &j.RedditResponse, &j.RedditSkipped, &j.Error, &j.CreatedAt, &j.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &j, nil
}

func (s *Store) CreateResearchJob(userID int64, title, query, model, mode string, forceSearch, deepReport, pauseRedditImport, autoHTMLReport bool, htmlReportDirection, htmlReportModel, workerModel string, effort, maxTimeSeconds int) (*ResearchJob, error) {
	t := now()
	var titlePtr *string
	if title != "" {
		titlePtr = &title
	}
	var directionPtr *string
	if htmlReportDirection != "" {
		directionPtr = &htmlReportDirection
	}
	var htmlModelPtr *string
	if htmlReportModel != "" {
		htmlModelPtr = &htmlReportModel
	}
	var workerModelPtr *string
	if workerModel != "" {
		workerModelPtr = &workerModel
	}
	res, err := s.db.Exec(
		`INSERT INTO research_job (user_id, title, query, model, mode, force_search, deep_report, auto_html_report, html_report_direction, html_report_model, worker_model, pause_reddit_import, status, effort, max_time_seconds, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 'pending', ?, ?, ?, ?)`,
		userID, titlePtr, query, model, mode, forceSearch, deepReport, autoHTMLReport, directionPtr, htmlModelPtr, workerModelPtr, pauseRedditImport, effort, maxTimeSeconds, t, t,
	)
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	return &ResearchJob{ID: id, UserID: userID, Title: titlePtr, Query: query, Model: model, Mode: mode, ForceSearch: forceSearch, DeepReport: deepReport,
		AutoHTMLReport: autoHTMLReport, HTMLReportDirection: directionPtr, HTMLReportModel: htmlModelPtr, WorkerModel: workerModelPtr, PauseRedditImport: pauseRedditImport, Status: ResearchStatusPending,
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
func (s *Store) CheckpointResearchJob(id int64, round, emptyRounds int, elapsedMS int64, priceUSD *float64, category, slug, plan, report, findings, queriesUsed, analyzedURLs string) error {
	_, err := s.db.Exec(
		`UPDATE research_job SET round = ?, empty_rounds = ?, elapsed_ms = ?, price_usd = ?, category = ?, slug = ?, plan = ?,
		 report = ?, findings = ?, queries_used = ?, analyzed_urls = ?, updated_at = ? WHERE id = ?`,
		round, emptyRounds, elapsedMS, priceUSD, category, slug, plan, report, findings, queriesUsed, analyzedURLs, now(), id)
	return err
}

// SetResearchJobAwaitingReddit durably pauses a job after its complete pending
// round has been serialized. The request ID is stored separately for atomic
// matching by import and skip submissions.
func (s *Store) SetResearchJobAwaitingReddit(id int64, requestID, pendingRound string, elapsedMS int64) error {
	res, err := s.db.Exec(`UPDATE research_job SET status = ?, phase = ?, reddit_request_id = ?, pending_reddit_round = ?,
		reddit_response = NULL, reddit_skipped = 0, elapsed_ms = ?, updated_at = ?
		WHERE id = ? AND status IN ('pending', 'running')`,
		ResearchStatusAwaitingReddit, ResearchStatusAwaitingReddit, requestID, pendingRound, elapsedMS, now(), id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

// ResumeResearchRedditImport atomically accepts an import or skip for the
// matching owner and request. transitioned=false makes duplicate submissions
// harmless and prevents two callers from starting runners.
func (s *Store) ResumeResearchRedditImport(id, userID int64, requestID string, response *string, skipped bool) (transitioned bool, err error) {
	res, err := s.db.Exec(`UPDATE research_job SET status = ?, phase = NULL, reddit_response = ?, reddit_skipped = ?, updated_at = ?
		WHERE id = ? AND user_id = ? AND status = ? AND reddit_request_id = ?`,
		ResearchStatusPending, response, skipped, now(), id, userID, ResearchStatusAwaitingReddit, requestID)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	return n == 1, err
}

// ClearResearchRedditState removes a completed handoff after its pending round
// has been merged and checkpointed.
func (s *Store) ClearResearchRedditState(id int64) error {
	_, err := s.db.Exec(`UPDATE research_job SET reddit_request_id = NULL, pending_reddit_round = NULL,
		reddit_response = NULL, reddit_skipped = 0, updated_at = ? WHERE id = ?`, now(), id)
	return err
}

// CompleteResearchRedditRound atomically checkpoints the merged round and
// clears its handoff so it cannot be merged twice after a restart.
func (s *Store) CompleteResearchRedditRound(id int64, round, emptyRounds int, elapsedMS int64, category, slug, plan, report, findings, queriesUsed, analyzedURLs string) error {
	_, err := s.db.Exec(`UPDATE research_job SET round = ?, empty_rounds = ?, elapsed_ms = ?, category = ?, slug = ?, plan = ?,
		report = ?, findings = ?, queries_used = ?, analyzed_urls = ?, reddit_request_id = NULL,
		pending_reddit_round = NULL, reddit_response = NULL, reddit_skipped = 0, updated_at = ? WHERE id = ?`,
		round, emptyRounds, elapsedMS, category, slug, plan, report, findings, queriesUsed, analyzedURLs, now(), id)
	return err
}

// FinishResearchJob marks a job done, errored, or cancelled. When it completes
// with a report, that markdown is also recorded as the job's default report.
func (s *Store) FinishResearchJob(id int64, status string, finalReport, errMsg *string, elapsedMS int64, priceUSD *float64) error {
	if _, err := s.db.Exec(
		`UPDATE research_job SET status = ?, phase = NULL, final_report = ?, error = ?, elapsed_ms = ?, price_usd = ?, updated_at = ? WHERE id = ?`,
		status, finalReport, errMsg, elapsedMS, priceUSD, now(), id); err != nil {
		return err
	}
	if finalReport != nil && strings.TrimSpace(*finalReport) != "" {
		var model string
		if err := s.db.QueryRow(`SELECT model FROM research_job WHERE id = ?`, id).Scan(&model); err != nil {
			return err
		}
		return s.UpsertDefaultResearchReport(id, *finalReport, model)
	}
	return nil
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
