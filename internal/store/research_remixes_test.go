package store

import "testing"

func TestResearchRemixLifecycle(t *testing.T) {
	s := newTestStore(t)
	user, err := s.CreateUser("remix-researcher", nil, false, nil)
	if err != nil {
		t.Fatal(err)
	}
	job, err := s.CreateResearchJob(user.ID, "Title", "Question", "model", "research", false, false, false, false, "", 3, 600)
	if err != nil {
		t.Fatal(err)
	}
	price := 0.0123
	created, err := s.CreateResearchRemix(job.ID, "model", "earthy colours", "<!DOCTYPE html><html></html>", &price)
	if err != nil {
		t.Fatal(err)
	}
	items, err := s.ListResearchRemixes(job.ID)
	if err != nil || len(items) != 1 || items[0].HTML != "" || items[0].PriceUSD == nil || *items[0].PriceUSD != price {
		t.Fatalf("ListResearchRemixes = %#v, %v", items, err)
	}
	got, err := s.GetResearchRemix(created.ID, job.ID)
	if err != nil || got.HTML != "<!DOCTYPE html><html></html>" || got.Direction != "earthy colours" || got.PriceUSD == nil || *got.PriceUSD != price {
		t.Fatalf("GetResearchRemix = %#v, %v", got, err)
	}
	counts, err := s.ListResearchRemixCounts(user.ID)
	if err != nil || counts[job.ID] != 1 {
		t.Fatalf("ListResearchRemixCounts = %#v, %v", counts, err)
	}
}
