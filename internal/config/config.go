package config

import (
	"os"
	"strconv"

	"github.com/BurntSushi/toml"
)

type Config struct {
	Server      Server      `toml:"server"`
	Bootstrap   Bootstrap   `toml:"bootstrap"`
	ModelServer ModelServer `toml:"model_server"`
	Models      []Model     `toml:"model"`
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
	APIBase string `toml:"api_base"`
	Default string `toml:"default"`
}

type Model struct {
	Name        string `toml:"name"`
	DisplayName string `toml:"display_name"`
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

	if _, err := toml.DecodeFile(path, cfg); err != nil {
		return nil, err
	}

	applyEnv(cfg)
	return cfg, nil
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
