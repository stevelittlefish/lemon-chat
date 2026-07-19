package server

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stevelittlefish/lemon-chat/internal/config"
)

func TestStaticHandlerServesFilesAndRejectsDirectories(t *testing.T) {
	staticDir := t.TempDir()
	writeStaticTestFile(t, staticDir, "index.html", "index page")
	writeStaticTestFile(t, staticDir, "app.js", "app script")
	writeStaticTestFile(t, staticDir, "assets/private.txt", "not a directory listing")

	handler := (&Server{cfg: &config.Config{Server: config.Server{StaticDir: staticDir}}}).Handler()

	tests := []struct {
		path       string
		wantStatus int
		wantBody   string
	}{
		{path: "/", wantStatus: http.StatusOK, wantBody: "index page"},
		{path: "/app.js", wantStatus: http.StatusOK, wantBody: "app script"},
		{path: "/assets", wantStatus: http.StatusNotFound},
		{path: "/assets/", wantStatus: http.StatusNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			resp := httptest.NewRecorder()
			handler.ServeHTTP(resp, req)
			if resp.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d; body: %s", resp.Code, tt.wantStatus, resp.Body.String())
			}
			if tt.wantBody != "" && !strings.Contains(resp.Body.String(), tt.wantBody) {
				t.Fatalf("body %q does not contain %q", resp.Body.String(), tt.wantBody)
			}
			if strings.Contains(resp.Body.String(), "private.txt") {
				t.Fatalf("response exposed directory entry: %q", resp.Body.String())
			}
		})
	}
}

func writeStaticTestFile(t *testing.T, root, name, content string) {
	t.Helper()
	path := filepath.Join(root, name)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("create test directory: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write test file: %v", err)
	}
}
