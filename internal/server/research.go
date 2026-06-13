package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
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
		log.Printf("Resuming research job id=%d user_id=%d round=%d query=%q", job.ID, job.UserID, job.Round, job.Query)
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

	rc := s.cfg.Research
	cfg := research.Config{
		Query:                 job.Query,
		Model:                 job.Model,
		APIBase:               modelServer.APIBase,
		APIKey:                modelServer.APIKey,
		SearXNGURL:            s.cfg.SearXNG.URL,
		Location:              loc,
		MaxRounds:             rc.MaxRounds,
		MaxTime:               time.Duration(rc.MaxTimeSeconds) * time.Second,
		MaxURLsPerRound:       rc.MaxURLsPerRound,
		MaxContentChars:       rc.MaxContentChars,
		MaxReportTokens:       rc.MaxReportTokens,
		ExtractionConcurrency: rc.ExtractionConcurrency,
		MinRounds:             rc.MinRounds,
		MaxEmptyRounds:        rc.MaxEmptyRounds,
		SynthesisWindow:       rc.SynthesisWindow,
	}

	state := research.UnmarshalState(job.Round, job.EmptyRounds, job.ElapsedMS,
		job.Category, job.Plan, job.Report, job.Findings, job.QueriesUsed, job.AnalyzedURLs)

	lastPhase := ""
	onProgress := func(p research.Progress) {
		data, _ := json.Marshal(p)
		run.broadcast(data)
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
	ID        int64   `json:"id"`
	Query     string  `json:"query"`
	Model     string  `json:"model"`
	Status    string  `json:"status"`
	Phase     *string `json:"phase"`
	Round     int     `json:"round"`
	ElapsedMS int64   `json:"elapsed_ms"`
	Error     *string `json:"error"`
	CreatedAt string  `json:"created_at"`
	UpdatedAt string  `json:"updated_at"`
}

func researchView(j *store.ResearchJob) researchJobView {
	return researchJobView{
		ID: j.ID, Query: j.Query, Model: j.Model, Status: j.Status, Phase: j.Phase,
		Round: j.Round, ElapsedMS: j.ElapsedMS, Error: j.Error, CreatedAt: j.CreatedAt, UpdatedAt: j.UpdatedAt,
	}
}

func (s *Server) handleStartResearch(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)
	var req struct {
		Query string `json:"query"`
		Model string `json:"model"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request")
		return
	}
	if req.Query == "" {
		writeError(w, http.StatusBadRequest, "query required")
		return
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

	job, err := s.store.CreateResearchJob(user.ID, req.Query, model)
	if err != nil {
		internalError(w, err)
		return
	}
	log.Printf("Starting research job id=%d user_id=%d model=%q query=%q", job.ID, user.ID, model, req.Query)
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
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	job, err := s.store.GetResearchJob(id, user.ID)
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	if err != nil {
		internalError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, job)
}

// handleResearchEvents streams progress events for a job over SSE. For a
// finished job it emits the terminal status immediately.
func (s *Server) handleResearchEvents(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	job, err := s.store.GetResearchJob(id, user.ID)
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	if err != nil {
		internalError(w, err)
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
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
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
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
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
