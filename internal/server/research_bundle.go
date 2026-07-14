package server

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"hash/crc32"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/stevelittlefish/lemon-chat/internal/store"
)

var bundleSlugSeparator = regexp.MustCompile(`[^a-z0-9]+`)

type researchBundleInfo struct {
	ID                  int64                      `json:"id"`
	Title               *string                    `json:"title"`
	Question            string                     `json:"question"`
	Model               string                     `json:"model"`
	WorkerModel         *string                    `json:"worker_model"`
	HTMLReportModel     *string                    `json:"html_report_model"`
	Mode                string                     `json:"mode"`
	ForceSearch         bool                       `json:"force_search"`
	DeepReport          bool                       `json:"deep_report"`
	AutoHTMLReport      bool                       `json:"auto_html_report"`
	HTMLReportDirection *string                    `json:"html_report_direction"`
	PauseRedditImport   bool                       `json:"pause_reddit_import"`
	Status              string                     `json:"status"`
	Phase               *string                    `json:"phase"`
	Effort              int                        `json:"effort"`
	MaxTimeSeconds      int                        `json:"max_time_seconds"`
	Rounds              int                        `json:"rounds"`
	EmptyRounds         int                        `json:"empty_rounds"`
	ElapsedMS           int64                      `json:"elapsed_ms"`
	PriceUSD            *float64                   `json:"price_usd"`
	Category            *string                    `json:"category"`
	Slug                *string                    `json:"slug"`
	QueryCount          int                        `json:"query_count"`
	AnalyzedURLCount    int                        `json:"analyzed_url_count"`
	FindingCount        int                        `json:"finding_count"`
	ReportCount         int                        `json:"report_count"`
	Error               *string                    `json:"error"`
	CreatedAt           string                     `json:"created_at"`
	UpdatedAt           string                     `json:"updated_at"`
	Reports             []researchBundleReportInfo `json:"reports"`
}

type researchBundleReportInfo struct {
	ID                int64    `json:"id"`
	Folder            string   `json:"folder,omitempty"`
	Model             string   `json:"model"`
	Direction         string   `json:"direction"`
	MarkdownDirection string   `json:"markdown_direction,omitempty"`
	PriceUSD          *float64 `json:"price_usd"`
	IsDefault         bool     `json:"is_default"`
	HasMarkdown       bool     `json:"has_markdown"`
	HasHTML           bool     `json:"has_html"`
	CreatedAt         string   `json:"created_at"`
	UpdatedAt         string   `json:"updated_at"`
}

func bundleSlug(text string, fallback string) string {
	slug := strings.Trim(bundleSlugSeparator.ReplaceAllString(strings.ToLower(text), "-"), "-")
	if len(slug) > 60 {
		slug = strings.TrimRight(slug[:60], "-")
	}
	if slug == "" {
		return fallback
	}
	return slug
}

func jsonArrayLength(raw *string) int {
	if raw == nil || *raw == "" {
		return 0
	}
	var values []json.RawMessage
	if json.Unmarshal([]byte(*raw), &values) != nil {
		return 0
	}
	return len(values)
}

type bundleZipEntry struct {
	name   string
	crc    uint32
	size   uint32
	offset uint32
}

// bundleZipWriter emits uncompressed ZIP entries. Research bundles are small
// enough that pulling in a dependency solely for deflate support is unnecessary.
type bundleZipWriter struct {
	dst     io.Writer
	offset  uint32
	entries []bundleZipEntry
}

func (z *bundleZipWriter) write(value any) error {
	return binary.Write(z.dst, binary.LittleEndian, value)
}

func (z *bundleZipWriter) add(name, content string) error {
	data := []byte(content)
	if len(name) > 65535 || uint64(len(data)) > uint64(^uint32(0)) {
		return fmt.Errorf("bundle file too large: %s", name)
	}
	entry := bundleZipEntry{name: name, crc: crc32.ChecksumIEEE(data), size: uint32(len(data)), offset: z.offset}
	header := []any{
		uint32(0x04034b50), uint16(20), uint16(0x0800), uint16(0),
		uint16(0), uint16(0), entry.crc, entry.size, entry.size,
		uint16(len(name)), uint16(0),
	}
	for _, value := range header {
		if err := z.write(value); err != nil {
			return err
		}
	}
	if _, err := io.WriteString(z.dst, name); err != nil {
		return err
	}
	if _, err := z.dst.Write(data); err != nil {
		return err
	}
	z.offset += uint32(30 + len(name) + len(data))
	z.entries = append(z.entries, entry)
	return nil
}

func (z *bundleZipWriter) close() error {
	centralOffset := z.offset
	for _, entry := range z.entries {
		header := []any{
			uint32(0x02014b50), uint16(20), uint16(20), uint16(0x0800), uint16(0),
			uint16(0), uint16(0), entry.crc, entry.size, entry.size,
			uint16(len(entry.name)), uint16(0), uint16(0), uint16(0), uint16(0),
			uint32(0), entry.offset,
		}
		for _, value := range header {
			if err := z.write(value); err != nil {
				return err
			}
		}
		if _, err := io.WriteString(z.dst, entry.name); err != nil {
			return err
		}
		z.offset += uint32(46 + len(entry.name))
	}
	if len(z.entries) > 65535 {
		return fmt.Errorf("too many bundle files")
	}
	centralSize := z.offset - centralOffset
	footer := []any{
		uint32(0x06054b50), uint16(0), uint16(0), uint16(len(z.entries)),
		uint16(len(z.entries)), centralSize, centralOffset, uint16(0),
	}
	for _, value := range footer {
		if err := z.write(value); err != nil {
			return err
		}
	}
	return nil
}

func writeResearchBundle(dst io.Writer, job *store.ResearchJob, reports []store.ResearchReport) error {
	heading := job.Query
	if job.Title != nil && *job.Title != "" {
		heading = *job.Title
	}
	root := bundleSlug(heading, fmt.Sprintf("research-%d", job.ID))
	multiple := len(reports) > 1
	info := researchBundleInfo{
		ID: job.ID, Title: job.Title, Question: job.Query, Model: job.Model,
		WorkerModel: job.WorkerModel, HTMLReportModel: job.HTMLReportModel,
		Mode: job.Mode, ForceSearch: job.ForceSearch, DeepReport: job.DeepReport,
		AutoHTMLReport: job.AutoHTMLReport, HTMLReportDirection: job.HTMLReportDirection,
		PauseRedditImport: job.PauseRedditImport, Status: job.Status, Phase: job.Phase,
		Effort: job.Effort, MaxTimeSeconds: job.MaxTimeSeconds, Rounds: job.Round,
		EmptyRounds: job.EmptyRounds, ElapsedMS: job.ElapsedMS, PriceUSD: job.PriceUSD,
		Category: job.Category, Slug: job.Slug, QueryCount: jsonArrayLength(job.QueriesUsed),
		AnalyzedURLCount: jsonArrayLength(job.AnalyzedURLs), FindingCount: jsonArrayLength(job.Findings),
		ReportCount: len(reports), Error: job.Error, CreatedAt: job.CreatedAt, UpdatedAt: job.UpdatedAt,
		Reports: make([]researchBundleReportInfo, 0, len(reports)),
	}

	zw := &bundleZipWriter{dst: dst}
	for i := range reports {
		report := &reports[i]
		folder := ""
		if multiple {
			label := report.Direction
			if report.IsDefault {
				label = "default"
			} else if label == "" {
				label = fmt.Sprintf("report-%d", report.ID)
			}
			folder = fmt.Sprintf("%02d-%s", i+1, bundleSlug(label, fmt.Sprintf("report-%d", report.ID)))
		}
		info.Reports = append(info.Reports, researchBundleReportInfo{
			ID: report.ID, Folder: folder, Model: report.Model, Direction: report.Direction,
			MarkdownDirection: report.MarkdownDirection,
			PriceUSD:          report.PriceUSD, IsDefault: report.IsDefault,
			HasMarkdown: report.Markdown != nil && *report.Markdown != "", HasHTML: report.HTML != "",
			CreatedAt: report.CreatedAt, UpdatedAt: report.UpdatedAt,
		})
		prefix := root + "/"
		if folder != "" {
			prefix += folder + "/"
		}
		if report.Markdown != nil && *report.Markdown != "" {
			if err := zw.add(prefix+"report.md", *report.Markdown); err != nil {
				return err
			}
		}
		if report.HTML != "" {
			if err := zw.add(prefix+"report.html", report.HTML); err != nil {
				return err
			}
		}
	}

	metadata, err := json.MarshalIndent(info, "", "  ")
	if err != nil {
		return err
	}
	if err := zw.add(root+"/info.json", string(metadata)+"\n"); err != nil {
		return err
	}
	return zw.close()
}

// writeResearchDebugBundle zips the on-disk diagnostic run-log directory for a
// job (events.jsonl, per-round snapshots and reports, meta.json, and llm.jsonl
// when captured) under a single root folder. Files are read into memory — a
// run-log is small — and stored uncompressed via the shared bundle writer.
func writeResearchDebugBundle(dst io.Writer, root, dir string) (int, error) {
	zw := &bundleZipWriter{dst: dst}
	count := 0
	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		count++
		return zw.add(root+"/"+filepath.ToSlash(rel), string(content))
	})
	if err != nil {
		return 0, err
	}
	if err := zw.close(); err != nil {
		return 0, err
	}
	return count, nil
}

func (s *Server) handleDownloadResearchDebugBundle(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)
	jobID, ok := pathID(w, r)
	if !ok {
		return
	}
	// Ownership check — the job must exist and belong to this user.
	job, err := s.store.GetResearchJob(jobID, user.ID)
	if notFoundOr500(w, err) {
		return
	}
	dir := researchRunLogDir(s.cfg.Server.DataDir, job.ID)
	if info, err := os.Stat(dir); err != nil || !info.IsDir() {
		writeError(w, http.StatusNotFound, "no debug data for this job")
		return
	}

	root := fmt.Sprintf("research-%d-debug", job.ID)
	var bundle bytes.Buffer
	count, err := writeResearchDebugBundle(&bundle, root, dir)
	if err != nil {
		internalError(w, err)
		return
	}
	log.Printf("Downloading research debug bundle id=%d user_id=%d files=%d", job.ID, user.ID, count)
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s.zip"`, root))
	w.Header().Set("Content-Length", fmt.Sprintf("%d", bundle.Len()))
	w.WriteHeader(http.StatusOK)
	_, _ = bundle.WriteTo(w)
}

func (s *Server) handleDownloadResearchBundle(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)
	jobID, ok := pathID(w, r)
	if !ok {
		return
	}
	job, err := s.store.GetResearchJob(jobID, user.ID)
	if notFoundOr500(w, err) {
		return
	}
	reports, err := s.store.ListResearchReports(jobID)
	if err != nil {
		internalError(w, err)
		return
	}
	if len(reports) == 0 {
		writeError(w, http.StatusNotFound, "no reports to download")
		return
	}

	var bundle bytes.Buffer
	if err := writeResearchBundle(&bundle, job, reports); err != nil {
		internalError(w, err)
		return
	}
	heading := job.Query
	if job.Title != nil && *job.Title != "" {
		heading = *job.Title
	}
	filename := bundleSlug(heading, fmt.Sprintf("research-%d", job.ID)) + "-bundle.zip"
	log.Printf("Downloading research bundle id=%d user_id=%d reports=%d", job.ID, user.ID, len(reports))
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	w.Header().Set("Content-Length", fmt.Sprintf("%d", bundle.Len()))
	w.WriteHeader(http.StatusOK)
	_, _ = bundle.WriteTo(w)
}
