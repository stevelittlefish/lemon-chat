package config

import (
	"strings"
	"testing"
)

func TestExpandEnvSubstitutesNestedStrings(t *testing.T) {
	t.Setenv("LEMON_TEST_KEY", `sk-"quoted"#1`)
	t.Setenv("LEMON_TEST_HOST", "comfy.example")
	modes := []string{"${LEMON_TEST_HOST}"}
	cfg := &Config{
		ModelServers: []ModelServer{{Name: "or", APIKey: "${LEMON_TEST_KEY}"}},
		Models:       []Model{{Name: "m", Modes: &modes}},
		ComfyUI: ComfyUI{
			URL:      "https://${LEMON_TEST_HOST}/",
			Profiles: map[string]ComfyProfile{"general": {Workflow: "${LEMON_TEST_HOST}"}},
		},
		Bootstrap: Bootstrap{AdminPassword: "literal $${NOT_A_VAR} and $HOME"},
	}
	if err := expandEnv(cfg); err != nil {
		t.Fatal(err)
	}
	if got := cfg.ModelServers[0].APIKey; got != `sk-"quoted"#1` {
		t.Errorf("api_key = %q", got)
	}
	if got := cfg.ComfyUI.URL; got != "https://comfy.example/" {
		t.Errorf("comfyui url = %q", got)
	}
	if got := cfg.ComfyUI.Profiles["general"].Workflow; got != "comfy.example" {
		t.Errorf("profile workflow = %q", got)
	}
	if got := (*cfg.Models[0].Modes)[0]; got != "comfy.example" {
		t.Errorf("modes = %q", got)
	}
	if got := cfg.Bootstrap.AdminPassword; got != "literal ${NOT_A_VAR} and $HOME" {
		t.Errorf("escaped value = %q", got)
	}
}

func TestExpandEnvRejectsUnsetVariable(t *testing.T) {
	cfg := &Config{ModelServers: []ModelServer{{Name: "a"}, {Name: "b", APIKey: "${LEMON_TEST_DEFINITELY_UNSET}"}}}
	err := expandEnv(cfg)
	if err == nil {
		t.Fatal("expected an error for an unset variable")
	}
	for _, want := range []string{"model_server[1].api_key", "LEMON_TEST_DEFINITELY_UNSET"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q does not mention %q", err, want)
		}
	}
}

func TestExpandEnvAllowsEmptyVariable(t *testing.T) {
	t.Setenv("LEMON_TEST_EMPTY", "")
	cfg := &Config{SearXNG: SearXNG{URL: "${LEMON_TEST_EMPTY}"}}
	if err := expandEnv(cfg); err != nil {
		t.Fatal(err)
	}
	if cfg.SearXNG.URL != "" {
		t.Errorf("url = %q", cfg.SearXNG.URL)
	}
}
