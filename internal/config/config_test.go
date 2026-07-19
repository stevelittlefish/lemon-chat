package config

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestModelServerAPIAndAuthHelpers(t *testing.T) {
	def := ModelServer{Name: "local", APIBase: "http://x/v1"}
	if def.UsesResponses() {
		t.Error("empty api should default to chat completions")
	}
	if def.UsesOAuth() {
		t.Error("empty auth should default to api_key")
	}
	if got := def.Endpoint(); got != "http://x/v1/chat/completions" {
		t.Errorf("endpoint = %q", got)
	}

	resp := ModelServer{Name: "oai", APIBase: "http://x/v1", API: APIResponses, Auth: AuthOAuth}
	if !resp.UsesResponses() || !resp.UsesOAuth() {
		t.Error("responses/oauth flags not detected")
	}
	if got := resp.Endpoint(); got != "http://x/v1/responses" {
		t.Errorf("responses endpoint = %q", got)
	}
}

func TestValidateRejectsBadAPIAndAuth(t *testing.T) {
	badAPI := &Config{ModelServers: []ModelServer{{Name: "s", API: "bogus"}}}
	if err := badAPI.Validate(); err == nil {
		t.Error("expected error for invalid api")
	}
	badAuth := &Config{ModelServers: []ModelServer{{Name: "s", Auth: "bogus"}}}
	if err := badAuth.Validate(); err == nil {
		t.Error("expected error for invalid auth")
	}
	ok := &Config{ModelServers: []ModelServer{{Name: "s", API: APIResponses, Auth: AuthOAuth}}}
	if err := ok.Validate(); err != nil {
		t.Errorf("valid config rejected: %v", err)
	}
}

func TestStaticToken(t *testing.T) {
	got, err := StaticToken("abc")(context.Background())
	if err != nil || got != "abc" {
		t.Errorf("StaticToken = %q, %v", got, err)
	}
}

// writeStaticDirConfig writes a config containing only static_dir into a fresh
// temp directory and makes that directory the working directory, so relative
// static_dir values resolve there instead of against the package directory.
func writeStaticDirConfig(t *testing.T, staticDir string) string {
	t.Helper()
	dir := t.TempDir()
	t.Chdir(dir)
	path := filepath.Join(dir, "lemon.toml")
	body := fmt.Sprintf("[server]\nstatic_dir = %q\n", staticDir)
	if err := os.WriteFile(path, []byte(body), 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	return path
}

func TestLoadResolvesStaticDirAbsolutely(t *testing.T) {
	path := writeStaticDirConfig(t, "test-static")
	if err := os.Mkdir("test-static", 0755); err != nil {
		t.Fatalf("create static dir: %v", err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	want, err := filepath.Abs("test-static")
	if err != nil {
		t.Fatalf("resolve expected path: %v", err)
	}
	if cfg.Server.StaticDir != want {
		t.Fatalf("static dir = %q, want %q", cfg.Server.StaticDir, want)
	}
}

func TestLoadRejectsEmptyStaticDir(t *testing.T) {
	path := writeStaticDirConfig(t, "")

	if _, err := Load(path); err == nil {
		t.Fatal("load config with empty static_dir: got nil error")
	}
}

func TestLoadRejectsMissingStaticDir(t *testing.T) {
	path := writeStaticDirConfig(t, "nonexistent-static")

	// A typo must fail at startup rather than turning every page into a 404.
	if _, err := Load(path); err == nil {
		t.Fatal("load config with missing static_dir: got nil error")
	}
}

func TestLoadRejectsStaticDirThatIsAFile(t *testing.T) {
	path := writeStaticDirConfig(t, "static-file")
	if err := os.WriteFile("static-file", []byte("not a directory"), 0644); err != nil {
		t.Fatalf("write static file: %v", err)
	}

	if _, err := Load(path); err == nil {
		t.Fatal("load config with a file as static_dir: got nil error")
	}
}
