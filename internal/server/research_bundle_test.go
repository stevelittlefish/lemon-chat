package server

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"testing"

	"github.com/stevelittlefish/lemon-chat/internal/store"
)

func bundleFiles(t *testing.T, job *store.ResearchJob, reports []store.ResearchReport) map[string][]byte {
	t.Helper()
	var buf bytes.Buffer
	if err := writeResearchBundle(&buf, job, reports); err != nil {
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
