package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"slices"
	"strings"

	"github.com/stevelittlefish/lemon-chat/internal/config"
	"github.com/stevelittlefish/lemon-chat/internal/debug"
	"github.com/stevelittlefish/lemon-chat/internal/server"
	"github.com/stevelittlefish/lemon-chat/internal/store"
	"github.com/stevelittlefish/lemon-chat/internal/tasks"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	cfgPath := flag.String("config", "lemon.toml", "path to config file")
	debugFlag := flag.Bool("debug", false, "enable debug mode (overrides config)")
	listModelsFlag := flag.Bool("list-models", false, "list models from all configured model servers and exit")
	flag.Parse()

	cfg, err := config.Load(*cfgPath)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	debug.Enabled = cfg.Debug || *debugFlag

	if *listModelsFlag {
		listModels(cfg)
		os.Exit(0)
	}

	st, err := store.Open(cfg.Server.DBPath)
	if err != nil {
		log.Fatalf("open store: %v", err)
	}
	defer st.Close()

	if err := bootstrap(st, cfg); err != nil {
		log.Fatalf("bootstrap: %v", err)
	}

	hub := server.NewHub()
	tasks.StartTitleWorker(st, cfg, hub.BroadcastTitleUpdate)
	tasks.StartCleanupWorker(st)

	srv := server.New(cfg, st, hub)
	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	log.Printf("lemon-chat listening on http://localhost%s", addr)
	if err := http.ListenAndServe(addr, srv.Handler()); err != nil {
		log.Fatalf("server: %v", err)
	}
}

func listModels(cfg *config.Config) {
	type modelEntry struct {
		ID string `json:"id"`
	}
	type modelsResponse struct {
		Data []modelEntry `json:"data"`
	}

	for _, srv := range cfg.ModelServers {
		url := strings.TrimRight(srv.APIBase, "/") + "/models"
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  error building request for %s: %v\n", srv.Name, err)
			continue
		}
		if srv.APIKey != "" {
			req.Header.Set("Authorization", "Bearer "+srv.APIKey)
		}

		fmt.Printf("Model server: %s (%s)\n", srv.Name, srv.APIBase)

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  error: %v\n", err)
			continue
		}
		defer resp.Body.Close()

		var result modelsResponse
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			fmt.Fprintf(os.Stderr, "  error decoding response: %v\n", err)
			continue
		}

		slices.SortFunc(result.Data, func(a, b modelEntry) int { return strings.Compare(a.ID, b.ID) })

		if len(result.Data) == 0 {
			fmt.Println("  (no models)")
		}
		for _, m := range result.Data {
			fmt.Printf("  - %s\n", m.ID)
		}
	}
}

func bootstrap(st *store.Store, cfg *config.Config) error {
	n, err := st.UserCount()
	if err != nil {
		return err
	}
	if n > 0 {
		return nil
	}

	b := cfg.Bootstrap
	if b.AdminUsername == "" {
		return nil
	}

	var hash *string
	if b.AdminPassword != "" {
		h, err := bcrypt.GenerateFromPassword([]byte(b.AdminPassword), bcrypt.DefaultCost)
		if err != nil {
			return err
		}
		s := string(h)
		hash = &s
	}

	adminDisplayName := "Administrator"
	_, err = st.CreateUser(b.AdminUsername, hash, true, &adminDisplayName)
	if err != nil {
		return err
	}
	log.Printf("bootstrap: created admin user %q", b.AdminUsername)
	return nil
}
