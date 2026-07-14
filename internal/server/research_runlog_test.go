package server

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadResearchDebug(t *testing.T) {
	dataDir := t.TempDir()
	dir := researchRunLogDir(dataDir, 42)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	meta := `{"job_id":42,"model":"gemma","synthesis_tokens":8192,"final_report_tokens":32768,` +
		`"status":"done","stop_reason":"YES — plan covered","rounds_completed":2,"findings_count":20}`
	if err := os.WriteFile(filepath.Join(dir, "meta.json"), []byte(meta), 0o644); err != nil {
		t.Fatal(err)
	}
	events := `{"ts":"2026-07-14T12:07:10Z","phase":"planning"}` + "\n" +
		`{"ts":"2026-07-14T12:10:43Z","phase":"deciding","round":2,"message":"YES — plan covered"}` + "\n"
	if err := os.WriteFile(filepath.Join(dir, "events.jsonl"), []byte(events), 0o644); err != nil {
		t.Fatal(err)
	}

	dbg := readResearchDebug(dataDir, 42)
	if !dbg.Available {
		t.Fatal("expected Available=true")
	}
	if dbg.Meta == nil || dbg.Meta.StopReason != "YES — plan covered" || dbg.Meta.RoundsCompleted != 2 {
		t.Fatalf("meta not parsed as expected: %+v", dbg.Meta)
	}
	if len(dbg.Events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(dbg.Events))
	}
	if dbg.Events[1].Phase != "deciding" || dbg.Events[1].Round != 2 {
		t.Errorf("second event not parsed: %+v", dbg.Events[1])
	}

	// A job with no run-log dir must report unavailable rather than erroring.
	if missing := readResearchDebug(dataDir, 99); missing.Available {
		t.Error("expected Available=false for a job with no run-log")
	}
}
