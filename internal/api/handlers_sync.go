package api

import (
	"net/http"

	"github.com/mephistofox/fxtunnel/internal/api/dto"
	"github.com/mephistofox/fxtunnel/internal/auth"
	"github.com/mephistofox/fxtunnel/internal/database"
)

const maxSyncItems = 500

// handleGetSyncData returns all sync data for the user
func (s *Server) handleGetSyncData(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromContext(r.Context())
	if user == nil {
		s.respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	// Get bundles
	bundles, err := s.db.UserBundles.GetByUserID(user.ID)
	if err != nil {
		s.log.Error().Err(err).Msg("Failed to get user bundles")
		s.respondError(w, http.StatusInternalServerError, "failed to get bundles")
		return
	}

	// Get history (last 100 entries)
	history, err := s.db.UserHistory.GetRecent(user.ID, 100)
	if err != nil {
		s.log.Error().Err(err).Msg("Failed to get user history")
		s.respondError(w, http.StatusInternalServerError, "failed to get history")
		return
	}

	// Get settings
	settings, err := s.db.UserSettings.GetAllWithTimestamps(user.ID)
	if err != nil {
		s.log.Error().Err(err).Msg("Failed to get user settings")
		s.respondError(w, http.StatusInternalServerError, "failed to get settings")
		return
	}

	// Convert to DTOs
	bundleDTOs := make([]dto.BundleDTO, len(bundles))
	for i, b := range bundles {
		bundleDTOs[i] = dto.BundleDTOFromModel(b)
	}

	historyDTOs := make([]dto.HistoryDTO, len(history))
	for i, h := range history {
		historyDTOs[i] = dto.HistoryDTOFromModel(h)
	}

	settingDTOs := make([]dto.SettingDTO, len(settings))
	for i, st := range settings {
		settingDTOs[i] = dto.SettingDTOFromModel(st)
	}

	s.respondJSON(w, http.StatusOK, dto.SyncResponse{
		Bundles:  bundleDTOs,
		History:  historyDTOs,
		Settings: settingDTOs,
	})
}

// handleSync performs a full sync (receive client data, return merged server data)
func (s *Server) handleSync(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromContext(r.Context())
	if user == nil {
		s.respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req dto.SyncRequest
	if err := s.decodeJSON(r, &req); err != nil {
		s.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if len(req.Bundles) > maxSyncItems || len(req.History) > maxSyncItems || len(req.Settings) > maxSyncItems {
		s.respondError(w, http.StatusRequestEntityTooLarge, "too many items in sync request")
		return
	}

	// Sync bundles
	if len(req.Bundles) > 0 {
		bundles := make([]*database.UserBundle, 0, len(req.Bundles))
		for _, b := range req.Bundles {
			if b.Deleted {
				// Handle deletion
				if err := s.db.UserBundles.DeleteByName(user.ID, b.Name); err != nil && err != database.ErrBundleNotFound {
					s.log.Error().Err(err).Str("name", b.Name).Msg("Failed to delete bundle")
				}
				continue
			}
			bundles = append(bundles, b.ToUserBundle(user.ID))
		}

		if len(bundles) > 0 {
			if err := s.db.UserBundles.SyncBulk(user.ID, bundles); err != nil {
				s.log.Error().Err(err).Msg("Failed to sync bundles")
				s.respondError(w, http.StatusInternalServerError, "failed to sync bundles")
				return
			}
		}
	}

	// Sync history (just add new entries)
	if len(req.History) > 0 {
		entries := make([]*database.UserHistoryEntry, len(req.History))
		for i, h := range req.History {
			entries[i] = h.ToUserHistoryEntry(user.ID)
		}

		if err := s.db.UserHistory.AddBulk(user.ID, entries); err != nil {
			s.log.Error().Err(err).Msg("Failed to sync history")
			s.respondError(w, http.StatusInternalServerError, "failed to sync history")
			return
		}
	}

	// Sync settings
	if len(req.Settings) > 0 {
		settings := make([]*database.UserSetting, len(req.Settings))
		for i, st := range req.Settings {
			settings[i] = st.ToUserSetting(user.ID)
		}

		if err := s.db.UserSettings.SyncBulk(user.ID, settings); err != nil {
			s.log.Error().Err(err).Msg("Failed to sync settings")
			s.respondError(w, http.StatusInternalServerError, "failed to sync settings")
			return
		}
	}

	// Return current server state
	s.handleGetSyncData(w, r)
}

// handleSyncBundles syncs only bundles
func (s *Server) handleSyncBundles(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromContext(r.Context())
	if user == nil {
		s.respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req dto.SyncBundlesRequest
	if err := s.decodeJSON(r, &req); err != nil {
		s.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if len(req.Bundles) > maxSyncItems {
		s.respondError(w, http.StatusRequestEntityTooLarge, "too many items in sync request")
		return
	}

	bundles := make([]*database.UserBundle, 0, len(req.Bundles))
	for _, b := range req.Bundles {
		if b.Deleted {
			if err := s.db.UserBundles.DeleteByName(user.ID, b.Name); err != nil && err != database.ErrBundleNotFound {
				s.log.Error().Err(err).Str("name", b.Name).Msg("Failed to delete bundle")
			}
			continue
		}
		bundles = append(bundles, b.ToUserBundle(user.ID))
	}

	if len(bundles) > 0 {
		if err := s.db.UserBundles.SyncBulk(user.ID, bundles); err != nil {
			s.log.Error().Err(err).Msg("Failed to sync bundles")
			s.respondError(w, http.StatusInternalServerError, "failed to sync bundles")
			return
		}
	}

	// Return updated bundles
	serverBundles, err := s.db.UserBundles.GetByUserID(user.ID)
	if err != nil {
		s.log.Error().Err(err).Msg("Failed to get bundles")
		s.respondError(w, http.StatusInternalServerError, "failed to get bundles")
		return
	}

	bundleDTOs := make([]dto.BundleDTO, len(serverBundles))
	for i, b := range serverBundles {
		bundleDTOs[i] = dto.BundleDTOFromModel(b)
	}

	s.respondJSON(w, http.StatusOK, map[string]interface{}{
		"bundles": bundleDTOs,
	})
}

// handleSyncSettings syncs only settings
func (s *Server) handleSyncSettings(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromContext(r.Context())
	if user == nil {
		s.respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req dto.SyncSettingsRequest
	if err := s.decodeJSON(r, &req); err != nil {
		s.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if len(req.Settings) > maxSyncItems {
		s.respondError(w, http.StatusRequestEntityTooLarge, "too many items in sync request")
		return
	}

	if len(req.Settings) > 0 {
		settings := make([]*database.UserSetting, len(req.Settings))
		for i, st := range req.Settings {
			settings[i] = st.ToUserSetting(user.ID)
		}

		if err := s.db.UserSettings.SyncBulk(user.ID, settings); err != nil {
			s.log.Error().Err(err).Msg("Failed to sync settings")
			s.respondError(w, http.StatusInternalServerError, "failed to sync settings")
			return
		}
	}

	// Return updated settings
	serverSettings, err := s.db.UserSettings.GetAllWithTimestamps(user.ID)
	if err != nil {
		s.log.Error().Err(err).Msg("Failed to get settings")
		s.respondError(w, http.StatusInternalServerError, "failed to get settings")
		return
	}

	settingDTOs := make([]dto.SettingDTO, len(serverSettings))
	for i, st := range serverSettings {
		settingDTOs[i] = dto.SettingDTOFromModel(st)
	}

	s.respondJSON(w, http.StatusOK, map[string]interface{}{
		"settings": settingDTOs,
	})
}

// handleAddHistory adds new history entries
func (s *Server) handleAddHistory(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromContext(r.Context())
	if user == nil {
		s.respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req dto.SyncHistoryRequest
	if err := s.decodeJSON(r, &req); err != nil {
		s.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if len(req.History) > maxSyncItems {
		s.respondError(w, http.StatusRequestEntityTooLarge, "too many items in sync request")
		return
	}

	if len(req.History) == 0 {
		s.respondJSON(w, http.StatusOK, map[string]interface{}{
			"added": 0,
		})
		return
	}

	entries := make([]*database.UserHistoryEntry, len(req.History))
	for i, h := range req.History {
		entries[i] = h.ToUserHistoryEntry(user.ID)
	}

	if err := s.db.UserHistory.AddBulk(user.ID, entries); err != nil {
		s.log.Error().Err(err).Msg("Failed to add history")
		s.respondError(w, http.StatusInternalServerError, "failed to add history")
		return
	}

	s.respondJSON(w, http.StatusOK, map[string]interface{}{
		"added": len(entries),
	})
}

// handleClearHistory clears all history for the user
func (s *Server) handleClearHistory(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromContext(r.Context())
	if user == nil {
		s.respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	if err := s.db.UserHistory.Clear(user.ID); err != nil {
		s.log.Error().Err(err).Msg("Failed to clear history")
		s.respondError(w, http.StatusInternalServerError, "failed to clear history")
		return
	}

	s.respondJSON(w, http.StatusOK, dto.SuccessResponse{
		Success: true,
		Message: "history cleared",
	})
}

// handleGetHistoryStats returns history statistics
func (s *Server) handleGetHistoryStats(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromContext(r.Context())
	if user == nil {
		s.respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	stats, err := s.db.UserHistory.GetStats(user.ID)
	if err != nil {
		s.log.Error().Err(err).Msg("Failed to get history stats")
		s.respondError(w, http.StatusInternalServerError, "failed to get history stats")
		return
	}

	s.respondJSON(w, http.StatusOK, dto.HistoryStatsDTO{
		TotalConnections:   stats.TotalConnections,
		TotalBytesSent:     stats.TotalBytesSent,
		TotalBytesReceived: stats.TotalBytesReceived,
	})
}
