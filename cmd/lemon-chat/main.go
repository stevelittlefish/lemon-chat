package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"

	"github.com/stevelittlefish/lemon-chat/internal/config"
	"github.com/stevelittlefish/lemon-chat/internal/server"
	"github.com/stevelittlefish/lemon-chat/internal/store"
	"github.com/stevelittlefish/lemon-chat/internal/tasks"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	cfgPath := flag.String("config", "lemon.toml", "path to config file")
	flag.Parse()

	cfg, err := config.Load(*cfgPath)
	if err != nil {
		log.Fatalf("load config: %v", err)
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

	srv := server.New(cfg, st, hub)
	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	log.Printf("lemon-chat listening on http://localhost%s", addr)
	if err := http.ListenAndServe(addr, srv.Handler()); err != nil {
		log.Fatalf("server: %v", err)
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
