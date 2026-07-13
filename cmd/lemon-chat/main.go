package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/stevelittlefish/lemon-chat/internal/config"
	"github.com/stevelittlefish/lemon-chat/internal/debug"
	"github.com/stevelittlefish/lemon-chat/internal/llm"
	"github.com/stevelittlefish/lemon-chat/internal/server"
	"github.com/stevelittlefish/lemon-chat/internal/store"
	"github.com/stevelittlefish/lemon-chat/internal/tasks"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	cfgPath := flag.String("config", "lemon.toml", "path to config file")
	debugFlag := flag.Bool("debug", false, "enable debug mode (overrides config)")
	tokenLogFlag := flag.Bool("token-log", false, "log raw model SSE tokens to <data_dir>/model_tokens.log (overrides config)")
	listModelsFlag := flag.Bool("list-models", false, "list models from all configured model servers and exit")
	flag.Parse()

	cfg, err := config.Load(*cfgPath)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	debug.Enabled = cfg.Server.Debug || *debugFlag
	cfg.Server.TokenLog = cfg.Server.TokenLog || *tokenLogFlag
	if cfg.Server.TokenLog {
		log.Printf("token log enabled — writing to %s/model_tokens.log", cfg.Server.DataDir)
	}

	if *listModelsFlag {
		listModels(cfg)
		os.Exit(0)
	}

	st, err := store.Open(cfg.Server.DBPath)
	if err != nil {
		log.Fatalf("open store: %v", err)
	}
	defer st.Close()

	if n, err := st.ClearPendingAttachments(); err != nil {
		log.Printf("Warning: could not clear pending attachments: %v", err)
	} else if n > 0 {
		log.Printf("Cleared %d pending attachment(s) left over from previous run", n)
	}

	if err := bootstrap(st, cfg); err != nil {
		log.Fatalf("bootstrap: %v", err)
	}

	hub := server.NewHub()
	tasks.StartTitleWorker(st, cfg, hub.BroadcastTitleUpdate, hub.BroadcastCompletionTitleUpdate)
	tasks.StartCleanupWorker(st, hub.BroadcastConversationListChanged)
	tasks.StartPriceWorker(st, cfg)

	srv := server.New(cfg, st, hub)
	srv.ResumeResearchJobs()
	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	log.Printf("lemon-chat listening on http://localhost%s", addr)
	if err := http.ListenAndServe(addr, srv.Handler()); err != nil {
		log.Fatalf("server: %v", err)
	}
}

func listModels(cfg *config.Config) {
	for _, srv := range cfg.ModelServers {
		fmt.Printf("Model server: %s (%s)\n", srv.Name, srv.APIBase)
		models, err := llm.ListModels(context.Background(), http.DefaultClient, srv.APIBase, srv.APIKey)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  error: %v\n", err)
			continue
		}
		if len(models) == 0 {
			fmt.Println("  (no models)")
		}
		for _, model := range models {
			fmt.Printf("  - %s\n", model)
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
