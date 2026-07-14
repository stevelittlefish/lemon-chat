// Package research implements the iterative research engine: an
// LLM-in-the-loop Plan → Classify → (Think → Search → Extract → Synthesise →
// Decide)* → Final Report pipeline producing a long-form, cited markdown
// report. Ported from the Python deep research spec (docs/deep_research_spec.html).
package research

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/stevelittlefish/lemon-chat/internal/redditimport"
	"github.com/stevelittlefish/lemon-chat/internal/searx"
)

// Per-phase LLM timeouts (spec §5).
const (
	planningTimeout   = 90 * time.Second
	maxSlugLength     = 32
	classifyTimeout   = 15 * time.Second
	queryTimeout      = 120 * time.Second
	extractionTimeout = 90 * time.Second
	// Synthesis and final report are heavy generation tasks — local models
	// routinely need >60s; a shorter timeout would discard a whole round.
	synthesisTimeout = 180 * time.Second
	stopTimeout      = 60 * time.Second
)

var ErrAwaitingReddit = errors.New("research is awaiting a Reddit import")

var validCategories = map[string]bool{"product": true, "comparison": true, "howto": true, "factcheck": true}

var validBrainstormFormats = map[string]bool{
	"design-doc": true, "options": true, "ideas": true, "analysis": true, "explainer": true,
}

var reCombinedSourceCitation = regexp.MustCompile(`\[((?:S\d+\s*(?:[,;]|\band\b)?\s*){2,})\]`)

var ErrBudgetExhausted = errors.New("research budget exhausted")

// Research modes. "research" is the default web-search-driven pipeline;
// "brainstorm" is an ideation-driven variant where each round the model
// develops ideas and decides for itself whether it needs to search the web.
const (
	ModeResearch   = "research"
	ModeBrainstorm = "brainstorm"
)

// Finding is one successfully extracted page.
type Finding struct {
	URL      string `json:"url"`
	Title    string `json:"title"`
	OGImage  string `json:"og_image,omitempty"`
	Rational string `json:"rational,omitempty"`
	Evidence string `json:"evidence,omitempty"`
	Summary  string `json:"summary"`
}

// AnalyzedURL is any URL the engine attempted to read, whether or not
// extraction succeeded.
type AnalyzedURL struct {
	URL   string `json:"url"`
	Title string `json:"title"`
}

// State is the resumable engine state. It is checkpointed after planning and
// after every completed round; a job restarted from a persisted State picks
// up at the next round.
type State struct {
	Round        int           `json:"round"`
	EmptyRounds  int           `json:"empty_rounds"`
	ElapsedMS    int64         `json:"elapsed_ms"`
	Category     string        `json:"category"`
	Slug         string        `json:"slug"`
	Plan         string        `json:"plan"`
	Report       string        `json:"report"`
	Findings     []Finding     `json:"findings"`
	QueriesUsed  []string      `json:"queries_used"`
	AnalyzedURLs []AnalyzedURL `json:"analyzed_urls"`
	PriceUSD     *float64      `json:"price_usd,omitempty"`
}

// TokenLimits are the effective, model-clamped output ceilings persisted with
// a job so resumed runs use the same budgets even if server config changes.
type TokenLimits struct {
	Synthesis   int `json:"synthesis"`
	Memory      int `json:"memory"`
	FinalReport int `json:"final_report"`
	Section     int `json:"section"`
	HTMLReport  int `json:"html_report"`
}

// Config carries everything a run needs; all tuning parameters come from the
// [research] section of lemon.toml.
type Config struct {
	Query      string
	SlugSource string
	Model      string
	Mode       string // ModeResearch (default) or ModeBrainstorm
	// ForceSearch (brainstorm only) guarantees at least one web search: the
	// first round must produce a query rather than leaving it to the model.
	ForceSearch bool
	// DeepReport replaces the single-shot final write with a section-based
	// pipeline (outline → refine → per-section write from raw findings → glue),
	// producing a longer, more detailed report. Applies to both modes.
	DeepReport bool
	// ReportInstruction is an optional extra instruction from the user, appended
	// to the report-writing prompts (single-pass final report and the deep-report
	// outline/section writers). Empty means no extra instruction.
	ReportInstruction     string
	PauseRedditImport     bool
	OnRedditPause         func(PendingRedditRound) error
	RedditResume          *RedditResume
	OnRedditRoundComplete func(State) error
	APIBase               string
	APIKey                string
	// WorkerModel, WorkerAPIBase, and WorkerAPIKey optionally point the worker
	// tier (extraction + the mechanical slug/classify/query-gen/decide calls) at
	// a separate, cheaper model. When WorkerModel is empty the worker tier reuses
	// the job model (Model/APIBase/APIKey).
	WorkerModel   string
	WorkerAPIBase string
	WorkerAPIKey  string
	SearXNGURL    string
	Location      *time.Location

	MaxRounds               int
	MaxTime                 time.Duration
	MaxURLsPerRound         int
	MaxContentChars         int
	TokenLimits             TokenLimits
	MaxCostUSD              float64
	FinalReservePercent     int
	MaxWritingContinuations int
	ExtractionConcurrency   int
	MinRounds               int
	MaxEmptyRounds          int
	SynthesisWindow         int
	// ExtraRounds is the number of bonus "creative" rounds to run after the
	// report would normally be considered complete (effort 4 → 1, effort 5 → 2).
	ExtraRounds int
	// OnTrace receives immutable, structured debug events for persistence. It is
	// independent of OnProgress: trace events are durable job artifacts, while
	// progress events drive the live UI.
	OnTrace func(TraceEvent)
	// LLM call hooks persist requests before transport starts and finalize them
	// afterward. The returned ID is opaque to the engine and is used only to
	// associate completion and semantic-disposition updates.
	OnLLMCallStart       func(LLMCallStart) int64
	OnLLMCallFinish      func(LLMCallFinish)
	OnLLMCallDisposition func(callID int64, disposition string)
}

// PendingRedditRound is the complete durable boundary between search and
// extraction. Resuming from it does not repeat query generation or web search.
type PendingRedditRound struct {
	Round        int                  `json:"round"`
	Creativity   int                  `json:"creativity"`
	Queries      []string             `json:"queries"`
	OrdinaryURLs []AnalyzedURL        `json:"ordinary_urls"`
	Request      redditimport.Request `json:"request"`
	ElapsedMS    int64                `json:"elapsed_ms"`
}

type RedditResume struct {
	Pending PendingRedditRound
	Pages   []redditimport.NormalizedPage
	Skipped bool
}

// Progress is emitted at each phase transition. Events with Generated > 0
// are live generation updates (throttled to ~4/sec) carrying the tail of the
// text being generated.
type Progress struct {
	Phase         string   `json:"phase"` // planning | searching | reading | analyzing | deciding | writing | note | warning
	Round         int      `json:"round,omitempty"`
	Message       string   `json:"message,omitempty"`
	URL           string   `json:"url,omitempty"`
	Title         string   `json:"title,omitempty"`
	Queries       []string `json:"queries,omitempty"`
	TotalSources  int      `json:"total_sources,omitempty"`
	TotalFindings int      `json:"total_findings,omitempty"`
	Generated     int      `json:"generated,omitempty"` // chars generated so far in a streaming LLM call
	Snippet       string   `json:"snippet,omitempty"`   // tail of the text being generated
}

// TraceEvent is one structured event in a research job's append-only debug
// history. Data must contain JSON-marshalable values and must never contain
// credentials.
type TraceEvent struct {
	EventType string `json:"event_type"`
	Phase     string `json:"phase"`
	Round     int    `json:"round,omitempty"`
	Message   string `json:"message,omitempty"`
	Data      any    `json:"data,omitempty"`
}

type LLMCallStart struct {
	Operation  string              `json:"operation"`
	Phase      string              `json:"phase"`
	Round      int                 `json:"round"`
	Model      string              `json:"model"`
	APIBase    string              `json:"api_base"`
	Messages   []map[string]string `json:"messages"`
	Parameters map[string]any      `json:"parameters"`
}

type LLMCallFinish struct {
	CallID       int64           `json:"call_id"`
	DurationMS   int64           `json:"duration_ms"`
	Response     string          `json:"response"`
	FinishReason string          `json:"finish_reason"`
	Usage        json.RawMessage `json:"usage"`
	PriceUSD     *float64        `json:"price_usd"`
	HTTPStatus   int             `json:"http_status"`
	Error        string          `json:"error"`
}

type Researcher struct {
	cfg    Config
	state  State
	client *http.Client

	// writer is the strong job model (plan, synthesis, final/deep report);
	// worker is the cheap/fast model for extraction and the mechanical calls.
	// worker defaults to writer when no separate worker model is configured.
	writer modelEndpoint
	worker modelEndpoint

	startTime   time.Time
	baseElapsed int64 // ms accumulated by previous runs of a resumed job

	// onProgress is called at each phase transition; onCheckpoint persists
	// the state so the run can be resumed after a crash. Both may be nil.
	onProgress   func(Progress)
	onCheckpoint func(State)
	priceMu      sync.Mutex
}

// New creates a researcher. Pass a zero State for a fresh run or a persisted
// State to resume an interrupted one.
func New(cfg Config, state State, onProgress func(Progress), onCheckpoint func(State)) *Researcher {
	if cfg.Location == nil {
		cfg.Location = time.Local
	}
	if cfg.Mode == "" {
		cfg.Mode = ModeResearch
	}
	if cfg.TokenLimits.Synthesis <= 0 {
		cfg.TokenLimits.Synthesis = 8192
	}
	if cfg.TokenLimits.FinalReport <= 0 {
		cfg.TokenLimits.FinalReport = 32768
	}
	if cfg.TokenLimits.Memory <= 0 {
		cfg.TokenLimits.Memory = 6000
	}
	if cfg.TokenLimits.Section <= 0 {
		cfg.TokenLimits.Section = 12288
	}
	if cfg.TokenLimits.HTMLReport <= 0 {
		cfg.TokenLimits.HTMLReport = 32768
	}
	writer := modelEndpoint{Model: cfg.Model, APIBase: cfg.APIBase, APIKey: cfg.APIKey}
	worker := writer
	if cfg.WorkerModel != "" {
		worker = modelEndpoint{Model: cfg.WorkerModel, APIBase: cfg.WorkerAPIBase, APIKey: cfg.WorkerAPIKey}
	}
	return &Researcher{cfg: cfg, state: state, client: http.DefaultClient, writer: writer, worker: worker, onProgress: onProgress, onCheckpoint: onCheckpoint}
}

// brainstorm reports whether this is an ideation run rather than a
// web-search-driven research run.
func (r *Researcher) brainstorm() bool { return r.cfg.Mode == ModeBrainstorm }

func (r *Researcher) progress(p Progress) {
	if p.Generated == 0 {
		r.trace("progress", p.Phase, p.Round, p.Message, p)
	}
	if r.onProgress != nil {
		r.onProgress(p)
	}
}

func (r *Researcher) trace(eventType, phase string, round int, message string, data any) {
	if r.cfg.OnTrace != nil {
		r.cfg.OnTrace(TraceEvent{EventType: eventType, Phase: phase, Round: round, Message: message, Data: data})
	}
}

func (r *Researcher) checkpoint() {
	r.state.ElapsedMS = r.elapsedMS()
	r.trace("checkpoint", "checkpoint", r.state.Round, "", r.state)
	if r.onCheckpoint != nil {
		r.onCheckpoint(r.state)
	}
}

// elapsedMS includes time accumulated in previous runs of a resumed job.
func (r *Researcher) elapsedMS() int64 {
	return r.baseElapsed + time.Since(r.startTime).Milliseconds()
}

// State returns a copy of the current engine state.
func (r *Researcher) State() State { return r.state }

func (r *Researcher) addPrice(cost *float64) {
	if cost == nil {
		return
	}
	r.priceMu.Lock()
	defer r.priceMu.Unlock()
	if r.state.PriceUSD == nil {
		zero := 0.0
		r.state.PriceUSD = &zero
	}
	*r.state.PriceUSD += *cost
}

func (r *Researcher) knownCostUSD() float64 {
	r.priceMu.Lock()
	defer r.priceMu.Unlock()
	if r.state.PriceUSD == nil {
		return 0
	}
	return *r.state.PriceUSD
}

func (r *Researcher) finalWritingOperation(operation string) bool {
	if strings.HasSuffix(operation, "_continue") {
		return true
	}
	switch operation {
	case "final_report", "final_report_expand", "final_brainstorm", "outline", "outline_refine", "write_section", "write_glue":
		return true
	default:
		return false
	}
}

func continuationOverlap(text string, maxRunes int) string {
	runes := []rune(text)
	if len(runes) > maxRunes {
		runes = runes[len(runes)-maxRunes:]
	}
	return string(runes)
}

// mergeWritingContinuation removes only an exact suffix/prefix overlap. It
// never rewrites either segment, so citation IDs and already-written prose are
// preserved byte-for-byte.
func mergeWritingContinuation(base, continuation string) string {
	if base == "" {
		return continuation
	}
	if continuation == "" {
		return base
	}
	maxOverlap := len(base)
	if len(continuation) < maxOverlap {
		maxOverlap = len(continuation)
	}
	if maxOverlap > 4096 {
		maxOverlap = 4096
	}
	for overlap := maxOverlap; overlap >= 32; overlap-- {
		if base[len(base)-overlap:] == continuation[:overlap] {
			return base + continuation[overlap:]
		}
	}
	return strings.TrimRight(base, " \t\n") + "\n\n" + strings.TrimLeft(continuation, " \t\n")
}

func errorString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func (r *Researcher) continueWriting(ctx context.Context, operation string, round int, original []chatMsg, partial string, temperature float64, maxTokens int, timeout time.Duration, onDelta func(int, string)) (string, bool) {
	assembled := partial
	if strings.TrimSpace(assembled) == "" || r.cfg.MaxWritingContinuations <= 0 {
		return assembled, false
	}
	r.trace("writing_recovery_started", "writing", round, "", map[string]any{
		"operation": operation, "partial_characters": len(partial), "max_continuations": r.cfg.MaxWritingContinuations,
	})
	for attempt := 1; attempt <= r.cfg.MaxWritingContinuations; attempt++ {
		overlap := continuationOverlap(assembled, 240)
		instruction := fmt.Sprintf(`The previous response hit its output ceiling. Continue the SAME document; do not search, restart, summarize, or repeat completed sections.
Begin by repeating the required overlap text below exactly, then continue from that point. Preserve every citation ID exactly (for example [S3]). Finish all remaining sections without omitting any.

<required_overlap>%s</required_overlap>`, overlap)
		messages := append([]chatMsg(nil), original...)
		messages = append(messages, chatMsg{Role: "assistant", Content: assembled}, chatMsg{Role: "user", Content: instruction})
		continued, callID, err := r.llmCallStream(ctx, operation+"_continue", round, messages, temperature, maxTokens, timeout, onDelta)
		if continued != "" {
			assembled = mergeWritingContinuation(assembled, continued)
		}
		if err == nil && continued != "" {
			r.setCallDisposition(callID, "accepted")
			r.trace("writing_recovery_completed", "writing", round, "", map[string]any{
				"operation": operation, "continuations": attempt, "characters": len(assembled),
			})
			return assembled, true
		}
		if errors.Is(err, ErrOutputTruncated) && attempt < r.cfg.MaxWritingContinuations {
			r.setCallDisposition(callID, "retried")
			continue
		}
		r.setCallDisposition(callID, "rejected")
		break
	}
	r.trace("writing_recovery_exhausted", "warning", round, "bounded continuation recovery was exhausted", map[string]any{
		"operation": operation, "characters": len(assembled),
	})
	return assembled, false
}

func (r *Researcher) budgetFraction(finalWriting bool) float64 {
	if finalWriting {
		return 1
	}
	reserve := r.cfg.FinalReservePercent
	if reserve < 0 {
		reserve = 0
	}
	if reserve > 90 {
		reserve = 90
	}
	return float64(100-reserve) / 100
}

func (r *Researcher) budgetExhausted(finalWriting bool) (bool, string) {
	fraction := r.budgetFraction(finalWriting)
	if r.cfg.MaxTime > 0 && time.Duration(r.elapsedMS())*time.Millisecond >= time.Duration(float64(r.cfg.MaxTime)*fraction) {
		if finalWriting {
			return true, "time budget exhausted"
		}
		return true, "research time allocation reached — reserving the remainder for final writing"
	}
	if r.cfg.MaxCostUSD > 0 && r.knownCostUSD() >= r.cfg.MaxCostUSD*fraction {
		if finalWriting {
			return true, "cost budget exhausted"
		}
		return true, "research cost allocation reached — reserving the remainder for final writing"
	}
	return false, ""
}

func (r *Researcher) callContext(parent context.Context, operation string, requested time.Duration) (context.Context, context.CancelFunc, error) {
	finalWriting := r.finalWritingOperation(operation)
	if exhausted, message := r.budgetExhausted(finalWriting); exhausted {
		return nil, nil, fmt.Errorf("%w: %s", ErrBudgetExhausted, message)
	}
	limit := requested
	if r.cfg.MaxTime > 0 {
		available := time.Duration(float64(r.cfg.MaxTime)*r.budgetFraction(finalWriting)) - time.Duration(r.elapsedMS())*time.Millisecond
		if available <= 0 {
			return nil, nil, fmt.Errorf("%w: no time remains for %s", ErrBudgetExhausted, operation)
		}
		if limit <= 0 || available < limit {
			limit = available
		}
	}
	if limit > 0 {
		ctx, cancel := context.WithTimeout(parent, limit)
		return ctx, cancel, nil
	}
	ctx, cancel := context.WithCancel(parent)
	return ctx, cancel, nil
}

// Run executes the research loop and returns the formatted composite report.
func (r *Researcher) Run(ctx context.Context) (string, error) {
	r.startTime = time.Now()
	r.baseElapsed = r.state.ElapsedMS
	r.trace("run_started", "planning", r.state.Round, "", map[string]any{
		"model": r.writer.Model, "worker_model": r.worker.Model, "mode": r.cfg.Mode,
		"max_rounds": r.cfg.MaxRounds, "max_time_ms": r.cfg.MaxTime.Milliseconds(),
		"max_cost_usd": r.cfg.MaxCostUSD, "final_reserve_percent": r.cfg.FinalReservePercent,
		"token_limits": r.cfg.TokenLimits,
	})

	// Phases 1–2 only run before the first round (fresh job, or a job that
	// crashed before completing round 1 with no plan persisted).
	if r.state.Round == 0 && r.state.Slug == "" {
		r.state.Slug = r.generateSlug(ctx)
		if ctx.Err() != nil {
			return "", ctx.Err()
		}
	}
	if r.state.Round == 0 && r.state.Plan == "" {
		r.progress(Progress{Phase: "planning"})
		r.state.Plan = r.plan(ctx)
		if ctx.Err() != nil {
			return "", ctx.Err()
		}
		if r.state.Plan != "" {
			r.trace("plan_completed", "planning", 0, "", map[string]any{"plan": r.state.Plan})
			r.progress(Progress{Phase: "planning", Message: "plan ready — " + r.state.Plan})
		}
	}
	if r.state.Round == 0 && r.state.Category == "" {
		if r.brainstorm() {
			r.state.Category = r.classifyBrainstorm(ctx)
			r.trace("classification_completed", "planning", 0, "", map[string]any{"category": r.state.Category})
			r.progress(Progress{Phase: "planning", Message: "output format: " + r.state.Category})
		} else {
			r.state.Category = r.classify(ctx)
			r.trace("classification_completed", "planning", 0, "", map[string]any{"category": r.state.Category})
			r.progress(Progress{Phase: "planning", Message: "report category: " + r.state.Category})
		}
		if ctx.Err() != nil {
			return "", ctx.Err()
		}
		r.checkpoint()
	}

	continueRounds := true
	completedCreativity := 0
	if r.cfg.RedditResume != nil {
		var err error
		continueRounds, err = r.resumeRedditRound(ctx, *r.cfg.RedditResume)
		if err != nil {
			return "", err
		}
		completedCreativity = r.cfg.RedditResume.Pending.Creativity
	}

	for round := r.state.Round + 1; continueRounds && round <= r.cfg.MaxRounds; round++ {
		if ctx.Err() != nil {
			return "", ctx.Err()
		}
		if exhausted, message := r.budgetExhausted(false); exhausted {
			r.progress(Progress{Phase: "warning", Message: message})
			break
		}

		keepGoing, err := r.runOneRound(ctx, round, 0)
		if err != nil {
			return "", err
		}
		if !keepGoing {
			break
		}

		// No point running the stop-check on the final round — the loop exits
		// regardless. This keeps the low-effort modes (1–2 rounds) snappy.
		if round < r.cfg.MaxRounds && round >= r.cfg.MinRounds && r.shouldStop(ctx, round) {
			break
		}
		if ctx.Err() != nil {
			return "", ctx.Err()
		}
	}

	// If the main loop ran to its hard cap (rather than the model deciding the
	// report was complete or the run aborting), say so.
	if r.state.Round >= r.cfg.MaxRounds {
		r.progress(Progress{Phase: "note", Round: r.state.Round,
			Message: fmt.Sprintf("reached the maximum number of rounds (%d)", r.cfg.MaxRounds)})
	}

	// Bonus creative rounds (effort 4 → 1, effort 5 → 2). These run past the
	// normal stopping point, pushing the model to search from fresh angles —
	// only worth doing when we already have a report to extend.
	if r.cfg.ExtraRounds > 0 && len(r.state.Findings) > 0 {
		r.state.EmptyRounds = 0 // give the bonus rounds a fair chance
		for creativity := completedCreativity + 1; creativity <= r.cfg.ExtraRounds; creativity++ {
			if ctx.Err() != nil {
				return "", ctx.Err()
			}
			if exhausted, message := r.budgetExhausted(false); exhausted {
				r.progress(Progress{Phase: "warning", Message: message})
				break
			}
			round := r.state.Round + 1
			how := "more creatively"
			if creativity >= 2 {
				how = "even more creatively"
			}
			r.progress(Progress{Phase: "note", Round: round,
				Message: fmt.Sprintf("bonus round %d of %d — searching %s", creativity, r.cfg.ExtraRounds, how)})
			keepGoing, err := r.runOneRound(ctx, round, creativity)
			if err != nil {
				return "", err
			}
			if !keepGoing {
				break
			}
		}
	}

	if r.state.Report == "" {
		if len(r.state.Findings) == 0 {
			return "", fmt.Errorf("research produced no findings")
		}
		// Synthesis never succeeded — fall back to formatted raw findings.
		r.state.Report = "## Research Findings\n\nSynthesis was unavailable; the raw findings are listed below.\n\n" + r.formatFindings(r.state.Findings)
	}

	r.progress(Progress{Phase: "writing", TotalSources: len(r.state.AnalyzedURLs), TotalFindings: len(r.state.Findings)})
	var final string
	if r.cfg.DeepReport {
		final = r.deepReport(ctx)
	} else {
		final = r.finalReport(ctx)
	}
	if ctx.Err() != nil {
		return "", ctx.Err()
	}
	composite := r.formatCompositeReport(final)
	r.trace("final_report_completed", "writing", r.state.Round, "", map[string]any{"report": composite})
	return composite, nil
}

// RegenerateReport re-runs only the final-report phase over the researcher's
// already-loaded state (findings, plan, evolving report, category), honouring
// cfg.DeepReport. It re-uses the same writer path as a normal run's final step,
// so it can recover detail a previous single pass dropped — without searching
// the web again. The returned price covers only the calls this regeneration
// made; any price already on the loaded state is ignored.
func (r *Researcher) RegenerateReport(ctx context.Context) (string, *float64, error) {
	r.startTime = time.Now()
	r.baseElapsed = r.state.ElapsedMS
	r.state.PriceUSD = nil // report only the cost of this regeneration

	if r.state.Report == "" && len(r.state.Findings) == 0 {
		return "", nil, fmt.Errorf("job has no findings to regenerate a report from")
	}
	if r.state.Report == "" {
		r.state.Report = "## Research Findings\n\nSynthesis was unavailable; the raw findings are listed below.\n\n" + r.formatFindings(r.state.Findings)
	}

	r.progress(Progress{Phase: "writing", TotalSources: len(r.state.AnalyzedURLs), TotalFindings: len(r.state.Findings)})
	var final string
	if r.cfg.DeepReport {
		final = r.deepReport(ctx)
	} else {
		final = r.finalReport(ctx)
	}
	if ctx.Err() != nil {
		return "", nil, ctx.Err()
	}
	if strings.TrimSpace(final) == "" {
		return "", nil, fmt.Errorf("report regeneration produced no content")
	}
	composite := r.formatCompositeReport(final)
	r.trace("final_report_completed", "writing", r.state.Round, "", map[string]any{"report": composite, "regenerated": true})
	return composite, r.state.PriceUSD, nil
}

func (r *Researcher) resumeRedditRound(ctx context.Context, resume RedditResume) (bool, error) {
	pending := resume.Pending
	findings := r.extractAll(ctx, pending.Round, pending.OrdinaryURLs)

	// Imported, failed, and explicitly skipped threads are all considered
	// analyzed, preventing the same thread from prompting another handoff.
	for _, requested := range pending.Request.Pages {
		r.state.AnalyzedURLs = append(r.state.AnalyzedURLs, AnalyzedURL{URL: requested.URL, Title: requested.Title})
	}
	if !resume.Skipped {
		for _, page := range resume.Pages {
			if ctx.Err() != nil {
				return false, ctx.Err()
			}
			if page.Failure != "" {
				continue
			}
			r.progress(Progress{Phase: "reading", Round: pending.Round, URL: page.URL, Title: page.Title,
				TotalSources: len(r.state.AnalyzedURLs), TotalFindings: len(r.state.Findings)})
			if finding := r.ExtractText(ctx, pending.Round, page.URL, page.Title, page.Content); finding != nil {
				findings = append(findings, *finding)
			}
		}
	}
	if ctx.Err() != nil {
		return false, ctx.Err()
	}

	keepGoing := true
	if r.brainstorm() {
		r.state.Findings = append(r.state.Findings, findings...)
		r.progress(Progress{Phase: "analyzing", Round: pending.Round, TotalSources: len(r.state.AnalyzedURLs), TotalFindings: len(r.state.Findings)})
		if !r.synthesize(ctx, pending.Round, findings) {
			keepGoing = false
		}
	} else if len(findings) == 0 {
		r.state.EmptyRounds++
		if r.state.EmptyRounds >= r.cfg.MaxEmptyRounds {
			if len(r.state.Findings) == 0 {
				return false, fmt.Errorf("search returned no usable results after %d round(s) — check that SearXNG is running and reachable", pending.Round)
			}
			r.progress(Progress{Phase: "warning", Round: pending.Round, Message: "consecutive empty rounds — writing report with findings so far"})
			keepGoing = false
		}
	} else {
		r.state.EmptyRounds = 0
		r.state.Findings = append(r.state.Findings, findings...)
		r.progress(Progress{Phase: "analyzing", Round: pending.Round, TotalSources: len(r.state.AnalyzedURLs), TotalFindings: len(r.state.Findings)})
		if !r.synthesize(ctx, pending.Round, findings) {
			keepGoing = false
		}
	}
	if ctx.Err() != nil {
		return false, ctx.Err()
	}
	r.state.Round = pending.Round
	r.state.ElapsedMS = r.elapsedMS()
	if r.cfg.OnRedditRoundComplete != nil {
		if err := r.cfg.OnRedditRoundComplete(r.state); err != nil {
			return false, fmt.Errorf("checkpoint completed Reddit round: %w", err)
		}
	} else {
		r.checkpoint()
	}
	return keepGoing, nil
}

// runOneRound dispatches to the round implementation for the active mode.
func (r *Researcher) runOneRound(ctx context.Context, round, creativity int) (keepGoing bool, err error) {
	if r.brainstorm() {
		return r.runBrainstormRound(ctx, round, creativity)
	}
	return r.runRound(ctx, round, creativity)
}

// runBrainstormRound executes one ideation round: the model develops the
// design, deciding for itself whether it needs to search the web. When it asks
// for queries we search and extract as usual; when it doesn't, the round is
// pure ideation. Either way the design doc is developed before checkpointing.
// Unlike research mode, a round that finds nothing on the web is not a failure
// — the model can still make progress from its own knowledge — so empty rounds
// never abort the run.
func (r *Researcher) runBrainstormRound(ctx context.Context, round, creativity int) (keepGoing bool, err error) {
	queries := r.generateQueries(ctx, round, creativity)
	if ctx.Err() != nil {
		return false, nil
	}

	var findings []Finding
	if len(queries) > 0 {
		r.state.QueriesUsed = append(r.state.QueriesUsed, queries...)
		r.progress(Progress{Phase: "searching", Round: round, Queries: queries, TotalSources: len(r.state.AnalyzedURLs)})
		r.trace("queries_generated", "searching", round, "", map[string]any{"queries": queries, "creativity": creativity})
		newURLs := r.searchAll(ctx, round, queries)
		ordinaryURLs, err := r.pauseForReddit(round, creativity, queries, newURLs)
		if err != nil {
			return false, err
		}
		findings = r.extractAll(ctx, round, ordinaryURLs)
		if ctx.Err() != nil {
			return false, nil
		}
		if len(findings) > 0 {
			r.state.Findings = append(r.state.Findings, findings...)
		}
	} else {
		r.progress(Progress{Phase: "note", Round: round, Message: "developing ideas without web search this round"})
	}

	r.progress(Progress{Phase: "analyzing", Round: round, TotalSources: len(r.state.AnalyzedURLs), TotalFindings: len(r.state.Findings)})
	if !r.synthesize(ctx, round, findings) {
		return false, nil
	}
	if ctx.Err() != nil {
		return false, nil
	}

	r.state.Round = round
	r.checkpoint()
	return true, nil
}

// runRound executes one Think → Search → Extract → Synthesise round and
// checkpoints the result. creativity (0 = normal, 1 = creative, ≥2 = very
// creative) selects the query-generation instruction. It returns keepGoing =
// false when the loop should stop (no new queries, or too many empty rounds),
// and a non-nil error only on a fatal condition (no usable results at all).
func (r *Researcher) runRound(ctx context.Context, round, creativity int) (keepGoing bool, err error) {
	queries := r.generateQueries(ctx, round, creativity)
	if ctx.Err() != nil {
		return false, nil
	}
	if len(queries) == 0 {
		r.progress(Progress{Phase: "warning", Round: round, Message: "no new search queries generated — stopping"})
		return false, nil
	}
	r.state.QueriesUsed = append(r.state.QueriesUsed, queries...)
	r.progress(Progress{Phase: "searching", Round: round, Queries: queries, TotalSources: len(r.state.AnalyzedURLs)})

	r.trace("queries_generated", "searching", round, "", map[string]any{"queries": queries, "creativity": creativity})
	newURLs := r.searchAll(ctx, round, queries)
	ordinaryURLs, err := r.pauseForReddit(round, creativity, queries, newURLs)
	if err != nil {
		return false, err
	}
	findings := r.extractAll(ctx, round, ordinaryURLs)
	if ctx.Err() != nil {
		return false, nil
	}

	if len(findings) == 0 {
		r.state.EmptyRounds++
		if r.state.EmptyRounds >= r.cfg.MaxEmptyRounds {
			if len(r.state.Findings) == 0 {
				return false, fmt.Errorf("search returned no usable results after %d round(s) — check that SearXNG is running and reachable", round)
			}
			r.progress(Progress{Phase: "warning", Round: round, Message: "consecutive empty rounds — writing report with findings so far"})
			r.state.Round = round
			r.checkpoint()
			return false, nil
		}
	} else {
		r.state.EmptyRounds = 0
		r.state.Findings = append(r.state.Findings, findings...)
		r.progress(Progress{Phase: "analyzing", Round: round, TotalSources: len(r.state.AnalyzedURLs), TotalFindings: len(r.state.Findings)})
		if !r.synthesize(ctx, round, findings) {
			r.state.Round = round
			r.checkpoint()
			return false, nil
		}
		if ctx.Err() != nil {
			return false, nil
		}
	}

	r.state.Round = round
	r.checkpoint()
	return true, nil
}

// ── Phase 1: Plan ────────────────────────────────────────────

func (r *Researcher) generateSlug(ctx context.Context) string {
	source := strings.TrimSpace(r.cfg.SlugSource)
	if source == "" {
		source = r.cfg.Query
	}
	out, callID, err := r.llmCallWorker(ctx, "slug", 0, []chatMsg{{Role: "user", Content: fmt.Sprintf(slugPrompt, source)}}, 0.2, 64, planningTimeout)
	if err == nil {
		if slug := normalizeSlug(out); slug != "" {
			r.setCallDisposition(callID, "accepted")
			return slug
		}
	}
	r.setCallDisposition(callID, "fallback")
	if err != nil {
		r.progress(Progress{Phase: "warning", Message: "research name generation failed: " + err.Error()})
	}
	if slug := normalizeSlug(source); slug != "" {
		return slug
	}
	return "research_job"
}

func normalizeSlug(value string) string {
	value = strings.ToLower(stripToolCalls(value))
	var b strings.Builder
	underscore := false
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			if underscore && b.Len() > 0 {
				b.WriteByte('_')
			}
			underscore = false
			if b.Len() < maxSlugLength {
				b.WriteRune(r)
			}
		} else {
			underscore = true
		}
		if b.Len() >= maxSlugLength {
			break
		}
	}
	return strings.TrimRight(b.String(), "_")
}

func (r *Researcher) plan(ctx context.Context) string {
	if r.brainstorm() {
		prompt := currentDateContext(r.cfg.Location) + fmt.Sprintf(brainstormPlanPrompt, r.cfg.Query)
		out, callID, err := r.llmCall(ctx, "plan", 0, []chatMsg{{Role: "user", Content: prompt}}, 0.6, 1024, planningTimeout)
		if err != nil {
			r.setCallDisposition(callID, "rejected")
			r.progress(Progress{Phase: "warning", Message: "planning failed: " + err.Error()})
			return ""
		}
		r.setCallDisposition(callID, "accepted")
		return stripToolCalls(out)
	}
	prompt := currentDateContext(r.cfg.Location) + fmt.Sprintf(researchPlanPrompt, r.cfg.Query)
	out, callID, err := r.llmCall(ctx, "plan", 0, []chatMsg{{Role: "user", Content: prompt}}, 0.3, 1024, planningTimeout)
	if err != nil {
		r.setCallDisposition(callID, "rejected")
		r.progress(Progress{Phase: "warning", Message: "planning failed: " + err.Error()})
		return ""
	}
	var plan struct {
		SubQuestions    []string `json:"sub_questions"`
		KeyTopics       []string `json:"key_topics"`
		SuccessCriteria string   `json:"success_criteria"`
	}
	if parseJSONObject(out, &plan) != nil {
		// JSON parsing failed — use the raw text as the plan.
		r.setCallDisposition(callID, "fallback")
		return stripToolCalls(out)
	}
	r.setCallDisposition(callID, "accepted")
	return fmt.Sprintf("Sub-questions: %s\nKey topics: %s\nSuccess: %s",
		strings.Join(plan.SubQuestions, "; "), strings.Join(plan.KeyTopics, ", "), plan.SuccessCriteria)
}

// ── Phase 2: Classify ────────────────────────────────────────

// classify returns the report category, or "general" when none fits. The
// "general" value also marks the job as classified so a resumed run does not
// repeat this phase.
func (r *Researcher) classify(ctx context.Context) string {
	out, callID, err := r.llmCallWorker(ctx, "classify", 0, []chatMsg{{Role: "user", Content: fmt.Sprintf(classifyPrompt, r.cfg.Query)}}, 0, 20, classifyTimeout)
	if err != nil {
		r.setCallDisposition(callID, "fallback")
		return "general"
	}
	first := strings.ToLower(strings.Trim(strings.Fields(out + " x")[0], ".,!:;\"'"))
	if validCategories[first] {
		r.setCallDisposition(callID, "accepted")
		return first
	}
	for _, word := range strings.Fields(strings.ToLower(out)) {
		if validCategories[strings.Trim(word, ".,!:;\"'")] {
			r.setCallDisposition(callID, "accepted")
			return strings.Trim(word, ".,!:;\"'")
		}
	}
	r.setCallDisposition(callID, "fallback")
	return "general"
}

// classifyBrainstorm returns the output format for a brainstorm run, defaulting
// to "design-doc" when no other format fits. Mirrors classify for research mode.
func (r *Researcher) classifyBrainstorm(ctx context.Context) string {
	out, callID, err := r.llmCallWorker(ctx, "brainstorm_classify", 0, []chatMsg{{Role: "user", Content: fmt.Sprintf(brainstormClassifyPrompt, r.cfg.Query)}}, 0, 20, classifyTimeout)
	if err != nil {
		r.setCallDisposition(callID, "fallback")
		return "design-doc"
	}
	first := strings.ToLower(strings.Trim(strings.Fields(out + " x")[0], ".,!:;\"'"))
	if validBrainstormFormats[first] {
		r.setCallDisposition(callID, "accepted")
		return first
	}
	for _, word := range strings.Fields(strings.ToLower(out)) {
		w := strings.Trim(word, ".,!:;\"'")
		if validBrainstormFormats[w] {
			r.setCallDisposition(callID, "accepted")
			return w
		}
	}
	r.setCallDisposition(callID, "fallback")
	return "design-doc"
}

// ── Phase 3: Think (query generation) ────────────────────────

func (r *Researcher) generateQueries(ctx context.Context, round, creativity int) []string {
	report := r.state.Report

	// forceSearch makes the first brainstorm round mandatory-search so a run with
	// the toggle on never ends up as pure ideation. Bonus creative rounds (and
	// every round after the first) keep the model's own discretion.
	forceSearch := r.brainstorm() && r.cfg.ForceSearch && creativity == 0 && round == 1

	var prompt string
	if r.brainstorm() {
		instruction := brainstormSearchInstruction
		switch {
		case creativity == 1:
			instruction = brainstormCreativeInstruction
		case creativity >= 2:
			instruction = brainstormVeryCreativeInstruction
		}
		if report == "" {
			report = "(No ideas developed yet.)"
		}
		queryPrompt := brainstormQueryPrompt
		if forceSearch {
			queryPrompt = brainstormForcedQueryPrompt
			instruction = brainstormForcedSearchInstruction
		}
		prompt = currentDateContext(r.cfg.Location) +
			fmt.Sprintf(queryPrompt, r.cfg.Query, r.state.Plan, report, round, instruction)
	} else {
		numQueries, instruction := 4, queryGenFirstRoundInstruction
		if round > 1 {
			numQueries, instruction = 3, queryGenFollowUpInstruction
		}
		switch {
		case creativity == 1:
			instruction = queryGenCreativeInstruction
		case creativity >= 2:
			instruction = queryGenVeryCreativeInstruction
		}
		if report == "" {
			report = "(No findings yet.)"
		}
		prompt = currentDateContext(r.cfg.Location) +
			fmt.Sprintf(queryGenPrompt, r.cfg.Query, r.state.Plan, report, round, numQueries, instruction)
	}

	out, callID, err := r.llmCallWorker(ctx, "query_generation", round, []chatMsg{{Role: "user", Content: prompt}}, 0.5, 4096, queryTimeout)
	if err != nil {
		r.setCallDisposition(callID, "rejected")
		r.progress(Progress{Phase: "warning", Round: round, Message: "query generation failed: " + err.Error()})
		return nil
	}

	used := make(map[string]bool, len(r.state.QueriesUsed))
	for _, q := range r.state.QueriesUsed {
		used[strings.ToLower(q)] = true
	}
	var queries []string
	for _, q := range parseJSONStringArray(out) {
		q = strings.TrimSpace(q)
		if q == "" || used[strings.ToLower(q)] {
			continue
		}
		used[strings.ToLower(q)] = true
		queries = append(queries, q)
	}
	// With the toggle on, the model occasionally still returns no usable query;
	// fall back to the brief itself so the promised search always happens.
	usedFallback := false
	if forceSearch && len(queries) == 0 {
		if q := strings.TrimSpace(r.cfg.Query); q != "" && !used[strings.ToLower(q)] {
			queries = append(queries, q)
			usedFallback = true
		}
	}
	if usedFallback {
		r.setCallDisposition(callID, "fallback")
	} else if len(queries) == 0 {
		r.setCallDisposition(callID, "rejected")
	} else {
		r.setCallDisposition(callID, "accepted")
	}
	return queries
}

// ── Phase 4: Search ──────────────────────────────────────────

// searchAll runs all queries in parallel and returns new (unseen) URLs,
// capped at MaxURLsPerRound × len(queries).
func (r *Researcher) searchAll(ctx context.Context, round int, queries []string) []AnalyzedURL {
	results := make([][]searx.Result, len(queries))
	var wg sync.WaitGroup
	for i, q := range queries {
		wg.Add(1)
		go func(i int, q string) {
			defer wg.Done()
			res, err := searx.Search(ctx, r.cfg.SearXNGURL, q, 1)
			if err != nil {
				r.progress(Progress{Phase: "warning", Round: round, Message: fmt.Sprintf("search failed for %q: %v", q, err)})
				return
			}
			if len(res) > 10 {
				res = res[:10]
			}
			results[i] = res
		}(i, q)
	}
	wg.Wait()
	for i, query := range queries {
		r.trace("search_results", "searching", round, "", map[string]any{"query": query, "results": results[i]})
	}

	seen := make(map[string]bool, len(r.state.AnalyzedURLs))
	for _, u := range r.state.AnalyzedURLs {
		seen[u.URL] = true
	}
	limit := r.cfg.MaxURLsPerRound * len(queries)
	var newURLs []AnalyzedURL
	var rejected []map[string]string
	for _, res := range results {
		for _, sr := range res {
			if len(newURLs) >= limit {
				rejected = append(rejected, map[string]string{"url": sr.URL, "title": sr.Title, "reason": "round_limit"})
				continue
			}
			if sr.URL == "" {
				rejected = append(rejected, map[string]string{"url": sr.URL, "title": sr.Title, "reason": "empty_url"})
				continue
			}
			if seen[sr.URL] {
				rejected = append(rejected, map[string]string{"url": sr.URL, "title": sr.Title, "reason": "duplicate"})
				continue
			}
			seen[sr.URL] = true
			newURLs = append(newURLs, AnalyzedURL{URL: sr.URL, Title: sr.Title})
		}
	}
	r.trace("urls_selected", "searching", round, "", map[string]any{"selected": newURLs, "rejected": rejected, "limit": limit})
	return newURLs
}

func (r *Researcher) pauseForReddit(round, creativity int, queries []string, urls []AnalyzedURL) ([]AnalyzedURL, error) {
	if !r.cfg.PauseRedditImport {
		return urls, nil
	}
	ordinary, redditPages := SplitRedditURLs(urls, r.state.AnalyzedURLs)
	if len(redditPages) == 0 {
		return ordinary, nil
	}
	if r.cfg.OnRedditPause == nil {
		return nil, errors.New("Reddit import pause callback is not configured")
	}
	requestIDBytes := make([]byte, 16)
	if _, err := rand.Read(requestIDBytes); err != nil {
		return nil, fmt.Errorf("create Reddit request ID: %w", err)
	}
	req, err := redditimport.NewRequest(fmt.Sprintf("%x", requestIDBytes), redditPages, redditimport.CaptureLimits{})
	if err != nil {
		return nil, err
	}
	redditimport.SetRequestName(&req, fmt.Sprintf("%s_%d", r.state.Slug, round))
	r.checkpoint()
	pending := PendingRedditRound{
		Round: round, Creativity: creativity, Queries: append([]string(nil), queries...),
		OrdinaryURLs: ordinary, Request: req, ElapsedMS: r.state.ElapsedMS,
	}
	if err := r.cfg.OnRedditPause(pending); err != nil {
		return nil, fmt.Errorf("checkpoint Reddit import request: %w", err)
	}
	return nil, ErrAwaitingReddit
}

// SplitRedditURLs canonicalizes and groups Reddit search results while leaving
// ordinary results in search order. Previously analyzed Reddit threads are
// discarded so they cannot trigger another browser handoff.
func SplitRedditURLs(urls, analyzed []AnalyzedURL) ([]AnalyzedURL, []redditimport.RequestedPage) {
	seenThreads := make(map[string]bool)
	for _, item := range analyzed {
		if _, threadURL, err := redditimport.CanonicalizeURL(item.URL); err == nil {
			seenThreads[threadURL] = true
		}
	}
	ordinary := make([]AnalyzedURL, 0, len(urls))
	redditPages := make([]redditimport.RequestedPage, 0)
	for _, item := range urls {
		canonical, threadURL, err := redditimport.CanonicalizeURL(item.URL)
		if err != nil {
			ordinary = append(ordinary, item)
			continue
		}
		if seenThreads[threadURL] {
			continue
		}
		seenThreads[threadURL] = true
		redditPages = append(redditPages, redditimport.RequestedPage{URL: canonical, Title: item.Title})
	}
	return ordinary, redditPages
}

// ── Phase 5: Extract ─────────────────────────────────────────

// extractAll fetches and extracts all URLs concurrently, bounded by
// ExtractionConcurrency. Every attempted URL is recorded in AnalyzedURLs.
func (r *Researcher) extractAll(ctx context.Context, round int, urls []AnalyzedURL) []Finding {
	r.state.AnalyzedURLs = append(r.state.AnalyzedURLs, urls...)

	sem := make(chan struct{}, r.cfg.ExtractionConcurrency)
	findings := make([]*Finding, len(urls))
	var wg sync.WaitGroup
	for i, u := range urls {
		wg.Add(1)
		go func(i int, u AnalyzedURL) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			if ctx.Err() != nil {
				return
			}
			r.progress(Progress{Phase: "reading", Round: round, URL: u.URL, Title: u.Title,
				TotalSources: len(r.state.AnalyzedURLs), TotalFindings: len(r.state.Findings)})
			findings[i] = r.fetchAndExtract(ctx, round, u.URL, u.Title)
		}(i, u)
	}
	wg.Wait()

	var out []Finding
	for _, f := range findings {
		if f != nil {
			out = append(out, *f)
		}
	}
	r.trace("extraction_batch_completed", "reading", round, "", map[string]any{"attempted": len(urls), "findings": out})
	return out
}

// ── Phase 6: Synthesise ──────────────────────────────────────

func (r *Researcher) synthesize(ctx context.Context, round int, newFindings []Finding) bool {
	if len(newFindings) > r.cfg.SynthesisWindow {
		newFindings = newFindings[len(newFindings)-r.cfg.SynthesisWindow:]
	}
	report := r.state.Report
	if report == "" {
		report = "(First round — no report yet.)"
	}
	var prompt string
	if r.brainstorm() {
		findingsText := r.formatFindings(newFindings)
		if findingsText == "" {
			findingsText = "(No web research this round — develop the ideas from your own knowledge.)"
		}
		prompt = fmt.Sprintf(brainstormDevelopPrompt, r.cfg.Query, r.state.Plan, report, findingsText)
	} else {
		prompt = fmt.Sprintf(synthesizePrompt, r.cfg.Query, report, r.formatFindings(newFindings))
	}
	prompt += fmt.Sprintf("\n\nResearch-memory limit: keep this complete working synthesis within %d output tokens. Compress older detail before dropping claims, evidence gaps, contradictions, calculations, or citation IDs.", r.cfg.TokenLimits.Memory)
	out, callID, err := r.llmCallStream(ctx, "synthesize", round, []chatMsg{{Role: "user", Content: prompt}}, 0.3, r.cfg.TokenLimits.Synthesis, synthesisTimeout,
		func(generated int, tail string) {
			r.progress(Progress{Phase: "analyzing", Round: round, TotalFindings: len(r.state.Findings), Generated: generated, Snippet: tail})
		})
	if errors.Is(err, ErrOutputTruncated) && out != "" {
		r.setCallDisposition(callID, "retried")
		r.progress(Progress{Phase: "warning", Round: round, Message: "synthesis hit its output ceiling — compacting the same research memory without searching again"})
		r.trace("writing_recovery_started", "analyzing", round, "", map[string]any{"operation": "synthesize", "strategy": "compact", "memory_token_target": r.cfg.TokenLimits.Memory})
		compactPrompt := fmt.Sprintf(`The research-memory synthesis below was cut off by an output limit. Rewrite it as one COMPLETE compact working synthesis within %d output tokens.
Do not search or add unsupported facts. Preserve every citation ID exactly, plus material claims, contradictions, assumptions, calculations, and unresolved evidence gaps. Compress wording and older background first.

Research question:
%s

Previous complete memory:
%s

Truncated attempted update:
%s`, r.cfg.TokenLimits.Memory, r.cfg.Query, report, out)
		compacted, compactCallID, compactErr := r.llmCallStream(ctx, "synthesize_compact", round, []chatMsg{{Role: "user", Content: compactPrompt}}, 0.2, r.cfg.TokenLimits.Memory, synthesisTimeout,
			func(generated int, tail string) {
				r.progress(Progress{Phase: "analyzing", Round: round, TotalFindings: len(r.state.Findings), Generated: generated, Snippet: tail})
			})
		if compactErr == nil && compacted != "" {
			r.setCallDisposition(compactCallID, "accepted")
			r.state.Report = compacted
			r.trace("writing_recovery_completed", "analyzing", round, "", map[string]any{"operation": "synthesize", "strategy": "compact", "memory_token_target": r.cfg.TokenLimits.Memory})
			r.trace("synthesis_completed", "analyzing", round, "", map[string]any{"report": compacted, "new_findings": len(newFindings), "recovered": true, "memory_token_target": r.cfg.TokenLimits.Memory})
			return true
		}
		r.setCallDisposition(compactCallID, "rejected")
		r.trace("writing_recovery_exhausted", "warning", round, "synthesis compaction failed; stopping research rounds", map[string]any{"operation": "synthesize", "error": errorString(compactErr)})
		r.progress(Progress{Phase: "warning", Round: round, Message: "synthesis recovery failed — stopping research and writing from evidence already collected"})
		return false
	}
	if err != nil || out == "" {
		r.setCallDisposition(callID, "rejected")
		// Keep the current report unchanged.
		r.progress(Progress{Phase: "warning", Message: "synthesis failed — keeping previous report"})
		return !errors.Is(err, ErrBudgetExhausted)
	}
	r.setCallDisposition(callID, "accepted")
	r.state.Report = out
	r.trace("synthesis_completed", "analyzing", round, "", map[string]any{"report": out, "new_findings": len(newFindings), "memory_token_target": r.cfg.TokenLimits.Memory})
	return true
}

// ── Phase 7: Decide ──────────────────────────────────────────

func (r *Researcher) shouldStop(ctx context.Context, round int) bool {
	stop := stopPrompt
	if r.brainstorm() {
		stop = brainstormStopPrompt
	}
	prompt := fmt.Sprintf(stop, r.cfg.Query, r.state.Report, round, r.cfg.MaxRounds)
	out, callID, err := r.llmCallWorker(ctx, "stop_decision", round, []chatMsg{{Role: "user", Content: prompt}}, 0.1, 128, stopTimeout)
	if err != nil {
		r.setCallDisposition(callID, "fallback")
		return false
	}
	r.setCallDisposition(callID, "accepted")
	decision := strings.TrimLeft(stripToolCalls(out), "*_`\"'>#- \t\n")
	shouldStop := strings.HasPrefix(strings.ToUpper(decision), "YES")
	r.trace("stop_decision", "deciding", round, decision, map[string]any{"stop": shouldStop, "response": out})
	r.progress(Progress{Phase: "deciding", Round: round, Message: decision})
	return shouldStop
}

// ── Phase 8: Final report ────────────────────────────────────

// reportInstructionSuffix returns the user's optional extra report instruction
// formatted for appending to a writing prompt, or "" when none was given.
func (r *Researcher) reportInstructionSuffix() string {
	instruction := strings.TrimSpace(r.cfg.ReportInstruction)
	if instruction == "" {
		return ""
	}
	return "\n\nAdditional instruction from the user — follow it while writing this report:\n" + instruction
}

func (r *Researcher) finalReport(ctx context.Context) string {
	if r.brainstorm() {
		return r.finalBrainstorm(ctx)
	}
	findings := r.formatFindings(r.state.Findings)
	if findings == "" {
		findings = "(No raw web findings were collected.)"
	}
	prompt := fmt.Sprintf(finalReportPrompt, r.cfg.Query, r.state.Report, findings)
	if override, ok := categoryPrompts[r.state.Category]; ok {
		prompt += override
	}
	prompt += r.reportInstructionSuffix()
	onDelta := func(generated int, tail string) {
		r.progress(Progress{Phase: "writing", TotalFindings: len(r.state.Findings), Generated: generated, Snippet: tail})
	}
	messages := []chatMsg{{Role: "user", Content: prompt}}
	report, callID, err := r.llmCallStream(ctx, "final_report", r.state.Round, messages, 0.3, r.cfg.TokenLimits.FinalReport, synthesisTimeout, onDelta)
	continuedReport := false
	if errors.Is(err, ErrOutputTruncated) && report != "" {
		r.setCallDisposition(callID, "retried")
		recovered, complete := r.continueWriting(ctx, "final_report", r.state.Round, messages, report, 0.3, r.cfg.TokenLimits.FinalReport, synthesisTimeout, onDelta)
		if complete {
			report, err = recovered, nil
			continuedReport = true
		} else {
			r.setCallDisposition(callID, "rejected")
		}
	}
	if err != nil || report == "" {
		r.setCallDisposition(callID, "fallback")
		// Never return empty — fall back to the evolving synthesis.
		return r.state.Report
	}
	// Minimum-length retry: ask the model to expand reports under 400 words.
	if len(strings.Fields(report)) < 400 {
		r.setCallDisposition(callID, "retried")
		r.progress(Progress{Phase: "warning", Message: fmt.Sprintf("report is short (%d words) — asking the model to expand it", len(strings.Fields(report)))})
		expandMessages := []chatMsg{
			{Role: "user", Content: prompt},
			{Role: "assistant", Content: report},
			{Role: "user", Content: expandReportPrompt},
		}
		expanded, expandCallID, err := r.llmCallStream(ctx, "final_report_expand", r.state.Round, expandMessages, 0.4, r.cfg.TokenLimits.FinalReport, synthesisTimeout, onDelta)
		if errors.Is(err, ErrOutputTruncated) && expanded != "" {
			r.setCallDisposition(expandCallID, "retried")
			var complete bool
			expanded, complete = r.continueWriting(ctx, "final_report_expand", r.state.Round, expandMessages, expanded, 0.4, r.cfg.TokenLimits.FinalReport, synthesisTimeout, onDelta)
			if complete {
				err = nil
			}
		}
		if err == nil && len(expanded) > len(report) {
			r.setCallDisposition(expandCallID, "accepted")
			report = expanded
		} else {
			r.setCallDisposition(expandCallID, "rejected")
		}
	} else if !continuedReport {
		r.setCallDisposition(callID, "accepted")
	}
	return report
}

// finalBrainstorm writes the polished design document for a brainstorm run.
// Unlike finalReport it imposes no report-category overrides and no minimum
// length — the output is a structured design write-up, not a long-form article.
func (r *Researcher) finalBrainstorm(ctx context.Context) string {
	findings := r.formatFindings(r.state.Findings)
	if findings == "" {
		findings = "(No supporting web findings were collected.)"
	}
	prompt := fmt.Sprintf(brainstormFinalPrompt, r.cfg.Query, r.state.Plan, r.state.Report, findings)
	if override, ok := brainstormFormatOverrides[r.state.Category]; ok {
		prompt += override
	}
	prompt += r.reportInstructionSuffix()
	onDelta := func(generated int, tail string) {
		r.progress(Progress{Phase: "writing", TotalFindings: len(r.state.Findings), Generated: generated, Snippet: tail})
	}
	messages := []chatMsg{{Role: "user", Content: prompt}}
	report, callID, err := r.llmCallStream(ctx, "final_brainstorm", r.state.Round, messages, 0.5, r.cfg.TokenLimits.FinalReport, synthesisTimeout, onDelta)
	continuedReport := false
	if errors.Is(err, ErrOutputTruncated) && report != "" {
		r.setCallDisposition(callID, "retried")
		recovered, complete := r.continueWriting(ctx, "final_brainstorm", r.state.Round, messages, report, 0.5, r.cfg.TokenLimits.FinalReport, synthesisTimeout, onDelta)
		if complete {
			report, err = recovered, nil
			continuedReport = true
		} else {
			r.setCallDisposition(callID, "rejected")
		}
	}
	if err != nil || report == "" {
		r.setCallDisposition(callID, "fallback")
		// Never return empty — fall back to the evolving design doc.
		return r.state.Report
	}
	if !continuedReport {
		r.setCallDisposition(callID, "accepted")
	}
	return report
}

// ── Phase 8b: Section-based deep report ──────────────────────
//
// deepReport replaces the single-shot final write with a multi-call pipeline:
// outline (draft → self-critique refine) → write each section in depth from the
// raw findings → glue on an executive summary and conclusion, assembled
// deterministically (no full-document rewrite, which would re-compress the
// detail this mode exists to preserve). Any stage failing falls back to the
// standard single-shot writer so the run never returns empty.

// reportSection is one planned section of the report/design document.
type reportSection struct {
	Title  string `json:"title"`
	Intent string `json:"intent"`
}

func (r *Researcher) deepReport(ctx context.Context) string {
	r.progress(Progress{Phase: "writing", Message: "planning report structure"})
	sections := r.outline(ctx)
	if ctx.Err() != nil {
		return r.state.Report
	}
	if len(sections) == 0 {
		r.progress(Progress{Phase: "warning", Message: "outline generation failed — writing a standard report"})
		return r.finalReport(ctx)
	}

	titles := make([]string, len(sections))
	for i, s := range sections {
		titles[i] = s.Title
	}
	outline := strings.Join(titles, "\n")
	findings := r.formatFindings(r.state.Findings)
	report := r.state.Report

	var bodies []string
	for i, sec := range sections {
		if ctx.Err() != nil {
			break
		}
		r.progress(Progress{Phase: "writing", TotalFindings: len(r.state.Findings),
			Message: fmt.Sprintf("writing section %d of %d — %s", i+1, len(sections), sec.Title)})
		if body := r.writeSection(ctx, sec, outline, findings, report); body != "" {
			bodies = append(bodies, body)
		}
	}
	if len(bodies) == 0 {
		return r.finalReport(ctx)
	}

	// Glue: a short executive summary up top and a closing section, generated
	// from the outline + condensed overview (cheap, robust plain-text calls).
	r.progress(Progress{Phase: "writing", TotalFindings: len(r.state.Findings), Message: "writing summary and conclusion"})
	summary := r.gluePart(ctx, outline, report,
		"a concise executive summary (2-4 sentences) capturing the most important takeaways.")
	conclHeading, conclInstruction := "## Conclusion",
		"a conclusion that ties the findings together and directly answers the question."
	if r.brainstorm() {
		parts := brainstormFormatConclusion[r.state.Category]
		if parts[0] == "" {
			parts = brainstormFormatConclusion["design-doc"]
		}
		conclHeading, conclInstruction = parts[0], parts[1]
	}
	conclusion := r.gluePart(ctx, outline, report, conclInstruction)

	var sb strings.Builder
	if summary != "" {
		sb.WriteString("## Executive summary\n\n")
		sb.WriteString(summary)
		sb.WriteString("\n\n")
	}
	sb.WriteString(strings.Join(bodies, "\n\n"))
	if conclusion != "" {
		sb.WriteString("\n\n")
		sb.WriteString(conclHeading)
		sb.WriteString("\n\n")
		sb.WriteString(conclusion)
	}
	return sb.String()
}

// outline produces the section plan: a draft outline followed by a self-critique
// refine pass that catches gaps and prunes redundancy. Returns nil on failure.
func (r *Researcher) outline(ctx context.Context) []reportSection {
	findings := r.formatFindings(r.state.Findings)
	if findings == "" {
		findings = "(No web findings — work from the notes and the brief.)"
	}
	report := r.state.Report
	if report == "" {
		report = "(No evolving report yet.)"
	}

	var draftPrompt string
	if r.brainstorm() {
		draftPrompt = fmt.Sprintf(brainstormOutlineDraftPrompt, r.cfg.Query, r.state.Plan, report, findings)
		if hint, ok := brainstormFormatOverrides[r.state.Category]; ok {
			draftPrompt += hint
		}
	} else {
		catHint := ""
		if r.state.Category != "" && r.state.Category != "general" {
			catHint = fmt.Sprintf(" This is a %s-type report, so shape the sections accordingly.", r.state.Category)
		}
		draftPrompt = fmt.Sprintf(outlineDraftPrompt, r.cfg.Query, r.state.Plan, report, findings, catHint)
	}
	draftPrompt += r.reportInstructionSuffix()
	draft := r.parseOutline(ctx, "outline", draftPrompt)
	if ctx.Err() != nil || len(draft) == 0 {
		return draft
	}

	var refinePrompt string
	if r.brainstorm() {
		refinePrompt = fmt.Sprintf(brainstormOutlineRefinePrompt, r.cfg.Query, r.state.Plan, findings, marshalOutline(draft))
	} else {
		refinePrompt = fmt.Sprintf(outlineRefinePrompt, r.cfg.Query, r.state.Plan, findings, marshalOutline(draft))
	}
	if refined := r.parseOutline(ctx, "outline_refine", refinePrompt); len(refined) > 0 {
		return refined
	}
	return draft // refine failed — the draft is still usable
}

// parseOutline runs one outline LLM call and parses {"sections": [...]}.
func (r *Researcher) parseOutline(ctx context.Context, operation, prompt string) []reportSection {
	out, callID, err := r.llmCall(ctx, operation, r.state.Round, []chatMsg{{Role: "user", Content: prompt}}, 0.3, 1024, planningTimeout)
	if err != nil {
		r.setCallDisposition(callID, "rejected")
		return nil
	}
	var parsed struct {
		Sections []reportSection `json:"sections"`
	}
	if parseJSONObject(out, &parsed) != nil {
		r.setCallDisposition(callID, "rejected")
		return nil
	}
	var sections []reportSection
	for _, s := range parsed.Sections {
		s.Title = strings.TrimSpace(s.Title)
		if s.Title != "" {
			sections = append(sections, s)
		}
	}
	if len(sections) == 0 {
		r.setCallDisposition(callID, "rejected")
	} else {
		r.setCallDisposition(callID, "accepted")
	}
	return sections
}

// marshalOutline serialises an outline for the refine prompt.
func marshalOutline(sections []reportSection) string {
	b, _ := json.Marshal(struct {
		Sections []reportSection `json:"sections"`
	}{sections})
	return string(b)
}

// writeSection writes one section in depth from the raw findings. Returns "" on
// failure so the caller can skip it. A "##" heading is ensured.
func (r *Researcher) writeSection(ctx context.Context, sec reportSection, outline, findings, report string) string {
	if findings == "" {
		findings = "(No web findings — develop from the notes and your own knowledge.)"
	}
	if report == "" {
		report = "(No evolving report yet.)"
	}
	var prompt string
	if r.brainstorm() {
		prompt = fmt.Sprintf(brainstormSectionWritePrompt, r.cfg.Query, r.state.Plan, sec.Title, sec.Intent, outline, report, findings)
	} else {
		prompt = fmt.Sprintf(sectionWritePrompt, r.cfg.Query, r.state.Plan, sec.Title, sec.Intent, outline, report, findings)
	}
	prompt += r.reportInstructionSuffix()
	onDelta := func(generated int, tail string) {
		r.progress(Progress{Phase: "writing", TotalFindings: len(r.state.Findings), Generated: generated, Snippet: tail})
	}
	messages := []chatMsg{{Role: "user", Content: prompt}}
	out, callID, err := r.llmCallStream(ctx, "write_section", r.state.Round, messages, 0.4, r.cfg.TokenLimits.Section, synthesisTimeout, onDelta)
	continuedSection := false
	if errors.Is(err, ErrOutputTruncated) && out != "" {
		r.setCallDisposition(callID, "retried")
		recovered, complete := r.continueWriting(ctx, "write_section", r.state.Round, messages, out, 0.4, r.cfg.TokenLimits.Section, synthesisTimeout, onDelta)
		if complete {
			out, err = recovered, nil
			continuedSection = true
		} else {
			r.setCallDisposition(callID, "rejected")
		}
	}
	if err != nil || strings.TrimSpace(out) == "" {
		r.setCallDisposition(callID, "rejected")
		return ""
	}
	if !continuedSection {
		r.setCallDisposition(callID, "accepted")
	}
	out = strings.TrimSpace(out)
	if !strings.HasPrefix(out, "#") {
		out = "## " + sec.Title + "\n\n" + out
	}
	return out
}

// gluePart generates a short standalone piece (executive summary or conclusion)
// from the outline and condensed overview. Plain text, no heading.
func (r *Researcher) gluePart(ctx context.Context, outline, report, instruction string) string {
	if report == "" {
		report = "(No overview available.)"
	}
	tmpl := reportGluePartPrompt
	if r.brainstorm() {
		tmpl = brainstormGluePartPrompt
	}
	prompt := fmt.Sprintf(tmpl, r.cfg.Query, outline, report, instruction)
	out, callID, err := r.llmCall(ctx, "write_glue", r.state.Round, []chatMsg{{Role: "user", Content: prompt}}, 0.3, 1500, synthesisTimeout)
	if err != nil {
		r.setCallDisposition(callID, "rejected")
		return ""
	}
	r.setCallDisposition(callID, "accepted")
	return stripToolCalls(out)
}

// ── Report formatting ────────────────────────────────────────

// sourceID returns the stable citation ID for a finding within the whole run.
// Matching by URL keeps IDs stable when a round formats only its new findings.
func (r *Researcher) sourceID(f Finding) string {
	for i, existing := range r.state.Findings {
		if existing.URL == f.URL {
			return fmt.Sprintf("S%d", i+1)
		}
	}
	return "S?"
}

// formatFindings serialises findings for LLM prompts. Each source is labelled
// with a stable ID so the model can cite [S1] instead of preserving long URLs
// through multiple summarisation passes.
func (r *Researcher) formatFindings(findings []Finding) string {
	var sb strings.Builder
	for i, f := range findings {
		text := f.Summary
		if text == "" {
			text = f.Evidence
			if len(text) > 1000 {
				text = text[:1000]
			}
		}
		id := r.sourceID(f)
		if id == "S?" {
			id = fmt.Sprintf("S%d", i+1)
		}
		title := f.Title
		if title == "" {
			title = f.URL
		}
		fmt.Fprintf(&sb, "[%s] %s\nURL: %s\nSummary: %s\n\n", id, title, f.URL, text)
	}
	return strings.TrimSpace(sb.String())
}

// normalizeSourceCitations converts model output like "[S1, S2]" or
// "[S1 and S2]" into separate markdown reference links. Shortcut references
// only work one at a time, even though models often group citations together.
func normalizeSourceCitations(text string) string {
	return reCombinedSourceCitation.ReplaceAllStringFunc(text, func(match string) string {
		inner := strings.Trim(match, "[]")
		parts := strings.FieldsFunc(inner, func(r rune) bool {
			return r == ',' || r == ';' || r == ' ' || r == '\t' || r == '\n'
		})
		var ids []string
		for _, part := range parts {
			part = strings.TrimSpace(part)
			if part == "" || strings.EqualFold(part, "and") {
				continue
			}
			ids = append(ids, part)
		}
		if len(ids) < 2 {
			return match
		}
		for i, id := range ids {
			ids[i] = "[" + id + "]"
		}
		return strings.Join(ids, " ")
	})
}

// formatCompositeReport wraps the final LLM report in the composite markdown
// document: stats header, curated sources, all analyzed URLs, and the raw
// findings in a collapsible section.
func (r *Researcher) formatCompositeReport(final string) string {
	var sb strings.Builder

	duration := float64(r.elapsedMS()) / 1000.0
	sb.WriteString("---\n\n## Research Summary\n\n")
	fmt.Fprintf(&sb, "**Duration:** %.1fs | **Rounds:** %d | **Queries:** %d | **URLs analyzed:** %d | **Model:** %s",
		duration, r.state.Round, len(r.state.QueriesUsed), len(r.state.AnalyzedURLs), r.cfg.Model)
	if cat := r.state.Category; cat != "" && cat != "general" {
		fmt.Fprintf(&sb, " | **Category:** %s%s", strings.ToUpper(cat[:1]), cat[1:])
	}
	sb.WriteString("\n\n---\n\n")

	sb.WriteString(normalizeSourceCitations(final))

	// Curated sources: quality-filtered findings, each URL at most once.
	seen := map[string]bool{}
	var sources []Finding
	for _, f := range r.state.Findings {
		if seen[f.URL] || isLowQuality(f.Summary) {
			continue
		}
		seen[f.URL] = true
		sources = append(sources, f)
	}
	if len(sources) > 0 {
		sb.WriteString("\n\n### Sources\n\n")
		for _, f := range sources {
			title := f.Title
			if title == "" {
				title = f.URL
			}
			fmt.Fprintf(&sb, "- [%s] [%s](%s)\n", r.sourceID(f), title, f.URL)
		}
		sb.WriteString("\n")
		for _, f := range sources {
			fmt.Fprintf(&sb, "[%s]: %s\n", r.sourceID(f), f.URL)
		}
	}

	if len(r.state.AnalyzedURLs) > 0 {
		sb.WriteString("\n### Analyzed URLs\n\n")
		seen = map[string]bool{}
		n := 0
		for _, u := range r.state.AnalyzedURLs {
			if seen[u.URL] {
				continue
			}
			seen[u.URL] = true
			n++
			title := u.Title
			if title == "" {
				title = u.URL
			}
			fmt.Fprintf(&sb, "%d. [%s](%s)\n", n, title, u.URL)
		}
	}

	if len(r.state.Findings) > 0 {
		fmt.Fprintf(&sb, "\n<details>\n<summary><strong>Raw collected findings (%d sources)</strong></summary>\n\n", len(r.state.Findings))
		for i, f := range r.state.Findings {
			title := f.Title
			if title == "" {
				title = f.URL
			}
			fmt.Fprintf(&sb, "**%d. [%s] [%s](%s)**\n\n%s\n\n", i+1, r.sourceID(f), title, f.URL, f.Summary)
		}
		sb.WriteString("</details>\n")
	}

	return sb.String()
}

// MarshalState JSON-encodes the slices of a State for checkpoint storage.
func MarshalState(st State) (findings, queries, urls string) {
	f, _ := json.Marshal(st.Findings)
	q, _ := json.Marshal(st.QueriesUsed)
	u, _ := json.Marshal(st.AnalyzedURLs)
	return string(f), string(q), string(u)
}

// UnmarshalState rebuilds a State from persisted job columns. Nil pointers
// and invalid JSON yield empty slices, so a partially persisted job still
// resumes cleanly.
func UnmarshalState(round, emptyRounds int, elapsedMS int64, category, slug, plan, report, findings, queries, urls *string) State {
	st := State{Round: round, EmptyRounds: emptyRounds, ElapsedMS: elapsedMS}
	if category != nil {
		st.Category = *category
	}
	if slug != nil {
		st.Slug = *slug
	}
	if plan != nil {
		st.Plan = *plan
	}
	if report != nil {
		st.Report = *report
	}
	if findings != nil {
		json.Unmarshal([]byte(*findings), &st.Findings) //nolint:errcheck
	}
	if queries != nil {
		json.Unmarshal([]byte(*queries), &st.QueriesUsed) //nolint:errcheck
	}
	if urls != nil {
		json.Unmarshal([]byte(*urls), &st.AnalyzedURLs) //nolint:errcheck
	}
	return st
}
