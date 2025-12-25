package gui

import (
	"fmt"
	"time"

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

	// Clear history on server
	go s.app.SyncService.ClearHistory()

	return nil
}

// RecordConnect records a new tunnel connection to history
func (s *HistoryService) RecordConnect(bundleName, tunnelType string, localPort int, remoteAddr, url string) (*storage.HistoryEntry, error) {
	if s.app.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	repo := storage.NewHistoryRepository(s.app.db)
	entry := &storage.HistoryEntry{
		BundleName:  bundleName,
		TunnelType:  tunnelType,
		LocalPort:   localPort,
		RemoteAddr:  remoteAddr,
		URL:         url,
		ConnectedAt: time.Now(),
	}

	if err := repo.RecordConnect(entry); err != nil {
		return nil, err
	}

	s.log.Debug().
		Str("bundle", bundleName).
		Str("type", tunnelType).
		Int("local_port", localPort).
		Msg("Connection recorded")

	// Push history entry to server
	go s.app.SyncService.PushHistoryEntry(entry)

	return entry, nil
}

// RecordDisconnect updates a history entry with disconnect time and stats
func (s *HistoryService) RecordDisconnect(entryID int64, bytesSent, bytesReceived int64) error {
	if s.app.db == nil {
		return fmt.Errorf("database not initialized")
	}

	repo := storage.NewHistoryRepository(s.app.db)
	if err := repo.RecordDisconnect(entryID, bytesSent, bytesReceived); err != nil {
		return err
	}

	s.log.Debug().
		Int64("entry_id", entryID).
		Int64("bytes_sent", bytesSent).
		Int64("bytes_received", bytesReceived).
		Msg("Disconnect recorded")

	return nil
}
