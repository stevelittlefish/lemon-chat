package llm

import (
	"context"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
)

func TestListModels(t *testing.T) {
	var authorization string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authorization = r.Header.Get("Authorization")
		if r.URL.Path != "/v1/models" {
			t.Fatalf("path = %q, want /v1/models", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[{"id":"zeta"},{"id":"alpha"},{"id":""}]}`))
	}))
	defer srv.Close()

	got, err := ListModels(context.Background(), srv.Client(), srv.URL+"/v1/", "secret")
	if err != nil {
		t.Fatalf("ListModels: %v", err)
	}
	if want := []string{"alpha", "zeta"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("models = %#v, want %#v", got, want)
	}
	if authorization != "Bearer secret" {
		t.Fatalf("Authorization = %q, want bearer token", authorization)
	}
}

func TestListModelsReportsHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "provider unavailable", http.StatusBadGateway)
	}))
	defer srv.Close()

	_, err := ListModels(context.Background(), srv.Client(), srv.URL, "")
	if err == nil || !strings.Contains(err.Error(), "502") {
		t.Fatalf("error = %v, want status code", err)
	}
}
