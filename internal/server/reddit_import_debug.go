package server

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/stevelittlefish/lemon-chat/internal/redditimport"
	"github.com/stevelittlefish/lemon-chat/internal/research"
	"github.com/stevelittlefish/lemon-chat/internal/searx"
)

type redditHarnessRequest struct {
	Action   string                       `json:"action"`
	Query    string                       `json:"query,omitempty"`
	URLs     []redditimport.RequestedPage `json:"urls,omitempty"`
	Request  *redditimport.Request        `json:"request,omitempty"`
	Response *redditimport.Response       `json:"response,omitempty"`
	Goal     string                       `json:"goal,omitempty"`
	Model    string                       `json:"model,omitempty"`
}

func (s *Server) handleRedditImportHarness(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, redditimport.MaxTotalChars+512*1024)
	var req redditHarnessRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid or oversized request")
		return
	}

	switch req.Action {
	case "prepare":
		s.handleRedditHarnessPrepare(w, r, req)
	case "validate", "extract":
		s.handleRedditHarnessValidate(w, r, req)
	default:
		writeError(w, http.StatusBadRequest, "unknown action")
	}
}

func (s *Server) handleRedditHarnessPrepare(w http.ResponseWriter, r *http.Request, input redditHarnessRequest) {
	pages := append([]redditimport.RequestedPage(nil), input.URLs...)
	searchResults := []map[string]string{}
	if query := strings.TrimSpace(input.Query); query != "" {
		if s.cfg.SearXNG.URL == "" {
			writeError(w, http.StatusBadRequest, "SearXNG is not configured")
			return
		}
		results, err := searx.Search(r.Context(), s.cfg.SearXNG.URL, query+" site:reddit.com", 1)
		if err != nil {
			writeError(w, http.StatusBadGateway, err.Error())
			return
		}
		for _, result := range results {
			pages = append(pages, redditimport.RequestedPage{URL: result.URL, Title: result.Title})
			searchResults = append(searchResults, map[string]string{"url": result.URL, "title": result.Title})
		}
	}

	accepted := make([]redditimport.RequestedPage, 0, len(pages))
	rejected := make([]map[string]string, 0)
	for _, page := range pages {
		canonical, _, err := redditimport.CanonicalizeURL(page.URL)
		if err != nil {
			rejected = append(rejected, map[string]string{"url": page.URL, "reason": err.Error()})
			continue
		}
		accepted = append(accepted, redditimport.RequestedPage{URL: canonical, Title: page.Title})
	}
	if len(accepted) == 0 {
		writeJSON(w, http.StatusOK, map[string]any{"search_results": searchResults, "rejected": rejected})
		return
	}
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		internalError(w, err)
		return
	}
	bundle, err := redditimport.NewRequest(hex.EncodeToString(b), accepted, redditimport.CaptureLimits{})
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"search_results": searchResults, "rejected": rejected, "request": bundle})
}

func (s *Server) handleRedditHarnessValidate(w http.ResponseWriter, r *http.Request, input redditHarnessRequest) {
	if input.Request == nil || input.Response == nil {
		writeError(w, http.StatusBadRequest, "request and response bundles are required")
		return
	}
	pages, err := redditimport.ValidateAndNormalize(*input.Request, *input.Response)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	result := map[string]any{"pages": pages}
	if input.Action == "extract" {
		if strings.TrimSpace(input.Goal) == "" {
			writeError(w, http.StatusBadRequest, "an extraction goal is required")
			return
		}
		model := s.researchModel(input.Model)
		modelServer, err := s.cfg.ServerForModel(model)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		rc := s.cfg.Research
		researcher := research.New(research.Config{
			Query: input.Goal, Model: model, APIBase: modelServer.APIBase, APIKey: modelServer.APIKey,
			MaxContentChars: rc.MaxContentChars, Location: time.Local,
		}, research.State{}, nil, nil)
		findings := make([]research.Finding, 0, len(pages))
		for _, page := range pages {
			if page.Failure != "" {
				continue
			}
			if finding := researcher.ExtractText(r.Context(), 0, page.URL, page.Title, page.Content); finding != nil {
				findings = append(findings, *finding)
			}
		}
		result["findings"] = findings
		result["formatted_findings"] = formatHarnessFindings(findings)
	}
	writeJSON(w, http.StatusOK, result)
}

func formatHarnessFindings(findings []research.Finding) string {
	var b strings.Builder
	for i, finding := range findings {
		fmt.Fprintf(&b, "## Finding %d\n\nTitle: %s\n\nURL: %s\n\nSummary: %s\n\nEvidence: %s\n\n", i+1, finding.Title, finding.URL, finding.Summary, finding.Evidence)
	}
	return strings.TrimSpace(b.String())
}
