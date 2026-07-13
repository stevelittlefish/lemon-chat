package tasks

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/stevelittlefish/lemon-chat/internal/config"
	"github.com/stevelittlefish/lemon-chat/internal/store"
)

// priceInterval is how often the price worker refreshes model pricing.
const priceInterval = 24 * time.Hour

// priceFetchTimeout caps a single catalogue fetch.
const priceFetchTimeout = 30 * time.Second

// maxPriceResponseBytes caps the OpenRouter catalogue read.
const maxPriceResponseBytes = 20 * 1024 * 1024

// StartPriceWorker refreshes cached model prices once at startup and then daily.
// It only runs when at least one model is hosted on an OpenRouter server; with
// no priced models it logs once and exits, leaving no ticker running.
func StartPriceWorker(st *store.Store, cfg *config.Config) {
	if !hasOpenRouterModel(cfg) {
		log.Println("price worker: no OpenRouter-backed models configured, skipping")
		return
	}
	go func() {
		refreshModelPrices(st, cfg)
		ticker := time.NewTicker(priceInterval)
		defer ticker.Stop()
		for range ticker.C {
			refreshModelPrices(st, cfg)
		}
	}()
}

func hasOpenRouterModel(cfg *config.Config) bool {
	for i := range cfg.Models {
		if srv := serverFor(cfg, cfg.Models[i].ModelServer); srv != nil && srv.IsOpenRouter() {
			return true
		}
	}
	return false
}

func serverFor(cfg *config.Config, name string) *config.ModelServer {
	for i := range cfg.ModelServers {
		if cfg.ModelServers[i].Name == name {
			return &cfg.ModelServers[i]
		}
	}
	return nil
}

// refreshModelPrices fetches the OpenRouter catalogue and upserts a price row
// for every configured model hosted on an OpenRouter server that appears in it.
func refreshModelPrices(st *store.Store, cfg *config.Config) {
	// Collect the OpenRouter-backed model IDs we care about, and a base URL to
	// fetch the catalogue from (the same for every OpenRouter server).
	wanted := make(map[string]bool)
	base := ""
	for i := range cfg.Models {
		srv := serverFor(cfg, cfg.Models[i].ModelServer)
		if srv == nil || !srv.IsOpenRouter() {
			continue
		}
		wanted[cfg.Models[i].Name] = true
		if base == "" {
			base = strings.TrimRight(srv.APIBase, "/")
		}
	}
	if len(wanted) == 0 {
		return
	}

	catalogue, err := fetchOpenRouterPrices(base + "/models")
	if err != nil {
		log.Printf("price worker: fetch failed, keeping cached prices: %v", err)
		return
	}

	matched := 0
	for id := range wanted {
		p, ok := catalogue[id]
		if !ok {
			continue
		}
		if err := st.UpsertModelPrice(id, p.prompt, p.completion); err != nil {
			log.Printf("price worker: could not save price for %q: %v", id, err)
			continue
		}
		matched++
	}
	log.Printf("price worker: refreshed %d of %d OpenRouter-backed model price(s)", matched, len(wanted))
}

type tokenPrice struct {
	prompt     float64
	completion float64
}

// fetchOpenRouterPrices GETs the OpenRouter model catalogue and returns a map of
// model ID → per-token prices. Prices in the catalogue are strings in USD per
// token; entries that fail to parse are skipped.
func fetchOpenRouterPrices(url string) (map[string]tokenPrice, error) {
	ctx, cancel := context.WithTimeout(context.Background(), priceFetchTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxPriceResponseBytes))
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("catalogue returned %d: %.200s", resp.StatusCode, body)
	}

	var parsed struct {
		Data []struct {
			ID      string `json:"id"`
			Pricing struct {
				Prompt     string `json:"prompt"`
				Completion string `json:"completion"`
			} `json:"pricing"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("decode catalogue: %w", err)
	}

	prices := make(map[string]tokenPrice, len(parsed.Data))
	for _, m := range parsed.Data {
		prompt, err1 := strconv.ParseFloat(m.Pricing.Prompt, 64)
		completion, err2 := strconv.ParseFloat(m.Pricing.Completion, 64)
		if err1 != nil || err2 != nil {
			continue
		}
		prices[m.ID] = tokenPrice{prompt: prompt, completion: completion}
	}
	return prices, nil
}
