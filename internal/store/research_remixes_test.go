package store

import "testing"

func TestResearchRemixLifecycle(t *testing.T) {
	s := newTestStore(t)
	user, err := s.CreateUser("remix-researcher", nil, false, nil)
	if err != nil {
		t.Fatal(err)
	}
	job, err := s.CreateResearchJob(user.ID, "Title", "Question", "model", "research", false, false, false, 3, 600)
	if err != nil {
		t.Fatal(err)
	}
	created, err := s.CreateResearchRemix(job.ID, "model", "earthy colours", "<!DOCTYPE html><html></html>")
	if err != nil {
		t.Fatal(err)
	}
	items, err := s.ListResearchRemixes(job.ID)
	if err != nil || len(items) != 1 || items[0].HTML != "" {
		t.Fatalf("ListResearchRemixes = %#v, %v", items, err)
	}
	got, err := s.GetResearchRemix(created.ID, job.ID)
	if err != nil || got.HTML != "<!DOCTYPE html><html></html>" || got.Direction != "earthy colours" {
		t.Fatalf("GetResearchRemix = %#v, %v", got, err)
	}
}
