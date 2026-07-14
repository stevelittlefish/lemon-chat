package store

import (
	"sync"
	"testing"
)

func TestResearchEventsAreAppendOnlyAndSequencedPerJob(t *testing.T) {
	s := newTestStore(t)
	user, err := s.CreateUser("trace-researcher", nil, false, nil)
	if err != nil {
		t.Fatal(err)
	}
	job, err := s.CreateResearchJob(user.ID, "Trace", "Question", "model", "research", false, false, false, false, "", "", "", 3, 600)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := s.AppendResearchEvent(job.ID, "run_started", "planning", 0, "", `{"model":"model"}`); err != nil {
		t.Fatal(err)
	}
	if _, err := s.AppendResearchEvent(job.ID, "queries_generated", "searching", 1, "", `{"queries":["one"]}`); err != nil {
		t.Fatal(err)
	}
	events, err := s.ListResearchEvents(job.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 2 || events[0].Sequence != 1 || events[1].Sequence != 2 {
		t.Fatalf("events not sequenced: %+v", events)
	}
	if events[1].EventType != "queries_generated" || events[1].Round != 1 || events[1].Data != `{"queries":["one"]}` {
		t.Fatalf("unexpected second event: %+v", events[1])
	}
}

func TestResearchEventsSequenceConcurrentAppends(t *testing.T) {
	s := newTestStore(t)
	user, err := s.CreateUser("concurrent-trace", nil, false, nil)
	if err != nil {
		t.Fatal(err)
	}
	job, err := s.CreateResearchJob(user.ID, "Trace", "Question", "model", "research", false, false, false, false, "", "", "", 3, 600)
	if err != nil {
		t.Fatal(err)
	}

	const count = 12
	errs := make(chan error, count)
	var wg sync.WaitGroup
	for range count {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := s.AppendResearchEvent(job.ID, "extraction_completed", "reading", 1, "", "{}")
			errs <- err
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatalf("concurrent append: %v", err)
		}
	}
	events, err := s.ListResearchEvents(job.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != count {
		t.Fatalf("event count = %d, want %d", len(events), count)
	}
	for i, event := range events {
		if event.Sequence != int64(i+1) {
			t.Fatalf("event %d sequence = %d", i, event.Sequence)
		}
	}
}

func TestDeletingResearchJobDeletesTrace(t *testing.T) {
	s := newTestStore(t)
	user, err := s.CreateUser("trace-owner", nil, false, nil)
	if err != nil {
		t.Fatal(err)
	}
	job, err := s.CreateResearchJob(user.ID, "Trace", "Question", "model", "research", false, false, false, false, "", "", "", 3, 600)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.AppendResearchEvent(job.ID, "run_started", "planning", 0, "", "{}"); err != nil {
		t.Fatal(err)
	}
	if err := s.DeleteResearchJob(job.ID, user.ID); err != nil {
		t.Fatal(err)
	}
	events, err := s.ListResearchEvents(job.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 0 {
		t.Fatalf("trace survived job deletion: %+v", events)
	}
}
