package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/stevelittlefish/lemon-chat/internal/redditimport"
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

// htmlReportModel resolves which model to use for the HTML report step. An
// explicit request wins, then the configured default, and finally the job's own
// model. An empty return means "reuse the job model".
func (s *Server) htmlReportModel(requested string) string {
	if requested != "" {
		return requested
	}
	return s.cfg.Research.HTMLReportModel
}

// researchLocation resolves the configured timezone, falling back to the
// server's local time when unset or invalid.
func (s *Server) researchLocation() *time.Location {
	if s.cfg.Server.Timezone != "" {
		if l, err := time.LoadLocation(s.cfg.Server.Timezone); err == nil {
			return l
		}
	}
	return time.Local
}

// researchLLMQuery builds the LLM-facing prompt for a job from whichever of its
// title and query fields are set.
func researchLLMQuery(job *store.ResearchJob) string {
	if job.Title != nil && *job.Title != "" {
		if job.Query != "" {
			return *job.Title + "\n\n" + job.Query
		}
		return *job.Title
	}
	return job.Query
}

// workerModel resolves which model to use for the worker tier (extraction plus
// the mechanical slug/classify/query-gen/decide calls). An explicit request
// wins, then the configured default, and finally the job's own model. An empty
// return means "reuse the job model".
func (s *Server) workerModel(requested string) string {
	if requested != "" {
		return requested
	}
	return s.cfg.Research.WorkerModel
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
		s.finishResearch(job.ID, run, store.ResearchStatusError, nil, fmt.Sprintf("unknown model %q", job.Model), 0, job.PriceUSD)
		return
	}

	// Resolve the optional worker-tier endpoint; a different model may live on a
	// different server, so we resolve the whole (model, base, key) triple. On any
	// error we fall back to the job model rather than failing the run.
	workerModel, workerAPIBase, workerAPIKey := "", "", ""
	if job.WorkerModel != nil && *job.WorkerModel != "" && *job.WorkerModel != job.Model {
		if ws, wErr := s.cfg.ServerForModel(*job.WorkerModel); wErr == nil {
			workerModel, workerAPIBase, workerAPIKey = *job.WorkerModel, ws.APIBase, ws.APIKey
		} else {
			log.Printf("research: job %d: worker model %q unresolved, using job model: %v", job.ID, *job.WorkerModel, wErr)
		}
	}

	loc := s.researchLocation()

	// Build the LLM-facing prompt from whichever fields are set.
	llmQuery := researchLLMQuery(job)

	rc := s.cfg.Research
	maxRounds, minRounds, extraRounds := researchEffortRounds(job.Effort, rc.MaxRounds, rc.MinRounds)
	maxTimeSeconds := rc.MaxTimeSeconds
	if job.MaxTimeSeconds > 0 {
		maxTimeSeconds = job.MaxTimeSeconds
	}
	cfg := research.Config{
		Query:                 llmQuery,
		SlugSource:            llmQuery,
		Model:                 job.Model,
		Mode:                  job.Mode,
		ForceSearch:           job.ForceSearch,
		DeepReport:            job.DeepReport,
		PauseRedditImport:     job.PauseRedditImport,
		APIBase:               modelServer.APIBase,
		APIKey:                modelServer.APIKey,
		WorkerModel:           workerModel,
		WorkerAPIBase:         workerAPIBase,
		WorkerAPIKey:          workerAPIKey,
		SearXNGURL:            s.cfg.SearXNG.URL,
		Location:              loc,
		MaxRounds:             maxRounds,
		MaxTime:               time.Duration(maxTimeSeconds) * time.Second,
		MaxURLsPerRound:       rc.MaxURLsPerRound,
		MaxContentChars:       rc.MaxContentChars,
		SynthesisTokens:       rc.SynthesisTokens,
		FinalReportTokens:     rc.FinalReportTokens,
		ExtractionConcurrency: rc.ExtractionConcurrency,
		MinRounds:             minRounds,
		MaxEmptyRounds:        rc.MaxEmptyRounds,
		SynthesisWindow:       rc.SynthesisWindow,
		ExtraRounds:           extraRounds,
	}
	if job.Title != nil && *job.Title != "" {
		cfg.SlugSource = *job.Title
	}
	cfg.OnRedditPause = func(pending research.PendingRedditRound) error {
		body, err := json.Marshal(pending)
		if err != nil {
			return err
		}
		return s.store.SetResearchJobAwaitingReddit(job.ID, pending.Request.RequestID, string(body), pending.ElapsedMS)
	}
	cfg.OnRedditRoundComplete = func(st research.State) error {
		findings, queries, urls := research.MarshalState(st)
		return s.store.CompleteResearchRedditRound(job.ID, st.Round, st.EmptyRounds, st.ElapsedMS,
			st.Category, st.Slug, st.Plan, st.Report, findings, queries, urls)
	}
	if job.PendingRedditRound != nil && (job.RedditResponse != nil || job.RedditSkipped) {
		var pending research.PendingRedditRound
		if err := json.Unmarshal([]byte(*job.PendingRedditRound), &pending); err != nil {
			s.finishResearch(job.ID, run, store.ResearchStatusError, nil, "stored Reddit pending round is invalid", job.ElapsedMS, job.PriceUSD)
			return
		}
		resume := &research.RedditResume{Pending: pending, Skipped: job.RedditSkipped}
		if job.RedditResponse != nil {
			var response redditimport.Response
			if err := json.Unmarshal([]byte(*job.RedditResponse), &response); err != nil {
				s.finishResearch(job.ID, run, store.ResearchStatusError, nil, "stored Reddit response is invalid", job.ElapsedMS, job.PriceUSD)
				return
			}
			pages, err := redditimport.ValidateAndNormalize(pending.Request, response)
			if err != nil {
				s.finishResearch(job.ID, run, store.ResearchStatusError, nil, "stored Reddit response failed validation", job.ElapsedMS, job.PriceUSD)
				return
			}
			resume.Pages = pages
		}
		cfg.RedditResume = resume
	}

	state := research.UnmarshalState(job.Round, job.EmptyRounds, job.ElapsedMS,
		job.Category, job.Slug, job.Plan, job.Report, job.Findings, job.QueriesUsed, job.AnalyzedURLs)
	state.PriceUSD = job.PriceUSD

	rlog := newResearchRunLog(s.cfg.Server.DataDir, job.ID)
	rlog.start(job, cfg)

	lastPhase := ""
	onProgress := func(p research.Progress) {
		data, _ := json.Marshal(p)
		run.broadcast(data)
		// Streaming generation updates (~4/sec) are broadcast to the UI but
		// kept out of the log, the DB, and the run-log.
		if p.Generated > 0 {
			return
		}
		rlog.event(p)
		logResearchProgress(job.ID, p)
		if p.Phase != lastPhase && p.Phase != "warning" {
			lastPhase = p.Phase
			if err := s.store.UpdateResearchJobPhase(job.ID, store.ResearchStatusRunning, p.Phase); err != nil {
				log.Printf("research: job %d: update phase: %v", job.ID, err)
			}
		}
	}
	onCheckpoint := func(st research.State) {
		rlog.checkpoint(st)
		findings, queries, urls := research.MarshalState(st)
		if err := s.store.CheckpointResearchJob(job.ID, st.Round, st.EmptyRounds, st.ElapsedMS, st.PriceUSD,
			st.Category, st.Slug, st.Plan, st.Report, findings, queries, urls); err != nil {
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
	case errors.Is(runErr, research.ErrAwaitingReddit):
		log.Printf("Pausing research for Reddit import id=%d elapsed=%.1fs", job.ID, float64(r.State().ElapsedMS)/1000)
		data, _ := json.Marshal(map[string]any{"status": store.ResearchStatusAwaitingReddit})
		run.finish(data)
	case runErr == nil:
		price := r.State().PriceUSD
		if job.AutoHTMLReport {
			price = s.autoGenerateHTMLReport(ctx, run, job, report, price)
		}
		log.Printf("Research job finished id=%d rounds=%d elapsed=%.1fs", job.ID, r.State().Round, float64(elapsedMS)/1000)
		rlog.finish(store.ResearchStatusDone, report, r.State())
		s.finishResearch(job.ID, run, store.ResearchStatusDone, &report, "", elapsedMS, price)
	case errors.Is(runErr, context.Canceled) && run.wasCancelRequested():
		log.Printf("Research job cancelled id=%d", job.ID)
		rlog.finish(store.ResearchStatusCancelled, "", r.State())
		s.finishResearch(job.ID, run, store.ResearchStatusCancelled, nil, "", elapsedMS, r.State().PriceUSD)
	default:
		log.Printf("Research job failed id=%d: %v", job.ID, runErr)
		rlog.finish(store.ResearchStatusError, "", r.State())
		s.finishResearch(job.ID, run, store.ResearchStatusError, nil, runErr.Error(), elapsedMS, r.State().PriceUSD)
	}
}

// autoGenerateHTMLReport renders the finished markdown report as a designed
// HTML document and stores it on the job's default report. It is best-effort:
// any failure is logged and the job still completes with its markdown report.
// The returned price includes the HTML generation cost when one was incurred.
func (s *Server) autoGenerateHTMLReport(ctx context.Context, run *researchRun, job *store.ResearchJob, markdown string, price *float64) *float64 {
	// The default report must exist before HTML can be attached to it.
	if err := s.store.UpsertDefaultResearchReport(job.ID, markdown, job.Model); err != nil {
		log.Printf("research: job %d: prepare default report for HTML: %v", job.ID, err)
		return price
	}
	// The HTML step may use a dedicated model; fall back to the job's own model.
	htmlModel := job.Model
	if job.HTMLReportModel != nil && *job.HTMLReportModel != "" {
		htmlModel = *job.HTMLReportModel
	}
	direction := ""
	if job.HTMLReportDirection != nil {
		direction = *job.HTMLReportDirection
	}
	title := job.Query
	if job.Title != nil && *job.Title != "" {
		title = *job.Title
	}
	data, _ := json.Marshal(research.Progress{Phase: "designing", Message: "designing the HTML report"})
	run.broadcast(data)

	timeout := time.Duration(s.cfg.Server.ResponseTimeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = 10 * time.Minute
	}
	genCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	html, cost, err := s.generateReportHTML(genCtx, htmlModel, title, markdown, direction, func(generated int, tail string) {
		data, _ := json.Marshal(research.Progress{Phase: "designing", Generated: generated, Snippet: tail})
		run.broadcast(data)
	})
	if err != nil {
		log.Printf("research: job %d: auto HTML report: %v", job.ID, err)
		return price
	}
	if err := s.store.SetDefaultResearchReportHTML(job.ID, html, direction, cost); err != nil {
		log.Printf("research: job %d: save auto HTML report: %v", job.ID, err)
		return price
	}
	log.Printf("Generated HTML report id=%d model=%q chars=%d", job.ID, htmlModel, len(html))
	return addResearchPrice(price, cost)
}

// addResearchPrice sums two optional prices, treating nil as absent.
func addResearchPrice(a, b *float64) *float64 {
	if a == nil {
		return b
	}
	if b == nil {
		return a
	}
	sum := *a + *b
	return &sum
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

func (s *Server) finishResearch(jobID int64, run *researchRun, status string, finalReport *string, errMsg string, elapsedMS int64, priceUSD *float64) {
	var errPtr *string
	if errMsg != "" {
		errPtr = &errMsg
	}
	if err := s.store.FinishResearchJob(jobID, status, finalReport, errPtr, elapsedMS, priceUSD); err != nil {
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
	ID                int64    `json:"id"`
	Title             *string  `json:"title"`
	Query             string   `json:"query"`
	Model             string   `json:"model"`
	Mode              string   `json:"mode"`
	ForceSearch       bool     `json:"force_search"`
	DeepReport        bool     `json:"deep_report"`
	PauseRedditImport bool     `json:"pause_reddit_import"`
	Status            string   `json:"status"`
	Phase             *string  `json:"phase"`
	Effort            int      `json:"effort"`
	MaxTimeSeconds    int      `json:"max_time_seconds"`
	Round             int      `json:"round"`
	ElapsedMS         int64    `json:"elapsed_ms"`
	PriceUSD          *float64 `json:"price_usd"`
	ReportCount       int      `json:"report_count"`
	Error             *string  `json:"error"`
	CreatedAt         string   `json:"created_at"`
	UpdatedAt         string   `json:"updated_at"`
}

func researchView(j *store.ResearchJob) researchJobView {
	return researchJobView{
		ID: j.ID, Title: j.Title, Query: j.Query, Model: j.Model, Mode: j.Mode, ForceSearch: j.ForceSearch, DeepReport: j.DeepReport, PauseRedditImport: j.PauseRedditImport, Status: j.Status, Phase: j.Phase,
		Effort: j.Effort, MaxTimeSeconds: j.MaxTimeSeconds,
		Round: j.Round, ElapsedMS: j.ElapsedMS, PriceUSD: j.PriceUSD, Error: j.Error, CreatedAt: j.CreatedAt, UpdatedAt: j.UpdatedAt,
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
		Title               string `json:"title"`
		Query               string `json:"query"`
		Model               string `json:"model"`
		Mode                string `json:"mode"`
		ForceSearch         bool   `json:"force_search"`
		DeepReport          bool   `json:"deep_report"`
		AutoHTMLReport      bool   `json:"auto_html_report"`
		HTMLReportDirection string `json:"html_report_direction"`
		HTMLReportModel     string `json:"html_report_model"`
		WorkerModel         string `json:"worker_model"`
		PauseRedditImport   bool   `json:"pause_reddit_import"`
		Effort              int    `json:"effort"`
		MaxTimeMinutes      int    `json:"max_time_minutes"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request")
		return
	}
	if req.Title == "" && req.Query == "" {
		writeError(w, http.StatusBadRequest, "title or query required")
		return
	}
	req.HTMLReportDirection = strings.TrimSpace(req.HTMLReportDirection)
	if len(req.HTMLReportDirection) > 2000 {
		writeError(w, http.StatusBadRequest, "html report direction must be 2000 characters or fewer")
		return
	}
	// The style direction and model override are only meaningful when the HTML
	// report is requested.
	if !req.AutoHTMLReport {
		req.HTMLReportDirection = ""
		req.HTMLReportModel = ""
	}
	req.HTMLReportModel = strings.TrimSpace(req.HTMLReportModel)
	req.WorkerModel = strings.TrimSpace(req.WorkerModel)
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
	// The HTML report step may use a separate model; an empty result means "reuse
	// the job model", resolved when the report is generated.
	htmlReportModel := ""
	if req.AutoHTMLReport {
		htmlReportModel = s.htmlReportModel(req.HTMLReportModel)
		if htmlReportModel != "" {
			if _, err := s.cfg.ServerForModel(htmlReportModel); err != nil {
				writeError(w, http.StatusBadRequest, "unknown HTML report model")
				return
			}
		}
	}

	// The worker tier may use a separate model; an empty result means "reuse the
	// job model", resolved when the job runs. A worker model equal to the job
	// model is stored as empty (no override).
	workerModel := s.workerModel(req.WorkerModel)
	if workerModel == model {
		workerModel = ""
	}
	if workerModel != "" {
		if _, err := s.cfg.ServerForModel(workerModel); err != nil {
			writeError(w, http.StatusBadRequest, "unknown worker model")
			return
		}
	}

	// ForceSearch only changes brainstorm-mode behaviour; ignore it otherwise.
	forceSearch := req.ForceSearch && mode == research.ModeBrainstorm
	job, err := s.store.CreateResearchJob(user.ID, req.Title, req.Query, model, mode, forceSearch, req.DeepReport, req.PauseRedditImport, req.AutoHTMLReport, req.HTMLReportDirection, htmlReportModel, workerModel, effort, maxTimeSeconds)
	if err != nil {
		internalError(w, err)
		return
	}
	log.Printf("Starting research job id=%d user_id=%d model=%q worker_model=%q mode=%q force_search=%t deep_report=%t auto_html_report=%t html_report_model=%q pause_reddit_import=%t effort=%d max_time_s=%d title=%q query=%q", job.ID, user.ID, model, workerModel, mode, forceSearch, req.DeepReport, req.AutoHTMLReport, htmlReportModel, req.PauseRedditImport, effort, maxTimeSeconds, req.Title, req.Query)
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
	reportCounts, err := s.store.ListNonDefaultResearchReportCounts(user.ID)
	if err != nil {
		internalError(w, err)
		return
	}
	views := make([]researchJobView, 0, len(jobs))
	for i := range jobs {
		view := researchView(&jobs[i])
		view.ReportCount = reportCounts[jobs[i].ID]
		views = append(views, view)
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
	var request *redditimport.Request
	if job.PendingRedditRound != nil {
		var pending research.PendingRedditRound
		if json.Unmarshal([]byte(*job.PendingRedditRound), &pending) == nil {
			request = &pending.Request
		}
	}
	reports, err := s.store.ListNonDefaultResearchReports(job.ID)
	if err != nil {
		internalError(w, err)
		return
	}
	// Surface whether the default report has a designed HTML version so the UI
	// can offer to open it.
	reportHTML := false
	if def, err := s.store.GetDefaultResearchReport(job.ID); err == nil {
		reportHTML = def.HTML != ""
	} else if !errors.Is(err, store.ErrNotFound) {
		internalError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, struct {
		*store.ResearchJob
		RedditRequest *redditimport.Request         `json:"reddit_request,omitempty"`
		Reports       []store.ResearchReportSummary `json:"reports"`
		ReportHTML    bool                          `json:"report_html"`
	}{ResearchJob: job, RedditRequest: request, Reports: reports, ReportHTML: reportHTML})
}

// handleGetResearchReportDocument serves the default report's designed HTML
// document (the auto-generated or attached HTML version).
func (s *Server) handleGetResearchReportDocument(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	if _, err := s.store.GetResearchJob(id, user.ID); notFoundOr500(w, err) {
		return
	}
	def, err := s.store.GetDefaultResearchReport(id)
	if notFoundOr500(w, err) {
		return
	}
	if def.HTML == "" {
		writeError(w, http.StatusNotFound, "no HTML report")
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Content-Security-Policy", "sandbox allow-popups allow-popups-to-escape-sandbox; default-src 'none'; style-src 'unsafe-inline'; img-src data:; font-src data:")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(def.HTML))
}

func (s *Server) handleResearchRedditImport(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	job, err := s.store.GetResearchJob(id, user.ID)
	if notFoundOr500(w, err) {
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, redditimport.MaxTotalChars+512*1024)
	var response redditimport.Response
	if err := json.NewDecoder(r.Body).Decode(&response); err != nil {
		writeError(w, http.StatusBadRequest, "invalid or oversized Reddit response")
		return
	}
	request, err := pendingRedditRequest(job)
	if err != nil {
		writeError(w, http.StatusConflict, err.Error())
		return
	}
	if _, err := redditimport.ValidateAndNormalize(request, response); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	canonical, err := json.Marshal(response)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid Reddit response")
		return
	}
	canonicalText := string(canonical)
	transitioned, err := s.store.ResumeResearchRedditImport(id, user.ID, response.RequestID, &canonicalText, false)
	if err != nil {
		internalError(w, err)
		return
	}
	if !transitioned {
		fresh, getErr := s.store.GetResearchJob(id, user.ID)
		if getErr == nil && fresh.RedditRequestID != nil && *fresh.RedditRequestID == response.RequestID && fresh.RedditResponse != nil && *fresh.RedditResponse == canonicalText {
			writeJSON(w, http.StatusOK, map[string]any{"status": fresh.Status, "resumed": false})
			return
		}
		writeError(w, http.StatusConflict, "Reddit request is stale or already resolved")
		return
	}
	log.Printf("Importing Reddit response research_job_id=%d user_id=%d request_id=%q pages=%d", id, user.ID, response.RequestID, len(response.Pages))
	s.resumeResearchAfterReddit(w, id, user.ID)
}

func (s *Server) handleResearchRedditSkip(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	_, err := s.store.GetResearchJob(id, user.ID)
	if notFoundOr500(w, err) {
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 4096)
	var body struct {
		RequestID string `json:"request_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.RequestID == "" {
		writeError(w, http.StatusBadRequest, "request_id is required")
		return
	}
	transitioned, err := s.store.ResumeResearchRedditImport(id, user.ID, body.RequestID, nil, true)
	if err != nil {
		internalError(w, err)
		return
	}
	if !transitioned {
		fresh, getErr := s.store.GetResearchJob(id, user.ID)
		if getErr == nil && fresh.RedditRequestID != nil && *fresh.RedditRequestID == body.RequestID && fresh.RedditSkipped {
			writeJSON(w, http.StatusOK, map[string]any{"status": fresh.Status, "resumed": false})
			return
		}
		writeError(w, http.StatusConflict, "Reddit request is stale or already resolved")
		return
	}
	log.Printf("Skipping Reddit import research_job_id=%d user_id=%d request_id=%q", id, user.ID, body.RequestID)
	s.resumeResearchAfterReddit(w, id, user.ID)
}

func pendingRedditRequest(job *store.ResearchJob) (redditimport.Request, error) {
	if job.RedditRequestID == nil || job.PendingRedditRound == nil {
		return redditimport.Request{}, errors.New("research job has no pending Reddit request")
	}
	var pending research.PendingRedditRound
	if err := json.Unmarshal([]byte(*job.PendingRedditRound), &pending); err != nil {
		return redditimport.Request{}, errors.New("stored Reddit request is invalid")
	}
	if pending.Request.RequestID != *job.RedditRequestID {
		return redditimport.Request{}, errors.New("stored Reddit request does not match")
	}
	return pending.Request, nil
}

func (s *Server) resumeResearchAfterReddit(w http.ResponseWriter, id, userID int64) {
	job, err := s.store.GetResearchJob(id, userID)
	if err != nil {
		internalError(w, err)
		return
	}
	go s.runResearch(job)
	writeJSON(w, http.StatusOK, map[string]any{"status": store.ResearchStatusPending, "resumed": true})
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
	job, err := s.store.GetResearchJob(id, user.ID)
	if err != nil {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	log.Printf("Cancelling research job id=%d user_id=%d", id, user.ID)
	run := s.research.get(id)
	if run == nil {
		if job.Status == store.ResearchStatusAwaitingReddit {
			if err := s.store.FinishResearchJob(id, store.ResearchStatusCancelled, nil, nil, job.ElapsedMS, job.PriceUSD); err != nil {
				internalError(w, err)
				return
			}
			writeJSON(w, http.StatusOK, map[string]string{"status": store.ResearchStatusCancelled})
			return
		}
		writeError(w, http.StatusConflict, "job is not running")
		return
	}
	run.requestCancel()
	writeJSON(w, http.StatusOK, map[string]string{"status": "cancelling"})
}

// handleGetResearchDebug serves the on-disk diagnostic run-log (config, outcome,
// and milestone timeline) as JSON so the UI can show it inline without a bundle
// download. Returns {available:false} when the job has no run-log.
func (s *Server) handleGetResearchDebug(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	if _, err := s.store.GetResearchJob(id, user.ID); notFoundOr500(w, err) {
		return
	}
	writeJSON(w, http.StatusOK, readResearchDebug(s.cfg.Server.DataDir, id))
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
	// Best-effort removal of the on-disk diagnostic run-log for this job.
	os.RemoveAll(researchRunLogDir(s.cfg.Server.DataDir, id))
	w.WriteHeader(http.StatusNoContent)
}
