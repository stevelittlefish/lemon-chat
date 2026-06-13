package server

import (
	"fmt"
	"net/http"
	"net/http/httptest"
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
