package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/stevelittlefish/lemon-chat/internal/llm"
	"github.com/stevelittlefish/lemon-chat/internal/research"
	"github.com/stevelittlefish/lemon-chat/internal/store"
)

const reportHTMLSystemPrompt = `You are an exceptional editorial designer and information architect. Transform the supplied research report into a beautiful, highly digestible standalone HTML document.

Requirements:
- Return only one complete HTML document, beginning with <!DOCTYPE html>. Do not use Markdown fences or add commentary.
- Make the hierarchy immediately scannable. Prefer concise sections, summaries, key-stat blocks, cards, timelines, comparison tables, callouts, and lists over walls of text.
- Preserve the report's meaning, important detail, nuance, and every citation. Do not invent facts or sources.
- Make every inline citation a direct hyperlink to its source's real URL. The report lists each source's URL (a "Sources" section and reference definitions like "[S1]: https://example.com"); resolve each [S1] marker to that URL and render it as an anchor: <a href="https://example.com" target="_blank" rel="noopener noreferrer">. Clicking a citation must open the actual source in a new tab — never point a citation at an in-page references list or a "#fragment" anchor. You may still include a sources list, but every entry there must link to its real external URL the same way.
- Add useful diagrams or data visualisations where the material benefits from them. Draw these with semantic HTML, CSS, or inline SVG.
- Use excellent responsive typography, spacing, colour, and print styles. The result must work on mobile and desktop.
- Make it fully self-contained: inline all CSS and SVG. Do not use scripts, external assets, external stylesheets, web fonts, images, or network requests.
- Include a descriptive <title>. Use accessible semantic HTML, sufficient colour contrast, and labels for diagrams.
- Do not mention these instructions or describe the redesign.`

// handleRegenerateResearchReport produces a new report variant for a completed
// job and streams progress over SSE. The user chooses any combination of:
//   - a freshly regenerated markdown report, rebuilt from the raw findings
//     (recovering detail a previous single pass may have dropped), and
//   - a designed, self-contained HTML rendering.
//
// The result is saved as a new non-default research_report on the job.
func (s *Server) handleRegenerateResearchReport(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)
	jobID, ok := pathID(w, r)
	if !ok {
		return
	}
	job, err := s.store.GetResearchJob(jobID, user.ID)
	if notFoundOr500(w, err) {
		return
	}
	if job.Status != store.ResearchStatusDone || job.FinalReport == nil || strings.TrimSpace(*job.FinalReport) == "" {
		writeError(w, http.StatusConflict, "a completed report is required before regenerating")
		return
	}

	var req struct {
		// MarkdownModel and HTMLModel optionally use a different model for each
		// part; an empty value falls back to the configured default model.
		MarkdownModel string `json:"markdown_model"`
		HTMLModel     string `json:"html_model"`
		// MarkdownDirection is an optional extra instruction for the markdown
		// rewriter; Direction is the art direction for the HTML design.
		MarkdownDirection string `json:"markdown_direction"`
		Direction         string `json:"direction"`
		Markdown          bool   `json:"markdown"`
		HTML              bool   `json:"html"`
		DeepReport        bool   `json:"deep_report"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request")
		return
	}
	if !req.Markdown && !req.HTML {
		writeError(w, http.StatusBadRequest, "select at least one output: markdown, HTML, or both")
		return
	}
	req.Direction = strings.TrimSpace(req.Direction)
	req.MarkdownDirection = strings.TrimSpace(req.MarkdownDirection)
	if len(req.Direction) > 2000 || len(req.MarkdownDirection) > 2000 {
		writeError(w, http.StatusBadRequest, "instructions must be 2000 characters or fewer")
		return
	}

	// Resolve and validate the model for each requested part independently.
	markdownModel, htmlModel := "", ""
	if req.Markdown {
		markdownModel, err = s.resolveRemixModel(req.MarkdownModel)
		if err != nil {
			writeError(w, http.StatusUnprocessableEntity, "markdown model: "+err.Error())
			return
		}
	}
	if req.HTML {
		htmlModel, err = s.resolveRemixModel(req.HTMLModel)
		if err != nil {
			writeError(w, http.StatusUnprocessableEntity, "HTML model: "+err.Error())
			return
		}
	}

	title := job.Query
	if job.Title != nil && *job.Title != "" {
		title = *job.Title
	}
	timeout := time.Duration(s.cfg.Server.ResponseTimeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = 10 * time.Minute
	}
	ctx, cancel := context.WithTimeout(r.Context(), timeout)
	defer cancel()
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "streaming unsupported")
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("X-Accel-Buffering", "no")
	writeReportEvent(w, flusher, map[string]any{"phase": "connecting", "message": "connecting to model"})
	s.appendResearchTrace(jobID, research.TraceEvent{EventType: "report_regeneration_started", Phase: "writing", Data: map[string]any{
		"markdown": req.Markdown, "html": req.HTML, "deep_report": req.DeepReport,
		"markdown_model": markdownModel, "html_model": htmlModel,
	}})

	// Step 1: optionally regenerate the markdown from the raw findings.
	var freshMarkdown string
	var markdownCost *float64
	if req.Markdown {
		researcher, err := s.newReportRegenerator(job, markdownModel, req.DeepReport, req.MarkdownDirection, func(p research.Progress) {
			if p.Generated > 0 {
				writeReportEvent(w, flusher, map[string]any{"phase": "generating-markdown", "generated": p.Generated, "snippet": p.Snippet})
			}
		})
		if err != nil {
			writeReportEvent(w, flusher, map[string]any{"error": err.Error()})
			writeReportDone(w, flusher)
			return
		}
		writeReportEvent(w, flusher, map[string]any{"phase": "generating-markdown", "message": "regenerating the report from raw findings"})
		md, cost, err := researcher.RegenerateReport(ctx)
		if err != nil {
			log.Printf("research report: markdown regeneration failed job_id=%d user_id=%d: %v", job.ID, user.ID, err)
			s.appendResearchTrace(jobID, research.TraceEvent{EventType: "report_regeneration_failed", Phase: "writing", Message: err.Error(), Data: map[string]any{"stage": "markdown", "model": markdownModel}})
			writeReportEvent(w, flusher, map[string]any{"error": err.Error()})
			writeReportDone(w, flusher)
			return
		}
		freshMarkdown = md
		markdownCost = cost
	}

	// The markdown persisted with this variant: the fresh version when
	// regenerated, otherwise a copy of the master so the variant is
	// self-contained.
	var storedMarkdown *string
	if freshMarkdown != "" {
		storedMarkdown = &freshMarkdown
	} else if def, err := s.store.GetDefaultResearchReport(jobID); err == nil {
		storedMarkdown = def.Markdown
	}

	// Step 2: optionally design an HTML rendering, from the fresh markdown when
	// one was produced, otherwise from the existing master report.
	var html string
	var htmlCost *float64
	if req.HTML {
		if _, budgetErr := s.remainingResearchCostBudget(job); budgetErr != nil {
			s.appendResearchTrace(jobID, research.TraceEvent{EventType: "html_generation_failed", Phase: "designing", Message: budgetErr.Error(), Data: map[string]any{"stage": "budget", "model": htmlModel}})
			writeReportEvent(w, flusher, map[string]any{"error": budgetErr.Error()})
			writeReportDone(w, flusher)
			return
		}
		source := *job.FinalReport
		if freshMarkdown != "" {
			source = freshMarkdown
		}
		writeReportEvent(w, flusher, map[string]any{"phase": "generating-html", "message": "designing the HTML document"})
		s.appendResearchTrace(jobID, research.TraceEvent{EventType: "html_generation_started", Phase: "designing", Data: map[string]any{"automatic": false, "model": htmlModel}})
		h, cost, err := s.generateReportHTML(ctx, jobID, htmlModel, title, source, req.Direction, func(generated int, tail string) {
			writeReportEvent(w, flusher, map[string]any{"phase": "generating-html", "generated": generated, "snippet": tail})
		})
		if err != nil {
			log.Printf("research report: HTML generation failed job_id=%d user_id=%d: %v", job.ID, user.ID, err)
			s.appendResearchTrace(jobID, research.TraceEvent{EventType: "html_generation_failed", Phase: "designing", Message: err.Error(), Data: map[string]any{"stage": "generate", "model": htmlModel}})
			writeReportEvent(w, flusher, map[string]any{"error": err.Error()})
			writeReportDone(w, flusher)
			return
		}
		html = h
		htmlCost = cost
		s.appendResearchTrace(jobID, research.TraceEvent{EventType: "html_generation_completed", Phase: "designing", Data: map[string]any{"automatic": false, "model": htmlModel, "characters": len(html), "cost_usd": cost}})
	}

	// The report carries one model label; show both parts when they differ.
	storedModel := remixModelLabel(markdownModel, htmlModel)
	writeReportEvent(w, flusher, map[string]any{"phase": "saving", "message": "saving report"})
	report, err := s.store.CreateResearchReport(jobID, storedMarkdown, html, storedModel, req.Direction, addCost(markdownCost, htmlCost), false)
	if err != nil {
		log.Printf("research report: save failed job_id=%d user_id=%d: %v", job.ID, user.ID, err)
		s.appendResearchTrace(jobID, research.TraceEvent{EventType: "report_regeneration_failed", Phase: "saving", Message: err.Error(), Data: map[string]any{"stage": "save"}})
		writeReportEvent(w, flusher, map[string]any{"error": "could not save report"})
		writeReportDone(w, flusher)
		return
	}
	log.Printf("Creating research report id=%d research_job_id=%d user_id=%d markdown_model=%q html_model=%q markdown=%t html=%t", report.ID, jobID, user.ID, markdownModel, htmlModel, storedMarkdown != nil && *storedMarkdown != "", html != "")
	s.appendResearchTrace(jobID, research.TraceEvent{EventType: "report_regeneration_completed", Phase: "terminal", Data: map[string]any{"report_id": report.ID, "price_usd": report.PriceUSD}})
	summary := store.ResearchReportSummary{
		ID: report.ID, ResearchJobID: jobID, Model: storedModel, Direction: req.Direction,
		HasMarkdown: storedMarkdown != nil && *storedMarkdown != "", HasHTML: html != "",
		PriceUSD: report.PriceUSD, CreatedAt: report.CreatedAt,
	}
	writeReportEvent(w, flusher, map[string]any{"phase": "complete", "report": summary})
	writeReportDone(w, flusher)
}

// newReportRegenerator constructs a Researcher pre-loaded with a completed job's
// saved state so its report phase can be re-run without searching the web again.
// Only the writer tier is needed (the report phase never calls the worker model).
// instruction is an optional extra prompt appended to the report-writing prompts.
func (s *Server) newReportRegenerator(job *store.ResearchJob, model string, deepReport bool, instruction string, onProgress func(research.Progress)) (*research.Researcher, error) {
	modelServer, err := s.cfg.ServerForModel(model)
	if err != nil {
		return nil, fmt.Errorf("unknown model %q", model)
	}
	llmQuery := researchLLMQuery(job)
	limits := s.researchTokenLimits(model, model)
	maxTimeSeconds := job.MaxTimeSeconds
	if maxTimeSeconds <= 0 {
		maxTimeSeconds = s.cfg.Research.MaxTimeSeconds
	}
	maxCostUSD, err := s.remainingResearchCostBudget(job)
	if err != nil {
		return nil, err
	}
	cfg := research.Config{
		Query:                   llmQuery,
		SlugSource:              llmQuery,
		Model:                   model,
		Mode:                    job.Mode,
		DeepReport:              deepReport,
		ReportInstruction:       instruction,
		APIBase:                 modelServer.APIBase,
		APIKey:                  modelServer.APIKey,
		TokenLimits:             limits,
		MaxTime:                 time.Duration(maxTimeSeconds) * time.Second,
		MaxCostUSD:              maxCostUSD,
		FinalReservePercent:     0,
		MaxWritingContinuations: s.cfg.Research.MaxWritingContinuations,
		Location:                s.researchLocation(),
		OnTrace: func(event research.TraceEvent) {
			s.appendResearchTrace(job.ID, event)
		},
	}
	state := research.UnmarshalState(job.Round, job.EmptyRounds, job.ElapsedMS,
		job.Category, job.Slug, job.Plan, job.Report, job.Findings, job.QueriesUsed, job.AnalyzedURLs)
	state.PriceUSD = job.PriceUSD
	state.ElapsedMS = 0
	s.configureResearchCallTrace(job.ID, &cfg)
	return research.New(cfg, state, onProgress, nil), nil
}

// remainingResearchCostBudget returns the known-cost allowance left across all
// calls belonging to a job. Zero means cost limiting is disabled.
func (s *Server) remainingResearchCostBudget(job *store.ResearchJob) (float64, error) {
	limit := job.MaxCostUSD
	if limit <= 0 {
		limit = s.cfg.Research.MaxCostUSD
	}
	if limit <= 0 {
		return 0, nil
	}
	calls, err := s.store.ListResearchLLMCalls(job.ID)
	if err != nil {
		return 0, err
	}
	spent := 0.0
	for _, call := range calls {
		if call.PriceUSD != nil {
			spent += *call.PriceUSD
		}
	}
	remaining := limit - spent
	if remaining <= 0 {
		return 0, fmt.Errorf("research cost budget is exhausted")
	}
	return remaining, nil
}

// resolveRemixModel resolves a requested remix model, falling back to the
// configured default model, and validates that it exists.
func (s *Server) resolveRemixModel(requested string) (string, error) {
	model := strings.TrimSpace(requested)
	if model == "" {
		model = s.cfg.Server.DefaultModel
	}
	if model == "" {
		return "", fmt.Errorf("no default model configured")
	}
	if _, err := s.cfg.ServerForModel(model); err != nil {
		return "", fmt.Errorf("unknown model %q", model)
	}
	return model, nil
}

// remixModelLabel is the single model string stored on a report variant: one
// model when both parts share it (or only one part ran), or "md / html" when the
// markdown and HTML parts used different models.
func remixModelLabel(markdownModel, htmlModel string) string {
	switch {
	case markdownModel == "" || markdownModel == htmlModel:
		return htmlModel
	case htmlModel == "":
		return markdownModel
	default:
		return markdownModel + " / " + htmlModel
	}
}

// generateReportHTML renders report markdown as a self-contained HTML document
// using the editorial-design prompt shared by the interactive regenerate flow
// and the auto-generated report step. onProgress, if non-nil, receives throttled
// updates carrying the characters generated so far and the tail of the output.
func (s *Server) generateReportHTML(ctx context.Context, jobID int64, model, title, markdown, direction string, onProgress func(generated int, tail string)) (string, *float64, error) {
	modelServer, err := s.cfg.ServerForModel(model)
	if err != nil {
		return "", nil, fmt.Errorf("unknown model %q", model)
	}
	userPrompt := fmt.Sprintf("Document title: %s\n\nResearch report:\n%s", title, markdown)
	if direction != "" {
		userPrompt += "\n\nAdditional art direction from the user:\n" + direction
	}
	var generated strings.Builder
	lastProgress := time.Time{}
	messages := []llm.Message{{Role: "system", Content: reportHTMLSystemPrompt}, {Role: "user", Content: userPrompt}}
	htmlTokenLimit := s.researchTokenLimits(model, model).HTMLReport
	parameters := map[string]any{"temperature": 0.7, "stream": true, "max_tokens": htmlTokenLimit}
	encodedMessages, _ := json.Marshal(messages)
	encodedParameters, _ := json.Marshal(parameters)
	call, callErr := s.store.BeginResearchLLMCall(jobID, "designing", "html_report", 0, model, modelServer.APIBase, string(encodedMessages), string(encodedParameters))
	callID := int64(0)
	if callErr != nil {
		log.Printf("research: job %d: begin HTML LLM call: %v", jobID, callErr)
	} else {
		callID = call.ID
	}
	started := time.Now()
	completion, err := llm.ChatCompleteStreamWithUsage(ctx, s.modelClient, modelServer.APIBase+"/chat/completions", modelServer.APIKey, model,
		messages, parameters, func(delta string) {
			generated.WriteString(delta)
			if onProgress == nil || time.Since(lastProgress) < 250*time.Millisecond {
				return
			}
			lastProgress = time.Now()
			tail := generated.String()
			if len(tail) > 320 {
				tail = tail[len(tail)-320:]
			}
			onProgress(generated.Len(), tail)
		})
	if callID != 0 {
		errMessage := ""
		status := http.StatusOK
		if err != nil {
			errMessage = err.Error()
			status = llm.ErrorHTTPStatus(err)
		}
		if completeErr := s.store.CompleteResearchLLMCall(callID, time.Since(started).Milliseconds(), completion.Content,
			completion.FinishReason, string(completion.RawUsage), completion.UsageCost(), status, errMessage); completeErr != nil {
			log.Printf("research: job %d: finish HTML LLM call id=%d: %v", jobID, callID, completeErr)
		}
	}
	if err != nil {
		if callID != 0 {
			_ = s.store.SetResearchLLMCallDisposition(callID, "rejected")
		}
		return "", nil, fmt.Errorf("HTML generation failed: %w", err)
	}
	if completion.Truncated(htmlTokenLimit) {
		if callID != 0 {
			_ = s.store.SetResearchLLMCallDisposition(callID, "rejected")
		}
		return "", nil, fmt.Errorf("HTML generation was truncated at the model token limit (finish_reason=%q)", completion.FinishReason)
	}
	html, err := normalizeReportHTML(completion.Content)
	if err != nil {
		if callID != 0 {
			_ = s.store.SetResearchLLMCallDisposition(callID, "rejected")
		}
		return "", nil, err
	}
	if callID != 0 {
		_ = s.store.SetResearchLLMCallDisposition(callID, "accepted")
	}
	return html, completion.UsageCost(), nil
}

// addCost sums two optional costs; nil is treated as zero, and the result is nil
// only when both inputs are nil.
func addCost(a, b *float64) *float64 {
	if a == nil {
		return b
	}
	if b == nil {
		return a
	}
	sum := *a + *b
	return &sum
}

func writeReportEvent(w http.ResponseWriter, flusher http.Flusher, event any) {
	data, _ := json.Marshal(event)
	fmt.Fprintf(w, "data: %s\n\n", data)
	flusher.Flush()
}

func writeReportDone(w http.ResponseWriter, flusher http.Flusher) {
	fmt.Fprint(w, "data: [DONE]\n\n")
	flusher.Flush()
}

func normalizeReportHTML(content string) (string, error) {
	html := strings.TrimSpace(content)
	if start := strings.Index(strings.ToLower(html), "<!doctype html"); start >= 0 {
		html = html[start:]
	}
	if end := strings.LastIndex(strings.ToLower(html), "</html>"); end >= 0 {
		html = html[:end+len("</html>")]
	}
	lower := strings.ToLower(html)
	if !strings.HasPrefix(lower, "<!doctype html") || !strings.Contains(lower, "<html") || !strings.Contains(lower, "</html>") {
		return "", fmt.Errorf("model did not return a complete HTML document")
	}
	return html, nil
}

func (s *Server) getOwnedResearchReport(w http.ResponseWriter, r *http.Request) (*store.ResearchReport, bool) {
	user := currentUser(r)
	jobID, ok := pathID(w, r)
	if !ok {
		return nil, false
	}
	if _, err := s.store.GetResearchJob(jobID, user.ID); notFoundOr500(w, err) {
		return nil, false
	}
	reportID, err := strconv.ParseInt(r.PathValue("reportId"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid report id")
		return nil, false
	}
	report, err := s.store.GetResearchReport(reportID, jobID)
	if notFoundOr500(w, err) {
		return nil, false
	}
	return report, true
}

func (s *Server) handleGetResearchReportVariant(w http.ResponseWriter, r *http.Request) {
	report, ok := s.getOwnedResearchReport(w, r)
	if !ok {
		return
	}
	writeJSON(w, http.StatusOK, report)
}

func (s *Server) handleGetResearchReportVariantDocument(w http.ResponseWriter, r *http.Request) {
	report, ok := s.getOwnedResearchReport(w, r)
	if !ok {
		return
	}
	if report.HTML == "" {
		writeError(w, http.StatusNotFound, "no HTML report")
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Content-Security-Policy", "sandbox allow-popups allow-popups-to-escape-sandbox; default-src 'none'; style-src 'unsafe-inline'; img-src data:; font-src data:")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(report.HTML))
}
