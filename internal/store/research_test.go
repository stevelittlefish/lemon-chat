package store

import "testing"

func TestCreateResearchJobPersistsRedditImportOption(t *testing.T) {
	s := newTestStore(t)
	user, err := s.CreateUser("researcher", nil, false, nil)
	if err != nil {
		t.Fatal(err)
	}
	job, err := s.CreateResearchJob(user.ID, "Title", "Query", "model", "research", false, false, true, 3, 600)
	if err != nil {
		t.Fatal(err)
	}
	if !job.PauseRedditImport {
		t.Fatal("created job did not retain pause_reddit_import")
	}
	loaded, err := s.GetResearchJob(job.ID, user.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !loaded.PauseRedditImport {
		t.Fatal("loaded job did not persist pause_reddit_import")
	}
}
