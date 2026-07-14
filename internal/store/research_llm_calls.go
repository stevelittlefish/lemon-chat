package store

import (
	"database/sql"
	"time"
)

// ResearchLLMCall is one complete or interrupted model request made for a
// research job. Requests and responses are retained without content truncation
// until their parent job is deleted; credentials and HTTP headers are never
// accepted by this API.
type ResearchLLMCall struct {
	ID              int64    `json:"id"`
	ResearchJobID   int64    `json:"research_job_id"`
	Sequence        int64    `json:"sequence"`
	Phase           string   `json:"phase"`
	Operation       string   `json:"operation"`
	Round           int      `json:"round"`
	Attempt         int      `json:"attempt"`
	Model           string   `json:"model"`
	APIBase         string   `json:"api_base"`
	RequestMessages string   `json:"request_messages"`
	Parameters      string   `json:"parameters"`
	StartedAt       string   `json:"started_at"`
	CompletedAt     *string  `json:"completed_at"`
	DurationMS      *int64   `json:"duration_ms"`
	Response        *string  `json:"response"`
	FinishReason    *string  `json:"finish_reason"`
	Usage           *string  `json:"usage"`
	PriceUSD        *float64 `json:"price_usd"`
	HTTPStatus      *int     `json:"http_status"`
	Error           *string  `json:"error"`
	Outcome         string   `json:"outcome"`
	Disposition     string   `json:"disposition"`
}

func (s *Store) BeginResearchLLMCall(jobID int64, phase, operation string, round int, model, apiBase, messages, parameters string) (*ResearchLLMCall, error) {
	startedAt := time.Now().UTC().Format(time.RFC3339Nano)
	res, err := s.db.Exec(`
		INSERT INTO research_llm_call (
			research_job_id, sequence, phase, operation, round, attempt, model, api_base,
			request_messages, parameters, started_at)
		SELECT ?,
			COALESCE((SELECT MAX(sequence) FROM research_llm_call WHERE research_job_id = ?), 0) + 1,
			?, ?, ?,
			COALESCE((SELECT MAX(attempt) FROM research_llm_call WHERE research_job_id = ? AND phase = ? AND operation = ? AND round = ?), 0) + 1,
			?, ?, ?, ?, ?`,
		jobID, jobID, phase, operation, round, jobID, phase, operation, round,
		model, apiBase, messages, parameters, startedAt)
	if err != nil {
		return nil, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, err
	}
	return s.getResearchLLMCall(id)
}

func (s *Store) CompleteResearchLLMCall(id int64, durationMS int64, response, finishReason, usage string, priceUSD *float64, httpStatus int, errMessage string) error {
	completedAt := time.Now().UTC().Format(time.RFC3339Nano)
	var responseValue, finishValue, usageValue, statusValue, errorValue any
	if response != "" {
		responseValue = response
	}
	if finishReason != "" {
		finishValue = finishReason
	}
	if usage != "" {
		usageValue = usage
	}
	if httpStatus > 0 {
		statusValue = httpStatus
	}
	if errMessage != "" {
		errorValue = errMessage
	}
	outcome := "succeeded"
	if errMessage != "" {
		outcome = "failed"
	}
	_, err := s.db.Exec(`UPDATE research_llm_call SET completed_at = ?, duration_ms = ?, response = ?,
		finish_reason = ?, usage = ?, price_usd = ?, http_status = ?, error = ?, outcome = ? WHERE id = ?`,
		completedAt, durationMS, responseValue, finishValue, usageValue, priceUSD, statusValue, errorValue, outcome, id)
	return err
}

func (s *Store) SetResearchLLMCallDisposition(id int64, disposition string) error {
	_, err := s.db.Exec(`UPDATE research_llm_call SET disposition = ? WHERE id = ?`, disposition, id)
	return err
}

func (s *Store) getResearchLLMCall(id int64) (*ResearchLLMCall, error) {
	return scanResearchLLMCall(s.db.QueryRow(`SELECT id, research_job_id, sequence, phase, operation, round,
		attempt, model, api_base, request_messages, parameters, started_at, completed_at, duration_ms,
		response, finish_reason, usage, price_usd, http_status, error, outcome, disposition
		FROM research_llm_call WHERE id = ?`, id))
}

func scanResearchLLMCall(row interface{ Scan(...any) error }) (*ResearchLLMCall, error) {
	var call ResearchLLMCall
	var completedAt, response, finishReason, usage, errMessage sql.NullString
	var duration sql.NullInt64
	var price sql.NullFloat64
	var status sql.NullInt64
	if err := row.Scan(&call.ID, &call.ResearchJobID, &call.Sequence, &call.Phase, &call.Operation,
		&call.Round, &call.Attempt, &call.Model, &call.APIBase, &call.RequestMessages, &call.Parameters,
		&call.StartedAt, &completedAt, &duration, &response, &finishReason, &usage, &price, &status,
		&errMessage, &call.Outcome, &call.Disposition); err != nil {
		return nil, err
	}
	if completedAt.Valid {
		call.CompletedAt = &completedAt.String
	}
	if duration.Valid {
		call.DurationMS = &duration.Int64
	}
	if response.Valid {
		call.Response = &response.String
	}
	if finishReason.Valid {
		call.FinishReason = &finishReason.String
	}
	if usage.Valid {
		call.Usage = &usage.String
	}
	if price.Valid {
		call.PriceUSD = &price.Float64
	}
	if status.Valid {
		value := int(status.Int64)
		call.HTTPStatus = &value
	}
	if errMessage.Valid {
		call.Error = &errMessage.String
	}
	return &call, nil
}

func (s *Store) ListResearchLLMCalls(jobID int64) ([]ResearchLLMCall, error) {
	rows, err := s.db.Query(`SELECT id, research_job_id, sequence, phase, operation, round,
		attempt, model, api_base, request_messages, parameters, started_at, completed_at, duration_ms,
		response, finish_reason, usage, price_usd, http_status, error, outcome, disposition
		FROM research_llm_call WHERE research_job_id = ? ORDER BY sequence`, jobID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var calls []ResearchLLMCall
	for rows.Next() {
		call, err := scanResearchLLMCall(rows)
		if err != nil {
			return nil, err
		}
		calls = append(calls, *call)
	}
	return calls, rows.Err()
}
