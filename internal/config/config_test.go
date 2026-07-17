package config

import (
	"context"
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
