package store

// ModelPrice is a per-token price for a model, in USD. Prices are populated by
// the background price worker from the model server's catalogue (currently
// OpenRouter) and cached here so the UI keeps its last-known figures when a
// refresh fails.
type ModelPrice struct {
	ModelID       string  `json:"model_id"`
	PromptUSD     float64 `json:"prompt_usd"`     // USD per prompt (input) token
	CompletionUSD float64 `json:"completion_usd"` // USD per completion (output) token
	UpdatedAt     string  `json:"updated_at"`
}

// UpsertModelPrice inserts or updates the cached price for modelID.
func (s *Store) UpsertModelPrice(modelID string, promptUSD, completionUSD float64) error {
	_, err := s.db.Exec(
		`INSERT INTO model_price (model_id, prompt_usd, completion_usd, updated_at)
		 VALUES (?, ?, ?, ?)
		 ON CONFLICT(model_id) DO UPDATE SET
		   prompt_usd = excluded.prompt_usd,
		   completion_usd = excluded.completion_usd,
		   updated_at = excluded.updated_at`,
		modelID, promptUSD, completionUSD, now())
	return err
}

// ModelPrices returns every cached price keyed by model ID.
func (s *Store) ModelPrices() (map[string]ModelPrice, error) {
	rows, err := s.db.Query(`SELECT model_id, prompt_usd, completion_usd, updated_at FROM model_price`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	prices := make(map[string]ModelPrice)
	for rows.Next() {
		var p ModelPrice
		if err := rows.Scan(&p.ModelID, &p.PromptUSD, &p.CompletionUSD, &p.UpdatedAt); err != nil {
			return nil, err
		}
		prices[p.ModelID] = p
	}
	return prices, rows.Err()
}
