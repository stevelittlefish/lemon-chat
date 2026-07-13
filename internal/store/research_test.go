package store

import (
	"database/sql"
	"path/filepath"
	"testing"
)

func TestCreateResearchJobPersistsRedditImportOption(t *testing.T) {
	s := newTestStore(t)
	user, err := s.CreateUser("researcher", nil, false, nil)
	if err != nil {
		t.Fatal(err)
	}
	job, err := s.CreateResearchJob(user.ID, "Title", "Query", "model", "research", false, false, true, false, "", "", "", 3, 600)
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

func TestCreateResearchJobPersistsAutoHTMLReport(t *testing.T) {
	s := newTestStore(t)
	user, err := s.CreateUser("html-researcher", nil, false, nil)
	if err != nil {
		t.Fatal(err)
	}
	job, err := s.CreateResearchJob(user.ID, "Title", "Query", "model", "research", false, false, false, true, "earthy greens", "html-model", "", 3, 600)
	if err != nil {
		t.Fatal(err)
	}
	loaded, err := s.GetResearchJob(job.ID, user.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !loaded.AutoHTMLReport || loaded.HTMLReportDirection == nil || *loaded.HTMLReportDirection != "earthy greens" {
		t.Fatalf("auto HTML report options not persisted: %+v", loaded)
	}
	if loaded.HTMLReportModel == nil || *loaded.HTMLReportModel != "html-model" {
		t.Fatalf("HTML report model override not persisted: %+v", loaded)
	}
}

func TestCreateResearchJobPersistsWorkerModel(t *testing.T) {
	s := newTestStore(t)
	user, err := s.CreateUser("worker-researcher", nil, false, nil)
	if err != nil {
		t.Fatal(err)
	}
	job, err := s.CreateResearchJob(user.ID, "Title", "Query", "model", "research", false, false, false, false, "", "", "worker-model", 3, 600)
	if err != nil {
		t.Fatal(err)
	}
	loaded, err := s.GetResearchJob(job.ID, user.ID)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.WorkerModel == nil || *loaded.WorkerModel != "worker-model" {
		t.Fatalf("worker model override not persisted: %+v", loaded)
	}

	// No override stores NULL, not the empty string.
	job2, err := s.CreateResearchJob(user.ID, "Title", "Query", "model", "research", false, false, false, false, "", "", "", 3, 600)
	if err != nil {
		t.Fatal(err)
	}
	loaded2, err := s.GetResearchJob(job2.ID, user.ID)
	if err != nil {
		t.Fatal(err)
	}
	if loaded2.WorkerModel != nil {
		t.Fatalf("expected nil worker model, got %v", *loaded2.WorkerModel)
	}
}

func TestResearchRedditPauseAndResumeIsDurableAndIdempotent(t *testing.T) {
	s := newTestStore(t)
	user, err := s.CreateUser("reddit-researcher", nil, false, nil)
	if err != nil {
		t.Fatal(err)
	}
	job, err := s.CreateResearchJob(user.ID, "", "Query", "model", "research", false, false, true, false, "", "", "", 3, 600)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.SetResearchJobAwaitingReddit(job.ID, "request-1", `{"round":2}`, 1234); err != nil {
		t.Fatal(err)
	}
	paused, err := s.GetResearchJob(job.ID, user.ID)
	if err != nil {
		t.Fatal(err)
	}
	if paused.Status != ResearchStatusAwaitingReddit || paused.RedditRequestID == nil || *paused.RedditRequestID != "request-1" || paused.PendingRedditRound == nil {
		t.Fatalf("paused state was not persisted: %+v", paused)
	}
	resumable, err := s.ListResumableResearchJobs()
	if err != nil {
		t.Fatal(err)
	}
	if len(resumable) != 0 {
		t.Fatalf("awaiting Reddit job was returned for startup recovery: %+v", resumable)
	}
	if resumed, err := s.ResumeResearchRedditImport(job.ID, user.ID, "wrong-request", nil, true); err != nil || resumed {
		t.Fatalf("wrong request resumed=%t err=%v", resumed, err)
	}
	response := `{"version":1}`
	if resumed, err := s.ResumeResearchRedditImport(job.ID, user.ID, "request-1", &response, false); err != nil || !resumed {
		t.Fatalf("matching request resumed=%t err=%v", resumed, err)
	}
	if resumed, err := s.ResumeResearchRedditImport(job.ID, user.ID, "request-1", &response, false); err != nil || resumed {
		t.Fatalf("duplicate request resumed=%t err=%v", resumed, err)
	}
	resumed, err := s.GetResearchJob(job.ID, user.ID)
	if err != nil {
		t.Fatal(err)
	}
	if resumed.Status != ResearchStatusPending || resumed.RedditResponse == nil || *resumed.RedditResponse != response {
		t.Fatalf("resumed state was not persisted: %+v", resumed)
	}
	recovered, err := s.ListResumableResearchJobs()
	if err != nil {
		t.Fatal(err)
	}
	if len(recovered) != 1 || recovered[0].ID != job.ID || recovered[0].RedditResponse == nil {
		t.Fatalf("accepted import was not restart-resumable: %+v", recovered)
	}
	if err := s.CompleteResearchRedditRound(job.ID, 2, 0, 1500, "factcheck", "short_name", "plan", "report", `[]`, `["query"]`, `[]`); err != nil {
		t.Fatal(err)
	}
	completed, err := s.GetResearchJob(job.ID, user.ID)
	if err != nil {
		t.Fatal(err)
	}
	if completed.Round != 2 || completed.Slug == nil || *completed.Slug != "short_name" || completed.Report == nil || *completed.Report != "report" || completed.RedditRequestID != nil || completed.PendingRedditRound != nil || completed.RedditResponse != nil || completed.RedditSkipped {
		t.Fatalf("completed Reddit round was not atomically checkpointed and cleared: %+v", completed)
	}
}

func TestMigrationsV31AndV32PreserveExistingResearchData(t *testing.T) {
	path := filepath.Join(t.TempDir(), "v30.db")
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`
		PRAGMA foreign_keys = ON;
		CREATE TABLE user (id INTEGER PRIMARY KEY);
		INSERT INTO user (id) VALUES (7);
		CREATE TABLE schema_version (version INTEGER, timestamp TEXT);
		INSERT INTO schema_version (version, timestamp) VALUES (30, '2026-01-01T00:00:00Z');
		CREATE TABLE research_job (
			id INTEGER PRIMARY KEY, user_id INTEGER NOT NULL REFERENCES user(id) ON DELETE CASCADE,
			query TEXT NOT NULL, model TEXT NOT NULL,
			status TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending','running','done','error','cancelled')),
			phase TEXT, round INTEGER NOT NULL DEFAULT 0, empty_rounds INTEGER NOT NULL DEFAULT 0,
			elapsed_ms INTEGER NOT NULL DEFAULT 0, category TEXT, plan TEXT, report TEXT,
			final_report TEXT, findings TEXT, queries_used TEXT, analyzed_urls TEXT, error TEXT,
			created_at TEXT NOT NULL, updated_at TEXT NOT NULL, title TEXT,
			effort INTEGER NOT NULL DEFAULT 3, max_time_seconds INTEGER NOT NULL DEFAULT 0,
			mode TEXT NOT NULL DEFAULT 'research', force_search INTEGER NOT NULL DEFAULT 0,
			deep_report INTEGER NOT NULL DEFAULT 0, pause_reddit_import INTEGER NOT NULL DEFAULT 0
		);
		CREATE INDEX idx_research_job_user_id ON research_job(user_id);
		CREATE INDEX idx_research_job_status ON research_job(status);
		INSERT INTO research_job (
			id,user_id,query,model,status,phase,round,empty_rounds,elapsed_ms,category,plan,report,
			final_report,findings,queries_used,analyzed_urls,error,created_at,updated_at,title,effort,
			max_time_seconds,mode,force_search,deep_report,pause_reddit_import
		) VALUES (
			42,7,'original query','original model','done','writing',4,1,9876,'factcheck','saved plan','saved report',
			'saved final','[{"url":"https://example.com"}]','["query one"]','[{"url":"https://example.com"}]',NULL,
			'2026-01-01T00:00:00Z','2026-01-02T00:00:00Z','Original title',5,1200,'brainstorm',1,1,1
		);
	`)
	if err != nil {
		db.Close()
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}

	s, err := Open(path)
	if err != nil {
		t.Fatalf("migrate populated v30 database: %v", err)
	}
	defer s.Close()
	job, err := s.GetResearchJob(42, 7)
	if err != nil {
		t.Fatal(err)
	}
	if job.Query != "original query" || job.Model != "original model" || job.Status != ResearchStatusDone || job.Round != 4 || job.ElapsedMS != 9876 ||
		job.Title == nil || *job.Title != "Original title" || job.FinalReport == nil || *job.FinalReport != "saved final" || !job.PauseRedditImport || !job.DeepReport || !job.ForceSearch {
		t.Fatalf("migration did not preserve research job: %+v", job)
	}
	if job.RedditRequestID != nil || job.PendingRedditRound != nil || job.RedditResponse != nil || job.RedditSkipped {
		t.Fatalf("new Reddit state was not safely defaulted: %+v", job)
	}
	var version int
	if err := s.db.QueryRow(`SELECT MAX(version) FROM schema_version`).Scan(&version); err != nil || version != 38 {
		t.Fatalf("schema version = %d, err=%v; want 38", version, err)
	}
	var violations int
	rows, err := s.db.Query(`PRAGMA foreign_key_check`)
	if err != nil {
		t.Fatal(err)
	}
	for rows.Next() {
		violations++
	}
	rows.Close()
	if violations != 0 {
		t.Fatalf("foreign key violations after migration: %d", violations)
	}
	var foreignKeysEnabled int
	if err := s.db.QueryRow(`PRAGMA foreign_keys`).Scan(&foreignKeysEnabled); err != nil || foreignKeysEnabled != 1 {
		t.Fatalf("foreign_keys = %d, err=%v; want enabled", foreignKeysEnabled, err)
	}
	// The completed job's final_report was back-filled as its default report.
	def, err := s.GetDefaultResearchReport(42)
	if err != nil || def.Markdown == nil || *def.Markdown != "saved final" || !def.IsDefault {
		t.Fatalf("default report not back-filled: %+v, err=%v", def, err)
	}
}
