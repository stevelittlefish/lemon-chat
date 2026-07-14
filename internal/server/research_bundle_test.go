package server

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"strings"
	"testing"

	"github.com/stevelittlefish/lemon-chat/internal/store"
)

func bundleFiles(t *testing.T, job *store.ResearchJob, reports []store.ResearchReport) map[string][]byte {
	return bundleTraceFiles(t, job, reports, nil, nil)
}

func bundleTraceFiles(t *testing.T, job *store.ResearchJob, reports []store.ResearchReport, events []store.ResearchEvent, calls []store.ResearchLLMCall) map[string][]byte {
	t.Helper()
	var buf bytes.Buffer
	if err := writeResearchBundle(&buf, job, reports, events, calls); err != nil {
		t.Fatalf("writeResearchBundle: %v", err)
	}
	data := buf.Bytes()
	files := make(map[string][]byte)
	for offset := 0; offset+4 <= len(data) && binary.LittleEndian.Uint32(data[offset:]) == 0x04034b50; {
		if offset+30 > len(data) {
			t.Fatal("truncated local ZIP header")
		}
		nameLen := int(binary.LittleEndian.Uint16(data[offset+26:]))
		extraLen := int(binary.LittleEndian.Uint16(data[offset+28:]))
		size := int(binary.LittleEndian.Uint32(data[offset+18:]))
		start := offset + 30
		contentStart := start + nameLen + extraLen
		contentEnd := contentStart + size
		if contentEnd > len(data) {
			t.Fatal("truncated ZIP entry")
		}
		name := string(data[start : start+nameLen])
		files[name] = append([]byte(nil), data[contentStart:contentEnd]...)
		offset = contentEnd
	}
	return files
}

func TestResearchBundleIncludesSelfContainedDebugTree(t *testing.T) {
	title := "Trace bundle"
	plan := "## Plan\n\nCheck the evidence."
	category := "general"
	report := "## Evolving synthesis\n"
	queries := `["battery prices"]`
	analyzed := `[{"url":"https://example.com/source","title":"Source"}]`
	findings := `[{"url":"https://example.com/source","title":"Source","summary":"Useful"}]`
	final := "# Final report\n"
	job := &store.ResearchJob{
		ID: 9, Title: &title, Query: "Debug this research", Model: "writer", Mode: "research",
		Status: store.ResearchStatusDone, Round: 1, Plan: &plan, Category: &category, Report: &report,
		FinalReport: &final, QueriesUsed: &queries, AnalyzedURLs: &analyzed, Findings: &findings,
	}
	reports := []store.ResearchReport{{ID: 30, Markdown: &final, Model: "writer", IsDefault: true}}
	events := []store.ResearchEvent{
		{ID: 1, Sequence: 1, EventType: "run_started", Phase: "planning", Data: `{"max_rounds":3,"max_time_ms":600000}`, CreatedAt: "2026-07-14T10:00:00Z"},
		{ID: 2, Sequence: 2, EventType: "queries_generated", Phase: "searching", Round: 1, Data: `{"queries":["battery prices"]}`, CreatedAt: "2026-07-14T10:00:01Z"},
		{ID: 3, Sequence: 3, EventType: "search_results", Phase: "searching", Round: 1, Data: `{"query":"battery prices","results":[{"url":"https://example.com/source"}]}`, CreatedAt: "2026-07-14T10:00:02Z"},
		{ID: 4, Sequence: 4, EventType: "extraction_completed", Phase: "reading", Round: 1, Data: `{"url":"https://example.com/source","summary":"Useful"}`, CreatedAt: "2026-07-14T10:00:03Z"},
		{ID: 5, Sequence: 5, EventType: "synthesis_completed", Phase: "analyzing", Round: 1, Data: `{"report":"## Evolving synthesis\n"}`, CreatedAt: "2026-07-14T10:00:04Z"},
		{ID: 6, Sequence: 6, EventType: "stop_decision", Phase: "deciding", Round: 1, Message: "YES — enough evidence", Data: `{"stop":true}`, CreatedAt: "2026-07-14T10:00:05Z"},
		{ID: 7, Sequence: 7, EventType: "html_generation_failed", Phase: "designing", Message: "model output truncated", Data: `{"stage":"generate"}`, CreatedAt: "2026-07-14T10:00:06Z"},
	}
	finish := "length"
	usage := `{"prompt_tokens":100,"completion_tokens":50,"total_tokens":150}`
	response := "partial response"
	duration := int64(1200)
	price := 0.012
	calls := []store.ResearchLLMCall{{
		ID: 1, Sequence: 1, Phase: "analyzing", Operation: "synthesize", Round: 1, Attempt: 2,
		Model: "writer", APIBase: "https://user:secret@models.example/v1?api_key=hidden",
		RequestMessages: `[{"role":"user","content":"prompt"}]`, Parameters: `{"max_tokens":8192,"api_key":"hidden"}`,
		DurationMS: &duration, Response: &response, FinishReason: &finish, Usage: &usage, PriceUSD: &price,
		Outcome: "succeeded", Disposition: "rejected",
	}}

	files := bundleTraceFiles(t, job, reports, events, calls)
	root := "trace-bundle/debug/"
	for _, name := range []string{
		"README.md", "events.jsonl", "trace-summary.json", "job-state.json", "plan.md", "classification.txt",
		"sources/queries-used.json", "sources/analyzed-urls.json", "sources/findings.json",
		"rounds/round-01/events.jsonl", "rounds/round-01/queries.json", "rounds/round-01/search-results.json",
		"rounds/round-01/findings.json", "rounds/round-01/synthesis-01.md", "rounds/round-01/decisions.json",
		"calls/001-synthesize-attempt-2.json", "calls/001-synthesize-attempt-2-response.txt",
		"errors.json", "limits.json", "html-attempts.json",
	} {
		if _, ok := files[root+name]; !ok {
			t.Errorf("bundle missing %s", root+name)
		}
	}
	var summary researchBundleTraceSummary
	if err := json.Unmarshal(files[root+"trace-summary.json"], &summary); err != nil {
		t.Fatal(err)
	}
	if summary.TotalTokens != 150 || summary.TruncationCount != 1 || summary.RetryCount != 1 || summary.HTMLFailures != 1 {
		t.Fatalf("unexpected trace summary: %+v", summary)
	}
	callJSON := string(files[root+"calls/001-synthesize-attempt-2.json"])
	if strings.Contains(callJSON, "hidden") || strings.Contains(callJSON, "user:secret") || !strings.Contains(callJSON, "[redacted]") {
		t.Fatalf("call record leaked credentials or missed redaction: %s", callJSON)
	}
}

func TestResearchBundleSingleReportUsesOneFolder(t *testing.T) {
	title := "Battery benchmark"
	markdown := "# Findings\n"
	queries := `["one","two"]`
	job := &store.ResearchJob{ID: 7, Title: &title, Query: "Is it worthwhile?", Model: "writer", Mode: "research", Status: store.ResearchStatusDone, Round: 3, ElapsedMS: 1250, QueriesUsed: &queries}
	reports := []store.ResearchReport{{ID: 11, Markdown: &markdown, HTML: "<!doctype html><title>Findings</title>", Model: "writer", IsDefault: true}}

	files := bundleFiles(t, job, reports)
	for _, name := range []string{"battery-benchmark/report.md", "battery-benchmark/report.html", "battery-benchmark/info.json"} {
		if _, ok := files[name]; !ok {
			t.Errorf("bundle missing %s; files=%v", name, files)
		}
	}
	var info researchBundleInfo
	if err := json.Unmarshal(files["battery-benchmark/info.json"], &info); err != nil {
		t.Fatalf("decoding info.json: %v", err)
	}
	if info.QueryCount != 2 || info.ReportCount != 1 || info.ElapsedMS != 1250 {
		t.Fatalf("unexpected bundle info: %+v", info)
	}
}

func TestResearchBundleMultipleReportsUsesReportSubfolders(t *testing.T) {
	markdown := "default"
	variantMarkdown := "variant"
	job := &store.ResearchJob{ID: 8, Query: "A/B comparison", Model: "writer", Mode: "research", Status: store.ResearchStatusDone}
	reports := []store.ResearchReport{
		{ID: 20, Markdown: &markdown, Model: "writer", IsDefault: true},
		{ID: 21, Markdown: &variantMarkdown, HTML: "<!doctype html><title>Variant</title>", Model: "designer", Direction: "Warm & clear"},
	}

	files := bundleFiles(t, job, reports)
	for _, name := range []string{
		"a-b-comparison/01-default/report.md",
		"a-b-comparison/02-warm-clear/report.md",
		"a-b-comparison/02-warm-clear/report.html",
		"a-b-comparison/info.json",
	} {
		if _, ok := files[name]; !ok {
			t.Errorf("bundle missing %s; files=%v", name, files)
		}
	}
	if _, exists := files["a-b-comparison/report.md"]; exists {
		t.Fatal("multi-report bundle placed a report directly in the parent folder")
	}
}

func TestResearchBundleWithoutReportStillContainsDebugArchive(t *testing.T) {
	errMessage := "research produced no findings"
	job := &store.ResearchJob{ID: 10, Query: "Failed run", Model: "writer", Mode: "research", Status: store.ResearchStatusError, Error: &errMessage}
	events := []store.ResearchEvent{{ID: 1, Sequence: 1, EventType: "run_finished", Phase: "terminal", Message: errMessage, Data: `{"status":"error"}`}}

	files := bundleTraceFiles(t, job, nil, events, nil)
	for _, name := range []string{"failed-run/info.json", "failed-run/debug/events.jsonl", "failed-run/debug/errors.json"} {
		if _, ok := files[name]; !ok {
			t.Errorf("report-less bundle missing %s", name)
		}
	}
}
