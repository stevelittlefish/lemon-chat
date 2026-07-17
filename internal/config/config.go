package config

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/BurntSushi/toml"
)

// TokenSource resolves a bearer token at call time. It lets call sites obtain a
// (possibly refreshed) OAuth token or a static api_key through one seam without
// depending on the auth implementation. It returns "" (no error) when the
// request should be sent without an Authorization header.
type TokenSource func(context.Context) (string, error)

// StaticToken returns a TokenSource that always yields key (which may be empty).
func StaticToken(key string) TokenSource {
	return func(context.Context) (string, error) { return key, nil }
}

type Config struct {
	Server       Server        `toml:"server"`
	Bootstrap    Bootstrap     `toml:"bootstrap"`
	ModelServers []ModelServer `toml:"model_server"`
	Models       []Model       `toml:"model"`
	SearXNG      SearXNG       `toml:"searxng"`
	ComfyUI      ComfyUI       `toml:"comfyui"`
	Research     Research      `toml:"research"`
}

type Research struct {
	Model                 string `toml:"model"`                  // default model for research jobs (falls back to server.default_model)
	HTMLReportModel       string `toml:"html_report_model"`      // default model for the HTML report step (falls back to the job's model)
	WorkerModel           string `toml:"worker_model"`           // default model for the worker tier — extraction + slug/classify/query-gen/decide (falls back to the job's model)
	MaxRounds             int    `toml:"max_rounds"`             // hard upper bound on research rounds
	MaxTimeSeconds        int    `toml:"max_time_seconds"`       // wall-clock budget, checked at the start of each round
	MaxURLsPerRound       int    `toml:"max_urls_per_round"`     // URLs fetched per query per round
	MaxContentChars       int    `toml:"max_content_chars"`      // page content truncation limit before extraction
	MaxReportTokens       int    `toml:"max_report_tokens"`      // backward-compat alias — seeds synthesis_tokens when that key is unset
	SynthesisTokens       int    `toml:"synthesis_tokens"`       // max_tokens for the per-round synthesis call
	FinalReportTokens     int    `toml:"final_report_tokens"`    // max_tokens for the final report (and deep-report section) writes
	HTMLReportTokens      int    `toml:"html_report_tokens"`     // max_tokens per call for the designed HTML report step (truncated calls are auto-continued)
	ExtractionConcurrency int    `toml:"extraction_concurrency"` // concurrent URL fetch+extract tasks
	MinRounds             int    `toml:"min_rounds"`             // stop-check is skipped until this many rounds complete
	MaxEmptyRounds        int    `toml:"max_empty_rounds"`       // consecutive zero-finding rounds before aborting
	SynthesisWindow       int    `toml:"synthesis_window"`       // only the last N findings are passed to each synthesis call
}

type SearXNG struct {
	URL string `toml:"url"`
}

type ComfyUI struct {
	URL          string `toml:"url"`
	SDXLWorkflow string `toml:"sdxl_workflow"`
	FluxWorkflow string `toml:"flux_workflow"`
}

type Server struct {
	Port                   int    `toml:"port"`
	DataDir                string `toml:"data_dir"`
	DBPath                 string `toml:"db_path"`
	Debug                  bool   `toml:"debug"`
	TokenLog               bool   `toml:"token_log"`
	DefaultModel           string `toml:"default_model"`
	DialTimeoutSeconds     int    `toml:"dial_timeout_seconds"`
	ResponseTimeoutSeconds int    `toml:"response_timeout_seconds"`
	MaxToolLoops           int    `toml:"max_tool_loops"`
	Timezone               string `toml:"timezone"`
}

type Bootstrap struct {
	AdminUsername string `toml:"admin_username"`
	AdminPassword string `toml:"admin_password"`
}

// API surface constants for ModelServer.API.
const (
	APIChatCompletions = "chat_completions" // OpenAI-compatible /chat/completions (default)
	APIResponses       = "responses"        // OpenAI Responses API (/responses)
)

// Auth mode constants for ModelServer.Auth.
const (
	AuthAPIKey = "api_key" // static api_key from config (default)
	AuthOAuth  = "oauth"   // shared OAuth token from the token store (OpenAI login)
)

type ModelServer struct {
	Name    string `toml:"name"`
	APIBase string `toml:"api_base"`
	APIKey  string `toml:"api_key"`
	// API selects the request/response surface. Empty defaults to
	// chat_completions; set to "responses" for OpenAI's Responses API.
	API string `toml:"api"`
	// Auth selects how requests are authenticated. Empty defaults to api_key;
	// set to "oauth" to use the shared OpenAI login token instead of api_key.
	Auth string `toml:"auth"`
}

// UsesResponses reports whether this server speaks the OpenAI Responses API
// rather than the chat-completions surface.
func (s *ModelServer) UsesResponses() bool {
	return s.API == APIResponses
}

// UsesOAuth reports whether requests to this server should authenticate with
// the shared OAuth token rather than a static api_key.
func (s *ModelServer) UsesOAuth() bool {
	return s.Auth == AuthOAuth
}

// Endpoint returns the full URL for a chat/generation request against this
// server, selecting the path from the configured API surface.
func (s *ModelServer) Endpoint() string {
	if s.UsesResponses() {
		return s.APIBase + "/responses"
	}
	return s.APIBase + "/chat/completions"
}

// IsOpenRouter reports whether this server is OpenRouter, which publishes a
// public model catalogue with per-token pricing. Detected from the API base
// host so no extra config is required.
func (s *ModelServer) IsOpenRouter() bool {
	return strings.Contains(s.APIBase, "openrouter.ai")
}

type Model struct {
	Name        string    `toml:"name"`
	DisplayName string    `toml:"display_name"`
	ModelServer string    `toml:"model_server"`
	Modes       *[]string `toml:"modes"`
}

// AvailableIn reports whether the model is available in the given mode.
// An omitted Modes key (nil) means available in all modes. An explicitly
// empty list (modes = []) means available in none of the picker-filtered
// modes — useful for a model reserved for research, which lists all models
// regardless of mode.
func (m *Model) AvailableIn(mode string) bool {
	if m.Modes == nil {
		return true
	}
	for _, v := range *m.Modes {
		if v == mode {
			return true
		}
	}
	return false
}

func Load(path string) (*Config, error) {
	cfg := &Config{
		Server: Server{
			Port:                   8080,
			DataDir:                ".",
			DialTimeoutSeconds:     10,
			ResponseTimeoutSeconds: 600,
			MaxToolLoops:           5,
		},
		Bootstrap: Bootstrap{
			AdminUsername: "admin",
		},
		Research: Research{
			MaxRounds:             8,
			MaxTimeSeconds:        600,
			MaxURLsPerRound:       3,
			MaxContentChars:       15000,
			MaxReportTokens:       8192,
			FinalReportTokens:     32768,
			HTMLReportTokens:      16384,
			ExtractionConcurrency: 3,
			MinRounds:             2,
			MaxEmptyRounds:        2,
			SynthesisWindow:       10,
		},
	}

	meta, err := toml.DecodeFile(path, cfg)
	if err != nil {
		return nil, err
	}
	if keys := meta.Undecoded(); len(keys) > 0 {
		return nil, fmt.Errorf("config: unknown or misplaced keys: %v", keys)
	}

	applyEnv(cfg)

	// max_report_tokens historically capped both synthesis and the final report.
	// It now only seeds synthesis_tokens, and only when the new key is unset, so
	// existing configs keep their synthesis budget while the final report gets
	// the roomier default.
	if cfg.Research.SynthesisTokens == 0 {
		cfg.Research.SynthesisTokens = cfg.Research.MaxReportTokens
	}

	if cfg.Server.DBPath == "" {
		cfg.Server.DBPath = filepath.Join(cfg.Server.DataDir, "lemon.db")
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

func (c *Config) Validate() error {
	serverNames := make(map[string]struct{}, len(c.ModelServers))
	for _, s := range c.ModelServers {
		if s.Name == "" {
			return fmt.Errorf("config: model_server missing name field")
		}
		switch s.API {
		case "", APIChatCompletions, APIResponses:
		default:
			return fmt.Errorf("config: model_server %q has invalid api %q (want %q or %q)", s.Name, s.API, APIChatCompletions, APIResponses)
		}
		switch s.Auth {
		case "", AuthAPIKey, AuthOAuth:
		default:
			return fmt.Errorf("config: model_server %q has invalid auth %q (want %q or %q)", s.Name, s.Auth, AuthAPIKey, AuthOAuth)
		}
		serverNames[s.Name] = struct{}{}
	}
	for _, m := range c.Models {
		if m.ModelServer == "" {
			return fmt.Errorf("config: model %q missing model_server field", m.Name)
		}
		if _, ok := serverNames[m.ModelServer]; !ok {
			return fmt.Errorf("config: model %q references unknown model_server %q", m.Name, m.ModelServer)
		}
	}
	return nil
}

// ServerForModel returns the ModelServer that hosts modelName.
func (c *Config) ServerForModel(modelName string) (*ModelServer, error) {
	for _, m := range c.Models {
		if m.Name == modelName {
			for i, s := range c.ModelServers {
				if s.Name == m.ModelServer {
					srv := c.ModelServers[i]
					srv.APIBase = strings.TrimRight(s.APIBase, "/")
					return &srv, nil
				}
			}
			return nil, fmt.Errorf("config: model server %q not found", m.ModelServer)
		}
	}
	return nil, fmt.Errorf("config: model %q not found", modelName)
}

func applyEnv(cfg *Config) {
	if v := os.Getenv("LEMON_PORT"); v != "" {
		if port, err := strconv.Atoi(v); err == nil {
			cfg.Server.Port = port
		}
	}
	if v := os.Getenv("LEMON_DATA_DIR"); v != "" {
		cfg.Server.DataDir = v
	}
	if v := os.Getenv("LEMON_DB_PATH"); v != "" {
		cfg.Server.DBPath = v
	}
	if v := os.Getenv("LEMON_ADMIN_USERNAME"); v != "" {
		cfg.Bootstrap.AdminUsername = v
	}
	if v := os.Getenv("LEMON_ADMIN_PASSWORD"); v != "" {
		cfg.Bootstrap.AdminPassword = v
	}
}
