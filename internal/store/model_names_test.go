package store

import "testing"

func TestRenameModelUpdatesEveryReference(t *testing.T) {
	s, err := Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	u, _ := s.CreateUser("u", nil, false, nil)
	old, other := "gpt-6.1-sol", "gemma4-31b"
	if _, err := s.CreateConversation(u.ID, nil, &old, nil); err != nil {
		t.Fatal(err)
	}
	if _, err := s.CreateConversation(u.ID, nil, &other, nil); err != nil {
		t.Fatal(err)
	}
	if _, err := s.CreateCharacter("C", old, nil, nil, nil, u.ID, "private", false, nil); err != nil {
		t.Fatal(err)
	}
	if _, err := s.CreateResearchJob(u.ID, "", "q", old, "research", false, false, false, false, "", old, old, 1, 0); err != nil {
		t.Fatal(err)
	}

	n, err := s.RenameModel(old, "gpt-sol")
	if err != nil {
		t.Fatal(err)
	}
	if n != 5 { // conversation, character, research_job model + html_report_model + worker_model
		t.Errorf("renamed %d rows, want 5", n)
	}
	usage, err := s.ModelNamesInUse()
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]int{}
	for _, mu := range usage {
		got[mu.Name] = mu.Total
	}
	if got[old] != 0 || got["gpt-sol"] != 5 || got[other] != 1 {
		t.Errorf("usage after rename = %v", got)
	}
}
