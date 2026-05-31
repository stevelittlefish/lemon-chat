package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/BurntSushi/toml"
)

type Config struct {
	Server       Server        `toml:"server"`
	Bootstrap    Bootstrap     `toml:"bootstrap"`
	Debug        bool          `toml:"debug"`
	DefaultModel string        `toml:"default_model"`
	ModelServers []ModelServer `toml:"model_server"`
	Models       []Model       `toml:"model"`
}

type Server struct {
	Port   int    `toml:"port"`
	DBPath string `toml:"db_path"`
}

type Bootstrap struct {
	AdminUsername string `toml:"admin_username"`
	AdminPassword string `toml:"admin_password"`
}

type ModelServer struct {
	Name    string `toml:"name"`
	APIBase string `toml:"api_base"`
}

type Model struct {
	Name        string `toml:"name"`
	DisplayName string `toml:"display_name"`
	ModelServer string `toml:"model_server"`
}

func Load(path string) (*Config, error) {
	cfg := &Config{
		Server: Server{
			Port:   8080,
			DBPath: "lemon.db",
		},
		Bootstrap: Bootstrap{
			AdminUsername: "admin",
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

// APIBaseForModel returns the trimmed api_base URL for the server that hosts modelName.
func (c *Config) APIBaseForModel(modelName string) (string, error) {
	for _, m := range c.Models {
		if m.Name == modelName {
			for _, s := range c.ModelServers {
				if s.Name == m.ModelServer {
					return strings.TrimRight(s.APIBase, "/"), nil
				}
			}
			return "", fmt.Errorf("config: model server %q not found", m.ModelServer)
		}
	}
	return "", fmt.Errorf("config: model %q not found", modelName)
}

func applyEnv(cfg *Config) {
	if v := os.Getenv("LEMON_PORT"); v != "" {
		if port, err := strconv.Atoi(v); err == nil {
			cfg.Server.Port = port
		}
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
