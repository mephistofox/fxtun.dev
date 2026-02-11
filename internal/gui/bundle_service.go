package gui

import (
	"encoding/json"
	"fmt"

	"github.com/rs/zerolog"

	"github.com/mephistofox/fxtun.dev/internal/storage"
)

// BundleService handles bundle operations
type BundleService struct {
	app *App
	log zerolog.Logger
}

// NewBundleService creates a new bundle service
func NewBundleService(app *App) *BundleService {
	return &BundleService{
		app: app,
		log: app.log.With().Str("service", "bundle").Logger(),
	}
}

// List returns all bundles
func (s *BundleService) List() ([]storage.Bundle, error) {
	if s.app.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	repo := storage.NewBundleRepository(s.app.db)
	return repo.List()
}

// GetByID returns a bundle by ID
func (s *BundleService) GetByID(id int64) (*storage.Bundle, error) {
	if s.app.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	repo := storage.NewBundleRepository(s.app.db)
	return repo.GetByID(id)
}

// Create creates a new bundle
func (s *BundleService) Create(bundle *storage.Bundle) (*storage.Bundle, error) {
	if s.app.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	repo := storage.NewBundleRepository(s.app.db)
	if err := repo.Create(bundle); err != nil {
		return nil, err
	}

	s.log.Info().
		Int64("id", bundle.ID).
		Str("name", bundle.Name).
		Msg("Bundle created")

	// Sync bundles to server
	go s.app.SyncService.SyncBundles()

	return bundle, nil
}

// Update updates an existing bundle
func (s *BundleService) Update(bundle *storage.Bundle) error {
	if s.app.db == nil {
		return fmt.Errorf("database not initialized")
	}

	repo := storage.NewBundleRepository(s.app.db)
	if err := repo.Update(bundle); err != nil {
		return err
	}

	s.log.Info().
		Int64("id", bundle.ID).
		Str("name", bundle.Name).
		Msg("Bundle updated")

	// Sync bundles to server
	go s.app.SyncService.SyncBundles()

	return nil
}

// Delete deletes a bundle
func (s *BundleService) Delete(id int64) error {
	if s.app.db == nil {
		return fmt.Errorf("database not initialized")
	}

	repo := storage.NewBundleRepository(s.app.db)
	if err := repo.Delete(id); err != nil {
		return err
	}

	s.log.Info().Int64("id", id).Msg("Bundle deleted")

	// Sync bundles to server
	go s.app.SyncService.SyncBundles()

	return nil
}

// Connect creates a tunnel from a bundle
func (s *BundleService) Connect(id int64) (*TunnelInfo, error) {
	bundle, err := s.GetByID(id)
	if err != nil {
		return nil, err
	}
	if bundle == nil {
		return nil, fmt.Errorf("bundle not found")
	}

	cfg := TunnelConfig{
		Name:       bundle.Name,
		Type:       bundle.Type,
		LocalPort:  bundle.LocalPort,
		Subdomain:  bundle.Subdomain,
		RemotePort: bundle.RemotePort,
	}

	return s.app.TunnelService.CreateTunnel(cfg)
}

// ConnectAutoStart connects all bundles marked for auto-connect
func (s *BundleService) ConnectAutoStart() ([]TunnelInfo, error) {
	if s.app.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	repo := storage.NewBundleRepository(s.app.db)
	bundles, err := repo.GetAutoConnect()
	if err != nil {
		return nil, err
	}

	var tunnels []TunnelInfo
	for _, bundle := range bundles {
		tunnel, err := s.Connect(bundle.ID)
		if err != nil {
			s.log.Error().Err(err).Str("bundle", bundle.Name).Msg("Failed to auto-connect bundle")
			continue
		}
		tunnels = append(tunnels, *tunnel)
	}

	return tunnels, nil
}

// Export exports all bundles as JSON
func (s *BundleService) Export() (string, error) {
	bundles, err := s.List()
	if err != nil {
		return "", err
	}

	data, err := json.MarshalIndent(bundles, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshal bundles: %w", err)
	}

	return string(data), nil
}

// Import imports bundles from JSON
func (s *BundleService) Import(jsonData string) error {
	if s.app.db == nil {
		return fmt.Errorf("database not initialized")
	}

	var bundles []storage.Bundle
	if err := json.Unmarshal([]byte(jsonData), &bundles); err != nil {
		return fmt.Errorf("parse bundles: %w", err)
	}

	repo := storage.NewBundleRepository(s.app.db)
	for _, bundle := range bundles {
		bundle.ID = 0 // Reset ID for new creation
		if err := repo.Create(&bundle); err != nil {
			s.log.Error().Err(err).Str("name", bundle.Name).Msg("Failed to import bundle")
		}
	}

	s.log.Info().Int("count", len(bundles)).Msg("Bundles imported")
	return nil
}
