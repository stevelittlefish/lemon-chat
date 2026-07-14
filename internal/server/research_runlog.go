package server

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"
	"sync"
	"time"

	"github.com/stevelittlefish/lemon-chat/internal/research"
	"github.com/stevelittlefish/lemon-chat/internal/store"
)

// researchRunLog writes a per-job diagnostic trail to
// <data_dir>/research/<job_id>/, mirroring the attachments layout: milestone
// events as JSONL, a full state snapshot and the evolving report per round, and
// a meta.json summarising the run's config and outcome. It exists purely for
// debugging research behaviour and is downloadable as a zip bundle. All writes
// are best-effort — a failing run-log never affects the research run itself.
//
//	<data_dir>/research/<job_id>/
//	  meta.json          # config + outcome — the "why did it stop" summary
//	  events.jsonl       # one line per milestone (the streamed progress log)
//	  reports/
//	    round-00-plan.txt
//	    round-NN.md      # State.Report snapshot after each round
//	    final.md
//	  snapshots/
//	    round-NN.json    # full State per round
type researchRunLog struct {
	dir      string
	disabled bool

	mu          sync.Mutex
	meta        researchRunMeta
	planWritten bool
	// lastDecision holds the most recent stop-check verdict; terminalReason holds
	// a terminal note/warning (round cap, time budget, empty rounds) which, when
	// present, is the real reason the run ended. Both are teed from onProgress so
	// meta.json can record a stop_reason with no engine change.
	lastDecision   string
	terminalReason string
}

// researchRunMeta is the serialised meta.json. Start fields are written at job
// start; the outcome fields are filled in when the run finishes.
type researchRunMeta struct {
	JobID             int64    `json:"job_id"`
	Model             string   `json:"model"`
	WorkerModel       string   `json:"worker_model,omitempty"`
	Mode              string   `json:"mode"`
	Effort            int      `json:"effort"`
	SynthesisTokens   int      `json:"synthesis_tokens"`
	FinalReportTokens int      `json:"final_report_tokens"`
	MinRounds         int      `json:"min_rounds"`
	MaxRounds         int      `json:"max_rounds"`
	MaxTimeSeconds    float64  `json:"max_time_seconds"`
	Locale            string   `json:"locale,omitempty"`
	GitCommit         string   `json:"git_commit"`
	StartedAt         string   `json:"started_at"`
	Status            string   `json:"status,omitempty"`
	StopReason        string   `json:"stop_reason,omitempty"`
	RoundsCompleted   int      `json:"rounds_completed,omitempty"`
	ElapsedMS         int64    `json:"elapsed_ms,omitempty"`
	FindingsCount     int      `json:"findings_count,omitempty"`
	PriceUSD          *float64 `json:"price_usd,omitempty"`
	EndedAt           string   `json:"ended_at,omitempty"`
}

// researchRunLogDir is the on-disk root for one job's diagnostic trail.
func researchRunLogDir(dataDir string, jobID int64) string {
	return filepath.Join(dataDir, "research", fmt.Sprintf("%d", jobID))
}

// newResearchRunLog prepares the on-disk directory tree for a job. A failure to
// create it disables the run-log rather than failing the research run.
func newResearchRunLog(dataDir string, jobID int64) *researchRunLog {
	dir := researchRunLogDir(dataDir, jobID)
	rl := &researchRunLog{dir: dir}
	for _, sub := range []string{"reports", "snapshots"} {
		if err := os.MkdirAll(filepath.Join(dir, sub), 0o755); err != nil {
			log.Printf("research: job %d: run-log disabled: %v", jobID, err)
			rl.disabled = true
			return rl
		}
	}
	return rl
}

// start records the effective config at job start and writes the first meta.json.
func (rl *researchRunLog) start(job *store.ResearchJob, cfg research.Config) {
	if rl == nil || rl.disabled {
		return
	}
	locale := ""
	if cfg.Location != nil {
		locale = cfg.Location.String()
	}
	rl.mu.Lock()
	rl.meta = researchRunMeta{
		JobID:             job.ID,
		Model:             cfg.Model,
		WorkerModel:       cfg.WorkerModel,
		Mode:              cfg.Mode,
		Effort:            job.Effort,
		SynthesisTokens:   cfg.SynthesisTokens,
		FinalReportTokens: cfg.FinalReportTokens,
		MinRounds:         cfg.MinRounds,
		MaxRounds:         cfg.MaxRounds,
		MaxTimeSeconds:    cfg.MaxTime.Seconds(),
		Locale:            locale,
		GitCommit:         buildCommit(),
		StartedAt:         time.Now().UTC().Format(time.RFC3339),
	}
	rl.writeMetaLocked()
	rl.mu.Unlock()
}

// event appends one milestone line to events.jsonl. Streaming token updates
// (Generated > 0) are skipped, leaving exactly the milestone log. It also tees
// the stop decision and any terminal note/warning for meta.json.
func (rl *researchRunLog) event(p research.Progress) {
	if rl == nil || rl.disabled {
		return
	}
	rl.mu.Lock()
	defer rl.mu.Unlock()

	switch p.Phase {
	case "deciding":
		if p.Message != "" {
			rl.lastDecision = p.Message
		}
	case "note", "warning":
		if isTerminalReason(p.Message) {
			rl.terminalReason = p.Message
		}
	}

	line, err := json.Marshal(struct {
		TS      string `json:"ts"`
		Phase   string `json:"phase"`
		Round   int    `json:"round,omitempty"`
		Message string `json:"message,omitempty"`
	}{time.Now().UTC().Format(time.RFC3339Nano), p.Phase, p.Round, p.Message})
	if err != nil {
		return
	}
	rl.appendFile("events.jsonl", append(line, '\n'))
}

// checkpoint dumps the full State snapshot and the evolving report for the round,
// plus the plan the first time it appears.
func (rl *researchRunLog) checkpoint(st research.State) {
	if rl == nil || rl.disabled {
		return
	}
	rl.mu.Lock()
	defer rl.mu.Unlock()

	if !rl.planWritten && strings.TrimSpace(st.Plan) != "" {
		rl.writeFile(filepath.Join("reports", "round-00-plan.txt"), []byte(st.Plan))
		rl.planWritten = true
	}
	if snapshot, err := json.MarshalIndent(st, "", "  "); err == nil {
		rl.writeFile(filepath.Join("snapshots", fmt.Sprintf("round-%02d.json", st.Round)), snapshot)
	}
	if st.Report != "" {
		rl.writeFile(filepath.Join("reports", fmt.Sprintf("round-%02d.md", st.Round)), []byte(st.Report))
	}
}

// finish writes the final report (when present) and updates meta.json with the
// run's outcome. stopReason falls back to the derived decision/terminal notes.
func (rl *researchRunLog) finish(status, report string, st research.State) {
	if rl == nil || rl.disabled {
		return
	}
	rl.mu.Lock()
	defer rl.mu.Unlock()

	if report != "" {
		rl.writeFile(filepath.Join("reports", "final.md"), []byte(report))
	}
	reason := rl.terminalReason
	if reason == "" {
		reason = rl.lastDecision
	}
	rl.meta.Status = status
	rl.meta.StopReason = reason
	rl.meta.RoundsCompleted = st.Round
	rl.meta.ElapsedMS = st.ElapsedMS
	rl.meta.FindingsCount = len(st.Findings)
	rl.meta.PriceUSD = st.PriceUSD
	rl.meta.EndedAt = time.Now().UTC().Format(time.RFC3339)
	rl.writeMetaLocked()
}

// writeMetaLocked serialises meta.json. Caller must hold rl.mu.
func (rl *researchRunLog) writeMetaLocked() {
	if data, err := json.MarshalIndent(rl.meta, "", "  "); err == nil {
		rl.writeFile("meta.json", append(data, '\n'))
	}
}

func (rl *researchRunLog) writeFile(name string, data []byte) {
	if err := os.WriteFile(filepath.Join(rl.dir, name), data, 0o644); err != nil {
		log.Printf("research: job %d: run-log write %s: %v", rl.meta.JobID, name, err)
	}
}

func (rl *researchRunLog) appendFile(name string, data []byte) {
	f, err := os.OpenFile(filepath.Join(rl.dir, name), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		log.Printf("research: job %d: run-log append %s: %v", rl.meta.JobID, name, err)
		return
	}
	defer f.Close()
	_, _ = f.Write(data)
}

// isTerminalReason matches the milestone messages that signal the run ended for
// a structural reason (round cap, time budget, exhausted searches) rather than a
// transient warning, so they can be recorded as the stop_reason.
func isTerminalReason(message string) bool {
	for _, marker := range []string{
		"reached the maximum number of rounds",
		"time budget exhausted",
		"consecutive empty rounds",
		"no new search queries generated",
	} {
		if strings.Contains(message, marker) {
			return true
		}
	}
	return false
}

// buildCommit returns the VCS revision embedded at build time, suffixed -dirty
// when the working tree had uncommitted changes. Falls back to "unknown".
func buildCommit() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "unknown"
	}
	rev, dirty := "", false
	for _, s := range info.Settings {
		switch s.Key {
		case "vcs.revision":
			rev = s.Value
		case "vcs.modified":
			dirty = s.Value == "true"
		}
	}
	if rev == "" {
		return "unknown"
	}
	if dirty {
		rev += "-dirty"
	}
	return rev
}
