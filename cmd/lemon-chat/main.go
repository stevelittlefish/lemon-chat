package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/stevelittlefish/lemon-chat/internal/config"
	"github.com/stevelittlefish/lemon-chat/internal/debug"
	"github.com/stevelittlefish/lemon-chat/internal/llm"
	"github.com/stevelittlefish/lemon-chat/internal/openai_auth"
	"github.com/stevelittlefish/lemon-chat/internal/server"
	"github.com/stevelittlefish/lemon-chat/internal/store"
	"github.com/stevelittlefish/lemon-chat/internal/tasks"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	cfgPath := flag.String("config", "lemon.toml", "path to config file")
	debugFlag := flag.Bool("debug", false, "enable debug mode (overrides config)")
	tokenLogFlag := flag.Bool("token-log", false, "log replay-oriented model HTTP transcripts to <data_dir>/model_tokens.log (overrides config)")
	listModelsFlag := flag.Bool("list-models", false, "list models from all configured model servers and exit")
	openaiLoginFlag := flag.Bool("openai-login", false, "run the OpenAI (Codex) account login on this host and exit")
	modelsInUseFlag := flag.Bool("models-in-use", false, "list the model names stored in the database, flagging any not in the config, and exit")
	renameModelFlag := flag.Bool("rename-model", false, "rename stored model names and exit: --rename-model OLD NEW [OLD NEW ...]")
	flag.Parse()

	cfg, err := config.Load(*cfgPath)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	debug.Enabled = cfg.Server.Debug || *debugFlag
	cfg.Server.TokenLog = cfg.Server.TokenLog || *tokenLogFlag
	if cfg.Server.TokenLog {
		log.Printf("model wire log enabled — writing redacted transcripts to %s/model_tokens.log", cfg.Server.DataDir)
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

	if *openaiLoginFlag {
		openaiLogin(st)
		os.Exit(0)
	}

	if *modelsInUseFlag {
		modelsInUse(st, cfg)
		os.Exit(0)
	}

	if *renameModelFlag {
		renameModels(st, cfg, flag.Args())
		os.Exit(0)
	}

	if n, err := st.ClearPendingAttachments(); err != nil {
		log.Printf("Warning: could not clear pending attachments: %v", err)
	} else if n > 0 {
		log.Printf("Cleared %d pending attachment(s) left over from previous run", n)
	}

	if err := bootstrap(st, cfg); err != nil {
		log.Fatalf("bootstrap: %v", err)
	}

	hub := server.NewHub(st)
	tasks.StartCleanupWorker(st, cfg.Server.DataDir, hub.BroadcastConversationListChangedAll)
	tasks.StartPriceWorker(st, cfg)

	srv := server.New(cfg, st, hub)
	srv.StartTitleWorker()
	srv.ResumeResearchJobs()
	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	log.Printf("lemon-chat listening on http://localhost%s", addr)
	httpSrv := &http.Server{
		Addr:    addr,
		Handler: srv.Handler(),
		// Bound how long a client may take to send headers (slowloris). No
		// Read/WriteTimeout: SSE chat and research streams are long-lived.
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       2 * time.Minute,
	}
	if err := httpSrv.ListenAndServe(); err != nil {
		log.Fatalf("server: %v", err)
	}
}

// modelsInUse prints every model name the database references, with counts,
// marking names that no [[model]] in the config defines.
func modelsInUse(st *store.Store, cfg *config.Config) {
	usage, err := st.ModelNamesInUse()
	if err != nil {
		log.Fatalf("models-in-use: %v", err)
	}
	missing := 0
	for _, u := range usage {
		mark := "  "
		if cfg.ModelByName(u.Name) == nil {
			mark = "! "
			missing++
		}
		fmt.Printf("%s%-45s %4d  %v\n", mark, u.Name, u.Total, u.Counts)
	}
	if missing > 0 {
		fmt.Printf("\n! = not a configured model name (%d). Map them with --rename-model OLD NEW.\n", missing)
	}
}

// renameModels applies OLD NEW pairs to every stored model reference. NEW must
// be a configured model name, so a typo can't strand the rows.
func renameModels(st *store.Store, cfg *config.Config, args []string) {
	if len(args) == 0 || len(args)%2 != 0 {
		log.Fatalf("rename-model: expected OLD NEW pairs, got %d argument(s)", len(args))
	}
	for i := 0; i < len(args); i += 2 {
		if cfg.ModelByName(args[i+1]) == nil {
			log.Fatalf("rename-model: %q is not a model name in the config — add its [[model]] first", args[i+1])
		}
	}
	for i := 0; i < len(args); i += 2 {
		if _, err := st.RenameModel(args[i], args[i+1]); err != nil {
			log.Fatalf("rename-model: %v", err)
		}
	}
}

func listModels(cfg *config.Config) {
	// Open the store so oauth servers can resolve their shared token (the Codex
	// /models catalog needs it); non-oauth servers use their static api_key.
	var oauthProv *openai_auth.Provider
	if st, err := store.Open(cfg.Server.DBPath); err == nil {
		defer st.Close()
		oauthProv = openai_auth.NewProvider(openai_auth.NewStoreAdapter(st), http.DefaultClient)
	} else {
		fmt.Fprintf(os.Stderr, "warning: could not open store for oauth token: %v\n", err)
	}

	for i := range cfg.ModelServers {
		srv := &cfg.ModelServers[i]
		fmt.Printf("Model server: %s (%s)\n", srv.Name, srv.APIBase)
		token := config.StaticToken(srv.APIKey)
		accountID := ""
		if srv.UsesOAuth() && oauthProv != nil {
			token = oauthProv.Token
			accountID, _ = oauthProv.AccountID()
		}
		provider := llm.NewProvider(http.DefaultClient, srv, token, accountID)
		models, err := provider.ListModels(context.Background())
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

// openaiLogin runs the local-browser PKCE flow on this host and persists the
// shared token. Use this when lemon-chat runs where a browser is available; for
// a remote/LAN install, use the admin "Connect OpenAI account" paste-the-code UI.
func openaiLogin(st *store.Store) {
	provider := openai_auth.NewProvider(openai_auth.NewStoreAdapter(st), http.DefaultClient)
	tokens, err := openai_auth.Login(context.Background(), http.DefaultClient)
	if err != nil {
		log.Fatalf("openai-login: %v", err)
	}
	if err := provider.SetTokens(tokens); err != nil {
		log.Fatalf("openai-login: persisting token: %v", err)
	}
	acct := tokens.AccountID
	if acct == "" {
		acct = "(unknown)"
	}
	log.Printf("openai-login: linked OpenAI account %s — token stored", acct)
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
