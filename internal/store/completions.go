package store

type Completion struct {
	ID               int64   `json:"id"`
	UserID           int64   `json:"user_id"`
	Model            string  `json:"model"`
	Title            *string `json:"title"`
	Content          *string `json:"content"`
	PrevContent      *string `json:"prev_content"`
	PromptTokens     *int64  `json:"prompt_tokens"`
	CompletionTokens *int64  `json:"completion_tokens"`
	CreatedAt        string  `json:"created_at"`
	UpdatedAt        string  `json:"updated_at"`
}

func (s *Store) ListCompletions(userID int64) ([]Completion, error) {
	rows, err := s.db.Query(
		`SELECT id, user_id, model, title, content, prev_content, prompt_tokens, completion_tokens, created_at, updated_at
		 FROM completion WHERE user_id = ? ORDER BY updated_at DESC`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []Completion
	for rows.Next() {
		var c Completion
		if err := rows.Scan(&c.ID, &c.UserID, &c.Model, &c.Title, &c.Content, &c.PrevContent, &c.PromptTokens, &c.CompletionTokens, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, c)
	}
	return items, rows.Err()
}

func (s *Store) GetCompletion(id, userID int64) (*Completion, error) {
	var c Completion
	err := s.db.QueryRow(
		`SELECT id, user_id, model, title, content, prev_content, prompt_tokens, completion_tokens, created_at, updated_at
		 FROM completion WHERE id = ? AND user_id = ?`,
		id, userID,
	).Scan(&c.ID, &c.UserID, &c.Model, &c.Title, &c.Content, &c.PrevContent, &c.PromptTokens, &c.CompletionTokens, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		return nil, ErrNotFound
	}
	return &c, nil
}

func (s *Store) CreateCompletion(userID int64, model string) (*Completion, error) {
	t := now()
	res, err := s.db.Exec(
		`INSERT INTO completion (user_id, model, created_at, updated_at) VALUES (?, ?, ?, ?)`,
		userID, model, t, t,
	)
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	return &Completion{ID: id, UserID: userID, Model: model, CreatedAt: t, UpdatedAt: t}, nil
}

func (s *Store) UpdateCompletionTitle(id, userID int64, title string) error {
	res, err := s.db.Exec(
		`UPDATE completion SET title = ?, updated_at = ? WHERE id = ? AND user_id = ?`,
		title, now(), id, userID,
	)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) UpdateCompletionContent(id, userID int64, content string) error {
	res, err := s.db.Exec(
		`UPDATE completion SET content = ?, updated_at = ? WHERE id = ? AND user_id = ?`,
		content, now(), id, userID,
	)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) UpdateCompletionModel(id, userID int64, model string) error {
	res, err := s.db.Exec(
		`UPDATE completion SET model = ?, updated_at = ? WHERE id = ? AND user_id = ?`,
		model, now(), id, userID,
	)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) DeleteCompletion(id, userID int64) error {
	res, err := s.db.Exec(`DELETE FROM completion WHERE id = ? AND user_id = ?`, id, userID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

// SetPrevContent saves a snapshot of the current content before a run starts.
func (s *Store) SetPrevContent(id, userID int64, prevContent string) error {
	res, err := s.db.Exec(
		`UPDATE completion SET prev_content = ? WHERE id = ? AND user_id = ?`,
		prevContent, id, userID,
	)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

// UpdateTokenCounts saves prompt and completion token counts after a run.
func (s *Store) UpdateTokenCounts(id, userID int64, promptTokens, completionTokens int64) error {
	res, err := s.db.Exec(
		`UPDATE completion SET prompt_tokens = ?, completion_tokens = ? WHERE id = ? AND user_id = ?`,
		promptTokens, completionTokens, id, userID,
	)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

// UndoCompletion atomically restores content from prev_content, clears prev_content,
// and returns the restored content. Returns ErrNotFound if no snapshot exists.
func (s *Store) UndoCompletion(id, userID int64) (string, error) {
	var prevContent *string
	err := s.db.QueryRow(
		`SELECT prev_content FROM completion WHERE id = ? AND user_id = ?`,
		id, userID,
	).Scan(&prevContent)
	if err != nil || prevContent == nil {
		return "", ErrNotFound
	}
	content := *prevContent
	if _, err := s.db.Exec(
		`UPDATE completion SET content = ?, prev_content = NULL, updated_at = ? WHERE id = ? AND user_id = ?`,
		content, now(), id, userID,
	); err != nil {
		return "", err
	}
	return content, nil
}
