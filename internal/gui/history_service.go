package gui

import (
	"fmt"

	"github.com/rs/zerolog"

	"github.com/mephistofox/fxtunnel/internal/storage"
)

// HistoryService handles connection history operations
type HistoryService struct {
	app *App
	log zerolog.Logger
}

// NewHistoryService creates a new history service
func NewHistoryService(app *App) *HistoryService {
	return &HistoryService{
		app: app,
		log: app.log.With().Str("service", "history").Logger(),
	}
}

// HistoryResult represents paginated history results
type HistoryResult struct {
	Entries []storage.HistoryEntry `json:"entries"`
	Total   int                    `json:"total"`
}

// List returns history entries with pagination
func (s *HistoryService) List(limit, offset int) (*HistoryResult, error) {
	if s.app.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	if limit <= 0 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}

	repo := storage.NewHistoryRepository(s.app.db)
	entries, total, err := repo.List(limit, offset)
	if err != nil {
		return nil, err
	}

	return &HistoryResult{
		Entries: entries,
		Total:   total,
	}, nil
}

// GetRecent returns the most recent history entries
func (s *HistoryService) GetRecent(limit int) ([]storage.HistoryEntry, error) {
	if s.app.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	if limit <= 0 {
		limit = 10
	}

	repo := storage.NewHistoryRepository(s.app.db)
	return repo.GetRecent(limit)
}

// Clear deletes all history entries
func (s *HistoryService) Clear() error {
	if s.app.db == nil {
		return fmt.Errorf("database not initialized")
	}

	repo := storage.NewHistoryRepository(s.app.db)
	if err := repo.Clear(); err != nil {
		return err
	}

	s.log.Info().Msg("History cleared")
	return nil
}
