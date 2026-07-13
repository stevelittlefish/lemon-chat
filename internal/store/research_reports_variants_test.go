package store

import "testing"

func TestNonDefaultResearchReports(t *testing.T) {
	s := newTestStore(t)
	user, err := s.CreateUser("report-researcher", nil, false, nil)
	if err != nil {
		t.Fatal(err)
	}
	job, err := s.CreateResearchJob(user.ID, "Title", "Question", "model", "research", false, false, false, false, "", "", "", 3, 600)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.UpsertDefaultResearchReport(job.ID, "# master", "model"); err != nil {
		t.Fatal(err)
	}

	price := 0.02
	md := "# fresh markdown"
	// A markdown-only regeneration and an HTML-only variant.
	if _, err := s.CreateResearchReport(job.ID, &md, "", "writer", "", &price, false); err != nil {
		t.Fatal(err)
	}
	if _, err := s.CreateResearchReport(job.ID, nil, "<!DOCTYPE html><html></html>", "designer", "earthy", &price, false); err != nil {
		t.Fatal(err)
	}

	reports, err := s.ListNonDefaultResearchReports(job.ID)
	if err != nil || len(reports) != 2 {
		t.Fatalf("ListNonDefaultResearchReports = %#v, %v; want 2", reports, err)
	}
	// Newest first: the HTML-only variant was inserted last.
	if !reports[0].HasHTML || reports[0].HasMarkdown {
		t.Fatalf("expected newest report to be HTML-only: %#v", reports[0])
	}
	if !reports[1].HasMarkdown || reports[1].HasHTML {
		t.Fatalf("expected older report to be markdown-only: %#v", reports[1])
	}

	counts, err := s.ListNonDefaultResearchReportCounts(user.ID)
	if err != nil || counts[job.ID] != 2 {
		t.Fatalf("ListNonDefaultResearchReportCounts = %#v, %v; want job count 2", counts, err)
	}
}
