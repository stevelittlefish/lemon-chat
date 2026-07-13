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
	"github.com/stevelittlefish/lemon-chat/internal/store"
)

const remixSystemPrompt = `You are an exceptional editorial designer and information architect. Transform the supplied research report into a beautiful, highly digestible standalone HTML document.

Requirements:
- Return only one complete HTML document, beginning with <!DOCTYPE html>. Do not use Markdown fences or add commentary.
- Make the hierarchy immediately scannable. Prefer concise sections, summaries, key-stat blocks, cards, timelines, comparison tables, callouts, and lists over walls of text.
- Preserve the report's meaning, important detail, nuance, and citations. Do not invent facts or sources.
- Add useful diagrams or data visualisations where the material benefits from them. Draw these with semantic HTML, CSS, or inline SVG.
- Use excellent responsive typography, spacing, colour, and print styles. The result must work on mobile and desktop.
- Make it fully self-contained: inline all CSS and SVG. Do not use scripts, external assets, external stylesheets, web fonts, images, or network requests.
- Include a descriptive <title>. Use accessible semantic HTML, sufficient colour contrast, and labels for diagrams.
- Do not mention these instructions or describe the redesign.`

func (s *Server) handleCreateResearchRemix(w http.ResponseWriter, r *http.Request) {
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
		writeError(w, http.StatusConflict, "a completed report is required before creating a remix")
		return
	}

	var req struct {
		Model     string `json:"model"`
		Direction string `json:"direction"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request")
		return
	}
	req.Direction = strings.TrimSpace(req.Direction)
	if len(req.Direction) > 2000 {
		writeError(w, http.StatusBadRequest, "direction must be 2000 characters or fewer")
		return
	}
	model := strings.TrimSpace(req.Model)
	if model == "" {
		model = s.cfg.Server.DefaultModel
	}
	if model == "" {
		writeError(w, http.StatusUnprocessableEntity, "no default model configured")
		return
	}
	modelServer, err := s.cfg.ServerForModel(model)
	if err != nil {
		writeError(w, http.StatusBadRequest, "unknown model")
		return
	}

	title := job.Query
	if job.Title != nil && *job.Title != "" {
		title = *job.Title
	}
	userPrompt := fmt.Sprintf("Document title: %s\n\nResearch report:\n%s", title, *job.FinalReport)
	if req.Direction != "" {
		userPrompt += "\n\nAdditional art direction from the user:\n" + req.Direction
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
	writeRemixEvent(w, flusher, map[string]any{"phase": "connecting", "message": "connecting to model"})

	var generated strings.Builder
	lastProgress := time.Time{}
	completion, err := llm.ChatCompleteStreamWithUsage(ctx, s.modelClient, modelServer.APIBase+"/chat/completions", modelServer.APIKey, model,
		[]llm.Message{{Role: "system", Content: remixSystemPrompt}, {Role: "user", Content: userPrompt}},
		map[string]any{"temperature": 0.7}, func(delta string) {
			generated.WriteString(delta)
			if time.Since(lastProgress) < 250*time.Millisecond {
				return
			}
			lastProgress = time.Now()
			tail := generated.String()
			if len(tail) > 320 {
				tail = tail[len(tail)-320:]
			}
			writeRemixEvent(w, flusher, map[string]any{"phase": "generating", "generated": generated.Len(), "snippet": tail})
		})
	if err != nil {
		log.Printf("research remix: generation failed job_id=%d user_id=%d: %v", job.ID, user.ID, err)
		writeRemixEvent(w, flusher, map[string]any{"error": "remix generation failed: " + err.Error()})
		writeRemixDone(w, flusher)
		return
	}
	writeRemixEvent(w, flusher, map[string]any{"phase": "validating", "message": "checking the HTML document"})
	html, err := normalizeRemixHTML(completion.Content)
	if err != nil {
		writeRemixEvent(w, flusher, map[string]any{"error": err.Error()})
		writeRemixDone(w, flusher)
		return
	}
	writeRemixEvent(w, flusher, map[string]any{"phase": "saving", "message": "saving remix"})
	remix, err := s.store.CreateResearchRemix(job.ID, model, req.Direction, html, completion.UsageCost())
	if err != nil {
		log.Printf("research remix: save failed job_id=%d user_id=%d: %v", job.ID, user.ID, err)
		writeRemixEvent(w, flusher, map[string]any{"error": "could not save remix"})
		writeRemixDone(w, flusher)
		return
	}
	logResearchRemix(remix, user.ID)
	writeRemixEvent(w, flusher, map[string]any{"phase": "complete", "remix": remix})
	writeRemixDone(w, flusher)
}

func writeRemixEvent(w http.ResponseWriter, flusher http.Flusher, event any) {
	data, _ := json.Marshal(event)
	fmt.Fprintf(w, "data: %s\n\n", data)
	flusher.Flush()
}

func writeRemixDone(w http.ResponseWriter, flusher http.Flusher) {
	fmt.Fprint(w, "data: [DONE]\n\n")
	flusher.Flush()
}

func normalizeRemixHTML(content string) (string, error) {
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

func logResearchRemix(remix *store.ResearchRemix, userID int64) {
	// Kept separate so the handler's successful path remains easy to scan.
	log.Printf("Creating research remix id=%d research_job_id=%d user_id=%d model=%q", remix.ID, remix.ResearchJobID, userID, remix.Model)
}

func (s *Server) getOwnedResearchRemix(w http.ResponseWriter, r *http.Request) (*store.ResearchRemix, bool) {
	user := currentUser(r)
	jobID, ok := pathID(w, r)
	if !ok {
		return nil, false
	}
	if _, err := s.store.GetResearchJob(jobID, user.ID); notFoundOr500(w, err) {
		return nil, false
	}
	remixID, err := strconv.ParseInt(r.PathValue("remixId"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid remix id")
		return nil, false
	}
	remix, err := s.store.GetResearchRemix(remixID, jobID)
	if notFoundOr500(w, err) {
		return nil, false
	}
	return remix, true
}

func (s *Server) handleGetResearchRemix(w http.ResponseWriter, r *http.Request) {
	remix, ok := s.getOwnedResearchRemix(w, r)
	if !ok {
		return
	}
	writeJSON(w, http.StatusOK, remix)
}

func (s *Server) handleGetResearchRemixDocument(w http.ResponseWriter, r *http.Request) {
	remix, ok := s.getOwnedResearchRemix(w, r)
	if !ok {
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Content-Security-Policy", "sandbox; default-src 'none'; style-src 'unsafe-inline'; img-src data:; font-src data:")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(remix.HTML))
}
