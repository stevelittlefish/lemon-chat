package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/BurntSushi/toml"
)

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
	MaxRounds             int    `toml:"max_rounds"`             // hard upper bound on research rounds
	MaxTimeSeconds        int    `toml:"max_time_seconds"`       // wall-clock budget, checked at the start of each round
	MaxURLsPerRound       int    `toml:"max_urls_per_round"`     // URLs fetched per query per round
	MaxContentChars       int    `toml:"max_content_chars"`      // page content truncation limit before extraction
	MaxReportTokens       int    `toml:"max_report_tokens"`      // max_tokens for synthesis and final report calls
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

type ModelServer struct {
	Name    string `toml:"name"`
	APIBase string `toml:"api_base"`
	APIKey  string `toml:"api_key"`
}

type Model struct {
	Name        string   `toml:"name"`
	DisplayName string   `toml:"display_name"`
	ModelServer string   `toml:"model_server"`
	Modes       []string `toml:"modes"`
}

// AvailableIn reports whether the model is available in the given mode.
// An empty Modes list means available in all modes.
func (m *Model) AvailableIn(mode string) bool {
	if len(m.Modes) == 0 {
		return true
	}
	for _, v := range m.Modes {
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
