package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/stevelittlefish/lemon-chat/internal/research"
	"github.com/stevelittlefish/lemon-chat/internal/store"
)

// researchRun tracks one in-flight research job: its cancel function and the
// SSE subscribers receiving progress events.
type researchRun struct {
	cancel context.CancelFunc

	mu              sync.Mutex
	cancelRequested bool
	finished        bool
	subs            map[chan []byte]struct{}
	last            []byte // most recent event, replayed to new subscribers
}

func (run *researchRun) requestCancel() {
	run.mu.Lock()
	run.cancelRequested = true
	run.mu.Unlock()
	run.cancel()
}

func (run *researchRun) wasCancelRequested() bool {
	run.mu.Lock()
	defer run.mu.Unlock()
	return run.cancelRequested
}

func (run *researchRun) broadcast(data []byte) {
	run.mu.Lock()
	defer run.mu.Unlock()
	run.last = data
	for ch := range run.subs {
		select {
		case ch <- data:
		default: // slow subscriber — drop the event rather than block the engine
		}
	}
}

// finish broadcasts the terminal event and closes all subscriber channels.
func (run *researchRun) finish(data []byte) {
	run.mu.Lock()
	defer run.mu.Unlock()
	run.finished = true
	run.last = data
	for ch := range run.subs {
		select {
		case ch <- data:
		default:
		}
		close(ch)
	}
	run.subs = map[chan []byte]struct{}{}
}

func (run *researchRun) subscribe() chan []byte {
	run.mu.Lock()
	defer run.mu.Unlock()
	ch := make(chan []byte, 64)
	if run.finished {
		// finish() already ran — return a pre-closed channel so the SSE handler
		// drains the last event and sends [DONE] instead of blocking forever.
		if run.last != nil {
			ch <- run.last
		}
		close(ch)
		return ch
	}
	if run.last != nil {
		ch <- run.last
	}
	run.subs[ch] = struct{}{}
	return ch
}

func (run *researchRun) unsubscribe(ch chan []byte) {
	run.mu.Lock()
	defer run.mu.Unlock()
	if _, ok := run.subs[ch]; ok {
		delete(run.subs, ch)
		close(ch)
	}
}

type researchManager struct {
	mu   sync.Mutex
	runs map[int64]*researchRun
}

func newResearchManager() *researchManager {
	return &researchManager{runs: map[int64]*researchRun{}}
}

func (m *researchManager) get(jobID int64) *researchRun {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.runs[jobID]
}

func (m *researchManager) add(jobID int64, run *researchRun) {
	m.mu.Lock()
	m.runs[jobID] = run
	m.mu.Unlock()
}

func (m *researchManager) remove(jobID int64) {
	m.mu.Lock()
	delete(m.runs, jobID)
	m.mu.Unlock()
}

// researchEffortRounds maps a 1–5 effort level onto round limits, layered on
// top of the configured defaults:
//
//	1 (half arsed) — a single round
//	2 (low)        — two rounds
//	3 (default)    — the configured defaults, no bonus rounds
//	4 (extra)      — defaults plus one bonus creative round
//	5 (too much)   — defaults plus two bonus creative rounds
func researchEffortRounds(effort, defaultMaxRounds, defaultMinRounds int) (maxRounds, minRounds, extraRounds int) {
	switch effort {
	case 1:
		maxRounds = 1
	case 2:
		maxRounds = 2
	case 4:
		maxRounds, extraRounds = defaultMaxRounds, 1
	case 5:
		maxRounds, extraRounds = defaultMaxRounds, 2
	default: // 3 and any unexpected value
		maxRounds = defaultMaxRounds
	}
	minRounds = min(defaultMinRounds, maxRounds)
	return maxRounds, minRounds, extraRounds
}

// researchModel resolves the model to use for a research job.
func (s *Server) researchModel(requested string) string {
	if requested != "" {
		return requested
	}
	if s.cfg.Research.Model != "" {
		return s.cfg.Research.Model
	}
	return s.cfg.Server.DefaultModel
}

// ResumeResearchJobs restarts jobs that were in flight when the server last
// stopped. Each resumes from its last checkpoint. Call once at startup.
func (s *Server) ResumeResearchJobs() {
	jobs, err := s.store.ListResumableResearchJobs()
	if err != nil {
		log.Printf("research: could not list resumable jobs: %v", err)
		return
	}
	for i := range jobs {
		job := jobs[i]
		title := ""
		if job.Title != nil {
			title = *job.Title
		}
		log.Printf("Resuming research job id=%d user_id=%d round=%d title=%q query=%q", job.ID, job.UserID, job.Round, title, job.Query)
		go s.runResearch(&job)
	}
}

// runResearch executes one research job to completion. It runs detached from
// any HTTP request so the job survives the client disconnecting.
func (s *Server) runResearch(job *store.ResearchJob) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	run := &researchRun{cancel: cancel, subs: map[chan []byte]struct{}{}}
	s.research.add(job.ID, run)
	defer s.research.remove(job.ID)

	modelServer, err := s.cfg.ServerForModel(job.Model)
	if err != nil {
		s.finishResearch(job.ID, run, store.ResearchStatusError, nil, fmt.Sprintf("unknown model %q", job.Model), 0)
		return
	}

	loc := time.Local
	if s.cfg.Server.Timezone != "" {
		if l, tzErr := time.LoadLocation(s.cfg.Server.Timezone); tzErr == nil {
			loc = l
		}
	}

	// Build the LLM-facing prompt from whichever fields are set.
	llmQuery := job.Query
	if job.Title != nil && *job.Title != "" {
		if job.Query != "" {
			llmQuery = *job.Title + "\n\n" + job.Query
		} else {
			llmQuery = *job.Title
		}
	}

	rc := s.cfg.Research
	maxRounds, minRounds, extraRounds := researchEffortRounds(job.Effort, rc.MaxRounds, rc.MinRounds)
	maxTimeSeconds := rc.MaxTimeSeconds
	if job.MaxTimeSeconds > 0 {
		maxTimeSeconds = job.MaxTimeSeconds
	}
	cfg := research.Config{
		Query:                 llmQuery,
		Model:                 job.Model,
		Mode:                  job.Mode,
		ForceSearch:           job.ForceSearch,
		DeepReport:            job.DeepReport,
		APIBase:               modelServer.APIBase,
		APIKey:                modelServer.APIKey,
		SearXNGURL:            s.cfg.SearXNG.URL,
		Location:              loc,
		MaxRounds:             maxRounds,
		MaxTime:               time.Duration(maxTimeSeconds) * time.Second,
		MaxURLsPerRound:       rc.MaxURLsPerRound,
		MaxContentChars:       rc.MaxContentChars,
		MaxReportTokens:       rc.MaxReportTokens,
		ExtractionConcurrency: rc.ExtractionConcurrency,
		MinRounds:             minRounds,
		MaxEmptyRounds:        rc.MaxEmptyRounds,
		SynthesisWindow:       rc.SynthesisWindow,
		ExtraRounds:           extraRounds,
	}

	state := research.UnmarshalState(job.Round, job.EmptyRounds, job.ElapsedMS,
		job.Category, job.Plan, job.Report, job.Findings, job.QueriesUsed, job.AnalyzedURLs)

	lastPhase := ""
	onProgress := func(p research.Progress) {
		data, _ := json.Marshal(p)
		run.broadcast(data)
		// Streaming generation updates (~4/sec) are broadcast to the UI but
		// kept out of the log and the DB.
		if p.Generated > 0 {
			return
		}
		logResearchProgress(job.ID, p)
		if p.Phase != lastPhase && p.Phase != "warning" {
			lastPhase = p.Phase
			if err := s.store.UpdateResearchJobPhase(job.ID, store.ResearchStatusRunning, p.Phase); err != nil {
				log.Printf("research: job %d: update phase: %v", job.ID, err)
			}
		}
	}
	onCheckpoint := func(st research.State) {
		findings, queries, urls := research.MarshalState(st)
		if err := s.store.CheckpointResearchJob(job.ID, st.Round, st.EmptyRounds, st.ElapsedMS,
			st.Category, st.Plan, st.Report, findings, queries, urls); err != nil {
			log.Printf("research: job %d: checkpoint: %v", job.ID, err)
		}
	}

	if err := s.store.UpdateResearchJobPhase(job.ID, store.ResearchStatusRunning, "planning"); err != nil {
		log.Printf("research: job %d: mark running: %v", job.ID, err)
	}

	r := research.New(cfg, state, onProgress, onCheckpoint)
	started := time.Now()
	report, runErr := r.Run(ctx)
	elapsedMS := state.ElapsedMS + time.Since(started).Milliseconds()

	switch {
	case runErr == nil:
		log.Printf("Research job finished id=%d rounds=%d elapsed=%.1fs", job.ID, r.State().Round, float64(elapsedMS)/1000)
		s.finishResearch(job.ID, run, store.ResearchStatusDone, &report, "", elapsedMS)
	case errors.Is(runErr, context.Canceled) && run.wasCancelRequested():
		log.Printf("Research job cancelled id=%d", job.ID)
		s.finishResearch(job.ID, run, store.ResearchStatusCancelled, nil, "", elapsedMS)
	default:
		log.Printf("Research job failed id=%d: %v", job.ID, runErr)
		s.finishResearch(job.ID, run, store.ResearchStatusError, nil, runErr.Error(), elapsedMS)
	}
}

// logResearchProgress writes one log line per phase event so a tailed log
// shows what a job is doing. Streaming deltas are filtered out by the caller.
func logResearchProgress(jobID int64, p research.Progress) {
	const maxLogText = 300
	clip := func(s string) string {
		s = strings.ReplaceAll(s, "\n", " ")
		if len(s) > maxLogText {
			return s[:maxLogText] + "…"
		}
		return s
	}
	switch p.Phase {
	case "planning":
		if p.Message != "" {
			log.Printf("Planning research job_id=%d — %s", jobID, clip(p.Message))
		} else {
			log.Printf("Planning research job_id=%d", jobID)
		}
	case "searching":
		log.Printf("Searching web job_id=%d round=%d queries=%q", jobID, p.Round, p.Queries)
	case "reading":
		log.Printf("Reading page job_id=%d round=%d url=%s", jobID, p.Round, p.URL)
	case "analyzing":
		log.Printf("Synthesizing findings job_id=%d round=%d findings=%d", jobID, p.Round, p.TotalFindings)
	case "deciding":
		log.Printf("Deciding whether to stop job_id=%d round=%d — %s", jobID, p.Round, clip(p.Message))
	case "writing":
		log.Printf("Writing final report job_id=%d sources=%d findings=%d", jobID, p.TotalSources, p.TotalFindings)
	case "note":
		if p.Round > 0 {
			log.Printf("Research note job_id=%d round=%d — %s", jobID, p.Round, clip(p.Message))
		} else {
			log.Printf("Research note job_id=%d — %s", jobID, clip(p.Message))
		}
	case "warning":
		log.Printf("Research warning job_id=%d — %s", jobID, clip(p.Message))
	default:
		log.Printf("Research progress job_id=%d phase=%s %s", jobID, p.Phase, clip(p.Message))
	}
}

func (s *Server) finishResearch(jobID int64, run *researchRun, status string, finalReport *string, errMsg string, elapsedMS int64) {
	var errPtr *string
	if errMsg != "" {
		errPtr = &errMsg
	}
	if err := s.store.FinishResearchJob(jobID, status, finalReport, errPtr, elapsedMS); err != nil {
		log.Printf("research: job %d: finish: %v", jobID, err)
	}
	terminal := map[string]any{"status": status}
	if errMsg != "" {
		terminal["message"] = errMsg
	}
	data, _ := json.Marshal(terminal)
	run.finish(data)
}

// ── HTTP handlers ────────────────────────────────────────────

// researchJobView is the listing shape — heavy state columns omitted.
type researchJobView struct {
	ID                int64   `json:"id"`
	Title             *string `json:"title"`
	Query             string  `json:"query"`
	Model             string  `json:"model"`
	Mode              string  `json:"mode"`
	ForceSearch       bool    `json:"force_search"`
	DeepReport        bool    `json:"deep_report"`
	PauseRedditImport bool    `json:"pause_reddit_import"`
	Status            string  `json:"status"`
	Phase             *string `json:"phase"`
	Effort            int     `json:"effort"`
	MaxTimeSeconds    int     `json:"max_time_seconds"`
	Round             int     `json:"round"`
	ElapsedMS         int64   `json:"elapsed_ms"`
	Error             *string `json:"error"`
	CreatedAt         string  `json:"created_at"`
	UpdatedAt         string  `json:"updated_at"`
}

func researchView(j *store.ResearchJob) researchJobView {
	return researchJobView{
		ID: j.ID, Title: j.Title, Query: j.Query, Model: j.Model, Mode: j.Mode, ForceSearch: j.ForceSearch, DeepReport: j.DeepReport, PauseRedditImport: j.PauseRedditImport, Status: j.Status, Phase: j.Phase,
		Effort: j.Effort, MaxTimeSeconds: j.MaxTimeSeconds,
		Round: j.Round, ElapsedMS: j.ElapsedMS, Error: j.Error, CreatedAt: j.CreatedAt, UpdatedAt: j.UpdatedAt,
	}
}

// handleResearchDefaults reports the default form values so the UI can
// pre-fill the effort selector and time-limit field from server config.
func (s *Server) handleResearchDefaults(w http.ResponseWriter, r *http.Request) {
	maxTimeMinutes := s.cfg.Research.MaxTimeSeconds / 60
	if maxTimeMinutes < 1 {
		maxTimeMinutes = 10
	}
	writeJSON(w, http.StatusOK, map[string]int{
		"effort":           3,
		"max_time_minutes": maxTimeMinutes,
	})
}

func (s *Server) handleStartResearch(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)
	var req struct {
		Title             string `json:"title"`
		Query             string `json:"query"`
		Model             string `json:"model"`
		Mode              string `json:"mode"`
		ForceSearch       bool   `json:"force_search"`
		DeepReport        bool   `json:"deep_report"`
		PauseRedditImport bool   `json:"pause_reddit_import"`
		Effort            int    `json:"effort"`
		MaxTimeMinutes    int    `json:"max_time_minutes"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request")
		return
	}
	if req.Title == "" && req.Query == "" {
		writeError(w, http.StatusBadRequest, "title or query required")
		return
	}
	mode := req.Mode
	if mode == "" {
		mode = research.ModeResearch
	}
	if mode != research.ModeResearch && mode != research.ModeBrainstorm {
		writeError(w, http.StatusBadRequest, "invalid mode")
		return
	}
	effort := req.Effort
	if effort == 0 {
		effort = 3 // default
	}
	if effort < 1 || effort > 5 {
		writeError(w, http.StatusBadRequest, "effort must be between 1 and 5")
		return
	}
	// 0 means "use the configured default", resolved when the job runs.
	maxTimeSeconds := 0
	if req.MaxTimeMinutes > 0 {
		maxTimeSeconds = req.MaxTimeMinutes * 60
	}
	if s.cfg.SearXNG.URL == "" {
		writeError(w, http.StatusUnprocessableEntity, "research requires SearXNG — add [searxng] url to lemon.toml")
		return
	}
	model := s.researchModel(req.Model)
	if model == "" {
		writeError(w, http.StatusUnprocessableEntity, "no research model configured — set [research] model or default_model in lemon.toml")
		return
	}
	if _, err := s.cfg.ServerForModel(model); err != nil {
		writeError(w, http.StatusBadRequest, "unknown model")
		return
	}

	// ForceSearch only changes brainstorm-mode behaviour; ignore it otherwise.
	forceSearch := req.ForceSearch && mode == research.ModeBrainstorm
	job, err := s.store.CreateResearchJob(user.ID, req.Title, req.Query, model, mode, forceSearch, req.DeepReport, req.PauseRedditImport, effort, maxTimeSeconds)
	if err != nil {
		internalError(w, err)
		return
	}
	log.Printf("Starting research job id=%d user_id=%d model=%q mode=%q force_search=%t deep_report=%t pause_reddit_import=%t effort=%d max_time_s=%d title=%q query=%q", job.ID, user.ID, model, mode, forceSearch, req.DeepReport, req.PauseRedditImport, effort, maxTimeSeconds, req.Title, req.Query)
	go s.runResearch(job)
	writeJSON(w, http.StatusCreated, researchView(job))
}

func (s *Server) handleListResearch(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)
	jobs, err := s.store.ListResearchJobs(user.ID)
	if err != nil {
		internalError(w, err)
		return
	}
	views := make([]researchJobView, 0, len(jobs))
	for i := range jobs {
		views = append(views, researchView(&jobs[i]))
	}
	writeJSON(w, http.StatusOK, views)
}

func (s *Server) handleGetResearch(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	job, err := s.store.GetResearchJob(id, user.ID)
	if notFoundOr500(w, err) {
		return
	}
	writeJSON(w, http.StatusOK, job)
}

// handleResearchEvents streams progress events for a job over SSE. For a
// finished job it emits the terminal status immediately.
func (s *Server) handleResearchEvents(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	job, err := s.store.GetResearchJob(id, user.ID)
	if notFoundOr500(w, err) {
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("X-Accel-Buffering", "no")
	flusher := w.(http.Flusher)

	run := s.research.get(id)
	if run == nil {
		// Job is not running — emit its stored terminal state.
		terminal := map[string]any{"status": job.Status}
		if job.Error != nil {
			terminal["message"] = *job.Error
		}
		data, _ := json.Marshal(terminal)
		fmt.Fprintf(w, "data: %s\n\n", data)
		fmt.Fprintf(w, "data: [DONE]\n\n")
		flusher.Flush()
		return
	}

	ch := run.subscribe()
	defer run.unsubscribe(ch)

	for {
		select {
		case data, ok := <-ch:
			if !ok {
				fmt.Fprintf(w, "data: [DONE]\n\n")
				flusher.Flush()
				return
			}
			fmt.Fprintf(w, "data: %s\n\n", data)
			flusher.Flush()
		case <-r.Context().Done():
			return
		}
	}
}

func (s *Server) handleCancelResearch(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	if _, err := s.store.GetResearchJob(id, user.ID); err != nil {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	log.Printf("Cancelling research job id=%d user_id=%d", id, user.ID)
	run := s.research.get(id)
	if run == nil {
		writeError(w, http.StatusConflict, "job is not running")
		return
	}
	run.requestCancel()
	writeJSON(w, http.StatusOK, map[string]string{"status": "cancelling"})
}

func (s *Server) handleDeleteResearch(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	if s.research.get(id) != nil {
		writeError(w, http.StatusConflict, "job is running — cancel it first")
		return
	}
	log.Printf("Deleting research job id=%d user_id=%d", id, user.ID)
	if err := s.store.DeleteResearchJob(id, user.ID); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "not found")
			return
		}
		internalError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
