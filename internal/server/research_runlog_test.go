package server

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stevelittlefish/lemon-chat/internal/research"
	"github.com/stevelittlefish/lemon-chat/internal/store"
)

// TestRunLogPauseResumeNoCollision simulates a Reddit pause/resume: two run-log
// invocations write to the same directory. Every checkpoint must land in its own
// file (the engine's round number repeats across the plan checkpoint, the
// pre-increment pause checkpoint, and the resume), and the original start time
// must survive the resume.
func TestRunLogPauseResumeNoCollision(t *testing.T) {
	dataDir := t.TempDir()
	job := &store.ResearchJob{ID: 7, Effort: 3}
	cfg := research.Config{Model: "gemma", MaxRounds: 8, MinRounds: 2}

	// First invocation: plan checkpoint (round 0) then the Reddit-pause
	// checkpoint, which fires while state.Round is still 0.
	rl1 := newResearchRunLog(dataDir, job.ID)
	rl1.start(job, cfg)
	firstStart := rl1.meta.StartedAt
	rl1.checkpoint(research.State{Round: 0, Plan: "the plan"})
	rl1.checkpoint(research.State{Round: 0, QueriesUsed: []string{"q1", "q2"}})

	// Resume invocation reuses the directory.
	rl2 := newResearchRunLog(dataDir, job.ID)
	rl2.start(job, cfg)
	rl2.checkpoint(research.State{Round: 1, Report: "round 1 report"})
	rl2.checkpoint(research.State{Round: 2, Report: "round 2 report"})
	rl2.finish(store.ResearchStatusDone, "final report", research.State{Round: 2}, 42000)

	snaps, err := os.ReadDir(filepath.Join(researchRunLogDir(dataDir, job.ID), "snapshots"))
	if err != nil {
		t.Fatal(err)
	}
	if len(snaps) != 4 {
		names := make([]string, len(snaps))
		for i, e := range snaps {
			names[i] = e.Name()
		}
		t.Fatalf("expected 4 distinct snapshot files, got %d: %v", len(snaps), names)
	}
	if rl2.meta.StartedAt != firstStart {
		t.Errorf("resume overwrote started_at: first=%q resumed=%q", firstStart, rl2.meta.StartedAt)
	}
	if rl2.meta.ElapsedMS != 42000 {
		t.Errorf("expected server-provided elapsed 42000, got %d", rl2.meta.ElapsedMS)
	}
}

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
