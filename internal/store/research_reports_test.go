package store

import "testing"

func TestResearchReportDefaultUpsert(t *testing.T) {
	s := newTestStore(t)
	user, err := s.CreateUser("report-researcher", nil, false, nil)
	if err != nil {
		t.Fatal(err)
	}
	job, err := s.CreateResearchJob(user.ID, "Title", "Question", "model-a", "research", false, false, false, false, "", "", "", 3, 600)
	if err != nil {
		t.Fatal(err)
	}

	// Finishing a job with a report creates the default report.
	report := "final markdown"
	if err := s.FinishResearchJob(job.ID, ResearchStatusDone, &report, nil, 1000, nil); err != nil {
		t.Fatal(err)
	}
	def, err := s.GetDefaultResearchReport(job.ID)
	if err != nil || def.Markdown == nil || *def.Markdown != report || !def.IsDefault || def.Model != "model-a" || def.HTML != "" {
		t.Fatalf("GetDefaultResearchReport = %+v, %v", def, err)
	}

	// Re-finishing updates the same default report rather than adding one.
	report2 := "revised markdown"
	if err := s.FinishResearchJob(job.ID, ResearchStatusDone, &report2, nil, 1200, nil); err != nil {
		t.Fatal(err)
	}
	reports, err := s.ListResearchReports(job.ID)
	if err != nil || len(reports) != 1 || reports[0].Markdown == nil || *reports[0].Markdown != report2 {
		t.Fatalf("ListResearchReports after re-finish = %#v, %v", reports, err)
	}

	// An HTML report variant becomes a non-default report carrying its own
	// markdown copy (the regenerate handler copies the master markdown when only
	// an HTML rendering is requested).
	if _, err := s.CreateResearchReport(job.ID, &report2, "<!DOCTYPE html><html></html>", "model-b", "bold", nil, false); err != nil {
		t.Fatal(err)
	}
	reports, err = s.ListResearchReports(job.ID)
	if err != nil || len(reports) != 2 {
		t.Fatalf("ListResearchReports after variant = %#v, %v", reports, err)
	}
	// Default sorts first; the variant copied the current master markdown.
	if !reports[0].IsDefault || reports[1].IsDefault {
		t.Fatalf("expected default first: %#v", reports)
	}
	if reports[1].Markdown == nil || *reports[1].Markdown != report2 {
		t.Fatalf("variant report did not copy master markdown: %+v", reports[1])
	}
}
