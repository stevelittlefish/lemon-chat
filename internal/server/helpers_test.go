package server

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stevelittlefish/lemon-chat/internal/store"
)

func TestPathID(t *testing.T) {
	tests := []struct {
		name       string
		value      string
		wantID     int64
		wantOK     bool
		wantStatus int
	}{
		{"valid", "42", 42, true, http.StatusOK},
		{"non-numeric", "abc", 0, false, http.StatusBadRequest},
		{"empty", "", 0, false, http.StatusBadRequest},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest("GET", "/x", nil)
			r.SetPathValue("id", tt.value)
			w := httptest.NewRecorder()

			id, ok := pathID(w, r)
			if ok != tt.wantOK {
				t.Fatalf("ok = %v, want %v", ok, tt.wantOK)
			}
			if id != tt.wantID {
				t.Errorf("id = %d, want %d", id, tt.wantID)
			}
			if !tt.wantOK && w.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", w.Code, tt.wantStatus)
			}
		})
	}
}

func TestNotFoundOr500(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantWrote  bool
		wantStatus int
	}{
		{"nil", nil, false, http.StatusOK},
		{"not found", store.ErrNotFound, true, http.StatusNotFound},
		{"other", fmt.Errorf("boom"), true, http.StatusInternalServerError},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			wrote := notFoundOr500(w, tt.err)
			if wrote != tt.wantWrote {
				t.Fatalf("wrote = %v, want %v", wrote, tt.wantWrote)
			}
			if tt.wantWrote && w.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", w.Code, tt.wantStatus)
			}
		})
	}
}

func TestLimitRequestBody(t *testing.T) {
	var readErr error
	h := limitRequestBody(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, readErr = io.Copy(io.Discard, r.Body)
	}))
	tests := []struct {
		name, path, contentType string
		size                    int
		wantErr                 bool
	}{
		{"json under limit", "/api/notes", "application/json", defaultBodyLimit, false},
		{"json over limit", "/api/notes", "application/json", defaultBodyLimit + 1, true},
		{"upload over json limit", "/api/conversations/1/attachments", "multipart/form-data; boundary=x", defaultBodyLimit + 1, false},
		{"upload over limit", "/api/conversations/1/attachments", "multipart/form-data; boundary=x", multipartBodyLimit + 1, true},
		{"import over json limit", "/api/conversations/import_chat", "application/json", defaultBodyLimit + 1, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest("POST", tt.path, strings.NewReader(strings.Repeat("x", tt.size)))
			r.Header.Set("Content-Type", tt.contentType)
			h.ServeHTTP(httptest.NewRecorder(), r)
			if (readErr != nil) != tt.wantErr {
				t.Fatalf("read err = %v, wantErr %v", readErr, tt.wantErr)
			}
		})
	}
}
