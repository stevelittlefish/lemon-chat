package server

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"hash/crc32"
	"io"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strings"

	"github.com/stevelittlefish/lemon-chat/internal/store"
)

var bundleSlugSeparator = regexp.MustCompile(`[^a-z0-9]+`)

type researchBundleInfo struct {
	ID                  int64                      `json:"id"`
	Title               *string                    `json:"title"`
	Question            string                     `json:"question"`
	Model               string                     `json:"model"`
	WorkerModel         *string                    `json:"worker_model"`
	HTMLReportModel     *string                    `json:"html_report_model"`
	Mode                string                     `json:"mode"`
	ForceSearch         bool                       `json:"force_search"`
	DeepReport          bool                       `json:"deep_report"`
	AutoHTMLReport      bool                       `json:"auto_html_report"`
	HTMLReportDirection *string                    `json:"html_report_direction"`
	PauseRedditImport   bool                       `json:"pause_reddit_import"`
	Status              string                     `json:"status"`
	Phase               *string                    `json:"phase"`
	Effort              int                        `json:"effort"`
	MaxTimeSeconds      int                        `json:"max_time_seconds"`
	Rounds              int                        `json:"rounds"`
	EmptyRounds         int                        `json:"empty_rounds"`
	ElapsedMS           int64                      `json:"elapsed_ms"`
	PriceUSD            *float64                   `json:"price_usd"`
	Category            *string                    `json:"category"`
	Slug                *string                    `json:"slug"`
	QueryCount          int                        `json:"query_count"`
	AnalyzedURLCount    int                        `json:"analyzed_url_count"`
	FindingCount        int                        `json:"finding_count"`
	ReportCount         int                        `json:"report_count"`
	Error               *string                    `json:"error"`
	CreatedAt           string                     `json:"created_at"`
	UpdatedAt           string                     `json:"updated_at"`
	Reports             []researchBundleReportInfo `json:"reports"`
}

type researchBundleReportInfo struct {
	ID          int64    `json:"id"`
	Folder      string   `json:"folder,omitempty"`
	Model       string   `json:"model"`
	Direction   string   `json:"direction"`
	PriceUSD    *float64 `json:"price_usd"`
	IsDefault   bool     `json:"is_default"`
	HasMarkdown bool     `json:"has_markdown"`
	HasHTML     bool     `json:"has_html"`
	CreatedAt   string   `json:"created_at"`
	UpdatedAt   string   `json:"updated_at"`
}

func bundleSlug(text string, fallback string) string {
	slug := strings.Trim(bundleSlugSeparator.ReplaceAllString(strings.ToLower(text), "-"), "-")
	if len(slug) > 60 {
		slug = strings.TrimRight(slug[:60], "-")
	}
	if slug == "" {
		return fallback
	}
	return slug
}

func jsonArrayLength(raw *string) int {
	if raw == nil || *raw == "" {
		return 0
	}
	var values []json.RawMessage
	if json.Unmarshal([]byte(*raw), &values) != nil {
		return 0
	}
	return len(values)
}

type bundleZipEntry struct {
	name   string
	crc    uint32
	size   uint32
	offset uint32
}

// bundleZipWriter emits uncompressed ZIP entries. Research bundles are small
// enough that pulling in a dependency solely for deflate support is unnecessary.
type bundleZipWriter struct {
	dst     io.Writer
	offset  uint32
	entries []bundleZipEntry
}

func (z *bundleZipWriter) write(value any) error {
	return binary.Write(z.dst, binary.LittleEndian, value)
}

func (z *bundleZipWriter) add(name, content string) error {
	data := []byte(content)
	if len(name) > 65535 || uint64(len(data)) > uint64(^uint32(0)) {
		return fmt.Errorf("bundle file too large: %s", name)
	}
	entry := bundleZipEntry{name: name, crc: crc32.ChecksumIEEE(data), size: uint32(len(data)), offset: z.offset}
	header := []any{
		uint32(0x04034b50), uint16(20), uint16(0x0800), uint16(0),
		uint16(0), uint16(0), entry.crc, entry.size, entry.size,
		uint16(len(name)), uint16(0),
	}
	for _, value := range header {
		if err := z.write(value); err != nil {
			return err
		}
	}
	if _, err := io.WriteString(z.dst, name); err != nil {
		return err
	}
	if _, err := z.dst.Write(data); err != nil {
		return err
	}
	z.offset += uint32(30 + len(name) + len(data))
	z.entries = append(z.entries, entry)
	return nil
}

func (z *bundleZipWriter) close() error {
	centralOffset := z.offset
	for _, entry := range z.entries {
		header := []any{
			uint32(0x02014b50), uint16(20), uint16(20), uint16(0x0800), uint16(0),
			uint16(0), uint16(0), entry.crc, entry.size, entry.size,
			uint16(len(entry.name)), uint16(0), uint16(0), uint16(0), uint16(0),
			uint32(0), entry.offset,
		}
		for _, value := range header {
			if err := z.write(value); err != nil {
				return err
			}
		}
		if _, err := io.WriteString(z.dst, entry.name); err != nil {
			return err
		}
		z.offset += uint32(46 + len(entry.name))
	}
	if len(z.entries) > 65535 {
		return fmt.Errorf("too many bundle files")
	}
	centralSize := z.offset - centralOffset
	footer := []any{
		uint32(0x06054b50), uint16(0), uint16(0), uint16(len(z.entries)),
		uint16(len(z.entries)), centralSize, centralOffset, uint16(0),
	}
	for _, value := range footer {
		if err := z.write(value); err != nil {
			return err
		}
	}
	return nil
}

func bundleJSON(value any) (string, error) {
	encoded, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return "", err
	}
	return string(encoded) + "\n", nil
}

func addBundleJSON(zw *bundleZipWriter, name string, value any) error {
	content, err := bundleJSON(value)
	if err != nil {
		return fmt.Errorf("encode %s: %w", name, err)
	}
	return zw.add(name, content)
}

func decodedBundleJSON(raw string) any {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	var value any
	if err := json.Unmarshal([]byte(raw), &value); err != nil {
		return raw
	}
	return value
}

func bundleEventRecord(event store.ResearchEvent) map[string]any {
	return map[string]any{
		"id": event.ID, "sequence": event.Sequence, "event_type": event.EventType,
		"phase": event.Phase, "round": event.Round, "message": event.Message,
		"data": decodedBundleJSON(event.Data), "created_at": event.CreatedAt,
	}
}

func bundleEventsJSONL(events []store.ResearchEvent) (string, error) {
	var out strings.Builder
	for _, event := range events {
		encoded, err := json.Marshal(bundleEventRecord(event))
		if err != nil {
			return "", err
		}
		out.Write(encoded)
		out.WriteByte('\n')
	}
	return out.String(), nil
}

func sanitizeBundleAPIBase(raw string) string {
	parsed, err := url.Parse(raw)
	if err != nil {
		return "[invalid API base omitted]"
	}
	parsed.User = nil
	parsed.RawQuery = ""
	parsed.ForceQuery = false
	parsed.Fragment = ""
	return parsed.String()
}

func redactBundleSecrets(value any) any {
	sensitive := func(key string) bool {
		normalized := strings.ToLower(strings.ReplaceAll(strings.ReplaceAll(key, "-", "_"), " ", "_"))
		for _, fragment := range []string{"api_key", "apikey", "authorization", "access_token", "secret", "password", "cookie"} {
			if strings.Contains(normalized, fragment) {
				return true
			}
		}
		return false
	}
	switch typed := value.(type) {
	case map[string]any:
		out := make(map[string]any, len(typed))
		for key, item := range typed {
			if sensitive(key) {
				out[key] = "[redacted]"
			} else {
				out[key] = redactBundleSecrets(item)
			}
		}
		return out
	case []any:
		out := make([]any, len(typed))
		for i, item := range typed {
			out[i] = redactBundleSecrets(item)
		}
		return out
	default:
		return value
	}
}

func bundleCallRecord(call store.ResearchLLMCall) (map[string]any, error) {
	encoded, err := json.Marshal(call)
	if err != nil {
		return nil, err
	}
	var record map[string]any
	if err := json.Unmarshal(encoded, &record); err != nil {
		return nil, err
	}
	delete(record, "research_job_id")
	record["api_base"] = sanitizeBundleAPIBase(call.APIBase)
	record["request_messages"] = redactBundleSecrets(decodedBundleJSON(call.RequestMessages))
	record["parameters"] = redactBundleSecrets(decodedBundleJSON(call.Parameters))
	if call.Usage != nil {
		record["usage"] = redactBundleSecrets(decodedBundleJSON(*call.Usage))
	}
	redacted, _ := redactBundleSecrets(record).(map[string]any)
	return redacted, nil
}

func bundleNumber(value any) int64 {
	switch typed := value.(type) {
	case float64:
		return int64(typed)
	case json.Number:
		result, _ := typed.Int64()
		return result
	default:
		return 0
	}
}

func bundleUsage(call store.ResearchLLMCall) (prompt, completion, total int64) {
	if call.Usage == nil {
		return 0, 0, 0
	}
	usage, ok := decodedBundleJSON(*call.Usage).(map[string]any)
	if !ok {
		return 0, 0, 0
	}
	first := func(keys ...string) int64 {
		for _, key := range keys {
			if value := bundleNumber(usage[key]); value != 0 {
				return value
			}
		}
		return 0
	}
	prompt = first("prompt_tokens", "input_tokens", "promptTokens", "inputTokens")
	completion = first("completion_tokens", "output_tokens", "completionTokens", "outputTokens")
	total = first("total_tokens", "totalTokens")
	if total == 0 {
		total = prompt + completion
	}
	return prompt, completion, total
}

func bundleCallTruncated(call store.ResearchLLMCall) bool {
	if call.FinishReason == nil {
		return false
	}
	reason := strings.ToLower(*call.FinishReason)
	return reason == "length" || strings.Contains(reason, "max_token") || strings.Contains(reason, "max token")
}

func bundleProblemEvent(event store.ResearchEvent) bool {
	return strings.HasSuffix(event.EventType, "_failed") || event.EventType == "fetch_failed" ||
		(event.EventType == "run_finished" && event.Message != "") ||
		event.Phase == "warning" || strings.Contains(strings.ToLower(event.Message), "truncat") ||
		strings.Contains(strings.ToLower(event.Message), "error") || strings.Contains(strings.ToLower(event.Message), "failed")
}

type researchBundleTraceSummary struct {
	EventCount       int            `json:"event_count"`
	LLMCallCount     int            `json:"llm_call_count"`
	TotalDurationMS  int64          `json:"total_call_duration_ms"`
	PromptTokens     int64          `json:"prompt_tokens"`
	CompletionTokens int64          `json:"completion_tokens"`
	TotalTokens      int64          `json:"total_tokens"`
	KnownCostUSD     float64        `json:"known_call_cost_usd"`
	PricedCallCount  int            `json:"priced_call_count"`
	ErrorCount       int            `json:"error_count"`
	TruncationCount  int            `json:"truncation_count"`
	RetryCount       int            `json:"retry_count"`
	HTMLAttempts     int            `json:"html_attempt_count"`
	HTMLCompletions  int            `json:"html_completion_count"`
	HTMLFailures     int            `json:"html_failure_count"`
	CallsByPhase     map[string]int `json:"calls_by_phase"`
	Dispositions     map[string]int `json:"dispositions"`
	FinishReasons    map[string]int `json:"finish_reasons"`
}

func buildBundleTraceSummary(events []store.ResearchEvent, calls []store.ResearchLLMCall) researchBundleTraceSummary {
	summary := researchBundleTraceSummary{
		EventCount: len(events), LLMCallCount: len(calls), CallsByPhase: map[string]int{},
		Dispositions: map[string]int{}, FinishReasons: map[string]int{},
	}
	for _, event := range events {
		if bundleProblemEvent(event) {
			summary.ErrorCount++
		}
		switch event.EventType {
		case "html_generation_started":
			summary.HTMLAttempts++
		case "html_generation_completed":
			summary.HTMLCompletions++
		case "html_generation_failed":
			summary.HTMLFailures++
		}
	}
	for _, call := range calls {
		summary.CallsByPhase[call.Phase]++
		if call.DurationMS != nil {
			summary.TotalDurationMS += *call.DurationMS
		}
		prompt, completion, total := bundleUsage(call)
		summary.PromptTokens += prompt
		summary.CompletionTokens += completion
		summary.TotalTokens += total
		if call.PriceUSD != nil {
			summary.KnownCostUSD += *call.PriceUSD
			summary.PricedCallCount++
		}
		if call.Error != nil || call.Outcome == "failed" {
			summary.ErrorCount++
		}
		if bundleCallTruncated(call) {
			summary.TruncationCount++
		}
		if call.Attempt > 1 || call.Disposition == "retried" {
			summary.RetryCount++
		}
		if call.Disposition != "" {
			summary.Dispositions[call.Disposition]++
		}
		if call.FinishReason != nil && *call.FinishReason != "" {
			summary.FinishReasons[*call.FinishReason]++
		}
	}
	return summary
}

func addResearchBundleDebug(zw *bundleZipWriter, prefix string, job *store.ResearchJob, events []store.ResearchEvent, calls []store.ResearchLLMCall) error {
	summary := buildBundleTraceSummary(events, calls)
	readme := fmt.Sprintf(`# Research debug archive

This directory is a self-contained record of research job %d. Start with
%s and events.jsonl, then inspect rounds/ and calls/ for exact inputs and
outputs. JSONL files contain one complete JSON object per line.

- events.jsonl: every durable trace event in chronological sequence
- trace-summary.json: aggregate timing, token, cost, retry, and failure counts
- job-state.json: latest persisted research state and configuration
- plan.md and classification.txt: planning outputs
- sources/: complete query, analyzed-URL, and finding state
- rounds/: queries, raw search results, URL choices, findings, synthesis versions,
  stop decisions, checkpoints, and warnings for each round
- calls/: every model request, parameters, raw/partial response, provider usage,
  HTTP outcome, timing, price, finish reason, and semantic disposition
- errors.json: warnings, failed calls, truncations, retries, and fallbacks
- limits.json: configured limits and every recorded limit/budget decision
- html-attempts.json: automatic and manual HTML-generation outcomes

API keys and HTTP headers are never recorded. API-base credentials, query
parameters, and fragments are removed, and secret-shaped parameter fields are
redacted defensively.
`, job.ID, "trace-summary.json")
	if err := zw.add(prefix+"/README.md", readme); err != nil {
		return err
	}
	if err := addBundleJSON(zw, prefix+"/trace-summary.json", summary); err != nil {
		return err
	}
	eventsJSONL, err := bundleEventsJSONL(events)
	if err != nil {
		return err
	}
	if err := zw.add(prefix+"/events.jsonl", eventsJSONL); err != nil {
		return err
	}

	state := map[string]any{
		"id": job.ID, "title": job.Title, "question": job.Query, "model": job.Model,
		"worker_model": job.WorkerModel, "html_report_model": job.HTMLReportModel,
		"mode": job.Mode, "force_search": job.ForceSearch, "deep_report": job.DeepReport,
		"auto_html_report": job.AutoHTMLReport, "html_report_direction": job.HTMLReportDirection,
		"pause_reddit_import": job.PauseRedditImport, "status": job.Status, "phase": job.Phase,
		"effort": job.Effort, "max_time_seconds": job.MaxTimeSeconds, "round": job.Round,
		"empty_rounds": job.EmptyRounds, "elapsed_ms": job.ElapsedMS, "price_usd": job.PriceUSD,
		"category": job.Category, "slug": job.Slug, "plan": job.Plan, "report": job.Report,
		"final_report": job.FinalReport, "findings": decodedBundleJSON(pointerString(job.Findings)),
		"queries_used":  decodedBundleJSON(pointerString(job.QueriesUsed)),
		"analyzed_urls": decodedBundleJSON(pointerString(job.AnalyzedURLs)), "error": job.Error,
		"created_at": job.CreatedAt, "updated_at": job.UpdatedAt,
	}
	if err := addBundleJSON(zw, prefix+"/job-state.json", state); err != nil {
		return err
	}
	if job.Plan != nil && *job.Plan != "" {
		if err := zw.add(prefix+"/plan.md", *job.Plan+trailingNewline(*job.Plan)); err != nil {
			return err
		}
	}
	if job.Category != nil && *job.Category != "" {
		if err := zw.add(prefix+"/classification.txt", strings.TrimSpace(*job.Category)+"\n"); err != nil {
			return err
		}
	}
	for name, raw := range map[string]*string{
		"queries-used.json": job.QueriesUsed, "analyzed-urls.json": job.AnalyzedURLs, "findings.json": job.Findings,
	} {
		if err := addBundleJSON(zw, prefix+"/sources/"+name, decodedBundleJSON(pointerString(raw))); err != nil {
			return err
		}
	}

	var problems []any
	var limitDecisions []any
	var htmlAttempts []any
	rounds := make(map[int][]store.ResearchEvent)
	for _, event := range events {
		if event.Round > 0 {
			rounds[event.Round] = append(rounds[event.Round], event)
		}
		if bundleProblemEvent(event) {
			problems = append(problems, map[string]any{"kind": "event", "record": bundleEventRecord(event)})
		}
		message := strings.ToLower(event.Message)
		if strings.Contains(message, "limit") || strings.Contains(message, "budget") || strings.Contains(message, "maximum") || strings.Contains(message, "exhaust") {
			limitDecisions = append(limitDecisions, map[string]any{"kind": "event", "record": bundleEventRecord(event)})
		}
		if strings.HasPrefix(event.EventType, "html_generation_") {
			htmlAttempts = append(htmlAttempts, bundleEventRecord(event))
		}
		if event.EventType == "run_started" {
			limitDecisions = append(limitDecisions, map[string]any{"kind": "configured_run_limits", "record": bundleEventRecord(event)})
		}
	}

	for _, call := range calls {
		record, err := bundleCallRecord(call)
		if err != nil {
			return err
		}
		base := fmt.Sprintf("%03d-%s-attempt-%d", call.Sequence, bundleSlug(call.Operation, "call"), call.Attempt)
		if err := addBundleJSON(zw, prefix+"/calls/"+base+".json", record); err != nil {
			return err
		}
		if call.Response != nil {
			if err := zw.add(prefix+"/calls/"+base+"-response.txt", *call.Response+trailingNewline(*call.Response)); err != nil {
				return err
			}
		}
		if call.Error != nil || call.Outcome == "failed" || bundleCallTruncated(call) ||
			call.Attempt > 1 || call.Disposition == "retried" || call.Disposition == "fallback" || call.Disposition == "rejected" {
			problems = append(problems, map[string]any{"kind": "llm_call", "record": record})
		}
		parameters, _ := record["parameters"].(map[string]any)
		limitDecisions = append(limitDecisions, map[string]any{
			"kind": "call_limit", "sequence": call.Sequence, "operation": call.Operation,
			"round": call.Round, "attempt": call.Attempt, "max_tokens": parameters["max_tokens"],
			"finish_reason": call.FinishReason, "truncated": bundleCallTruncated(call), "disposition": call.Disposition,
		})
		if call.Operation == "html_report" {
			htmlAttempts = append(htmlAttempts, map[string]any{
				"kind": "llm_call", "sequence": call.Sequence, "attempt": call.Attempt,
				"model": call.Model, "outcome": call.Outcome, "disposition": call.Disposition,
				"finish_reason": call.FinishReason, "error": call.Error, "price_usd": call.PriceUSD,
			})
		}
	}
	if err := addBundleJSON(zw, prefix+"/errors.json", problems); err != nil {
		return err
	}
	if err := addBundleJSON(zw, prefix+"/limits.json", limitDecisions); err != nil {
		return err
	}
	if err := addBundleJSON(zw, prefix+"/html-attempts.json", htmlAttempts); err != nil {
		return err
	}

	roundNumbers := make([]int, 0, len(rounds))
	for round := range rounds {
		roundNumbers = append(roundNumbers, round)
	}
	sort.Ints(roundNumbers)
	for _, round := range roundNumbers {
		if err := addResearchBundleRound(zw, fmt.Sprintf("%s/rounds/round-%02d", prefix, round), rounds[round]); err != nil {
			return err
		}
	}
	return nil
}

func pointerString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func trailingNewline(value string) string {
	if strings.HasSuffix(value, "\n") {
		return ""
	}
	return "\n"
}

func addResearchBundleRound(zw *bundleZipWriter, prefix string, events []store.ResearchEvent) error {
	eventsJSONL, err := bundleEventsJSONL(events)
	if err != nil {
		return err
	}
	if err := zw.add(prefix+"/events.jsonl", eventsJSONL); err != nil {
		return err
	}
	artifacts := map[string][]any{
		"queries.json": {}, "search-results.json": {}, "url-selection.json": {},
		"findings.json": {}, "extraction-batches.json": {}, "decisions.json": {},
		"checkpoints.json": {}, "warnings.json": {},
	}
	synthesis := 0
	for _, event := range events {
		data := decodedBundleJSON(event.Data)
		switch event.EventType {
		case "queries_generated":
			artifacts["queries.json"] = append(artifacts["queries.json"], data)
		case "search_results":
			artifacts["search-results.json"] = append(artifacts["search-results.json"], data)
		case "urls_selected":
			artifacts["url-selection.json"] = append(artifacts["url-selection.json"], data)
		case "extraction_completed", "extraction_rejected", "fetch_failed":
			artifacts["findings.json"] = append(artifacts["findings.json"], bundleEventRecord(event))
		case "extraction_batch_completed":
			artifacts["extraction-batches.json"] = append(artifacts["extraction-batches.json"], data)
		case "stop_decision":
			artifacts["decisions.json"] = append(artifacts["decisions.json"], bundleEventRecord(event))
		case "checkpoint":
			artifacts["checkpoints.json"] = append(artifacts["checkpoints.json"], data)
		case "synthesis_completed":
			synthesis++
			if object, ok := data.(map[string]any); ok {
				if report, ok := object["report"].(string); ok {
					name := fmt.Sprintf("%s/synthesis-%02d.md", prefix, synthesis)
					if err := zw.add(name, report+trailingNewline(report)); err != nil {
						return err
					}
				}
			}
		}
		if bundleProblemEvent(event) {
			artifacts["warnings.json"] = append(artifacts["warnings.json"], bundleEventRecord(event))
		}
	}
	for name, values := range artifacts {
		if len(values) == 0 {
			continue
		}
		if err := addBundleJSON(zw, prefix+"/"+name, values); err != nil {
			return err
		}
	}
	return nil
}

func writeResearchBundle(dst io.Writer, job *store.ResearchJob, reports []store.ResearchReport, events []store.ResearchEvent, calls []store.ResearchLLMCall) error {
	heading := job.Query
	if job.Title != nil && *job.Title != "" {
		heading = *job.Title
	}
	root := bundleSlug(heading, fmt.Sprintf("research-%d", job.ID))
	multiple := len(reports) > 1
	info := researchBundleInfo{
		ID: job.ID, Title: job.Title, Question: job.Query, Model: job.Model,
		WorkerModel: job.WorkerModel, HTMLReportModel: job.HTMLReportModel,
		Mode: job.Mode, ForceSearch: job.ForceSearch, DeepReport: job.DeepReport,
		AutoHTMLReport: job.AutoHTMLReport, HTMLReportDirection: job.HTMLReportDirection,
		PauseRedditImport: job.PauseRedditImport, Status: job.Status, Phase: job.Phase,
		Effort: job.Effort, MaxTimeSeconds: job.MaxTimeSeconds, Rounds: job.Round,
		EmptyRounds: job.EmptyRounds, ElapsedMS: job.ElapsedMS, PriceUSD: job.PriceUSD,
		Category: job.Category, Slug: job.Slug, QueryCount: jsonArrayLength(job.QueriesUsed),
		AnalyzedURLCount: jsonArrayLength(job.AnalyzedURLs), FindingCount: jsonArrayLength(job.Findings),
		ReportCount: len(reports), Error: job.Error, CreatedAt: job.CreatedAt, UpdatedAt: job.UpdatedAt,
		Reports: make([]researchBundleReportInfo, 0, len(reports)),
	}

	zw := &bundleZipWriter{dst: dst}
	for i := range reports {
		report := &reports[i]
		folder := ""
		if multiple {
			label := report.Direction
			if report.IsDefault {
				label = "default"
			} else if label == "" {
				label = fmt.Sprintf("report-%d", report.ID)
			}
			folder = fmt.Sprintf("%02d-%s", i+1, bundleSlug(label, fmt.Sprintf("report-%d", report.ID)))
		}
		info.Reports = append(info.Reports, researchBundleReportInfo{
			ID: report.ID, Folder: folder, Model: report.Model, Direction: report.Direction,
			PriceUSD: report.PriceUSD, IsDefault: report.IsDefault,
			HasMarkdown: report.Markdown != nil && *report.Markdown != "", HasHTML: report.HTML != "",
			CreatedAt: report.CreatedAt, UpdatedAt: report.UpdatedAt,
		})
		prefix := root + "/"
		if folder != "" {
			prefix += folder + "/"
		}
		if report.Markdown != nil && *report.Markdown != "" {
			if err := zw.add(prefix+"report.md", *report.Markdown); err != nil {
				return err
			}
		}
		if report.HTML != "" {
			if err := zw.add(prefix+"report.html", report.HTML); err != nil {
				return err
			}
		}
	}

	metadata, err := json.MarshalIndent(info, "", "  ")
	if err != nil {
		return err
	}
	if err := zw.add(root+"/info.json", string(metadata)+"\n"); err != nil {
		return err
	}
	if err := addResearchBundleDebug(zw, root+"/debug", job, events, calls); err != nil {
		return err
	}
	return zw.close()
}

func (s *Server) handleDownloadResearchBundle(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)
	jobID, ok := pathID(w, r)
	if !ok {
		return
	}
	job, err := s.store.GetResearchJob(jobID, user.ID)
	if notFoundOr500(w, err) {
		return
	}
	reports, err := s.store.ListResearchReports(jobID)
	if err != nil {
		internalError(w, err)
		return
	}
	events, err := s.store.ListResearchEvents(jobID)
	if err != nil {
		internalError(w, err)
		return
	}
	calls, err := s.store.ListResearchLLMCalls(jobID)
	if err != nil {
		internalError(w, err)
		return
	}

	var bundle bytes.Buffer
	if err := writeResearchBundle(&bundle, job, reports, events, calls); err != nil {
		internalError(w, err)
		return
	}
	heading := job.Query
	if job.Title != nil && *job.Title != "" {
		heading = *job.Title
	}
	filename := bundleSlug(heading, fmt.Sprintf("research-%d", job.ID)) + "-bundle.zip"
	log.Printf("Downloading research bundle id=%d user_id=%d reports=%d", job.ID, user.ID, len(reports))
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	w.Header().Set("Content-Length", fmt.Sprintf("%d", bundle.Len()))
	w.WriteHeader(http.StatusOK)
	_, _ = bundle.WriteTo(w)
}
