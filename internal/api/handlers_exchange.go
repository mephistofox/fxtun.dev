package api

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"
)

// Exchange rate cache
var (
	exchangeRateCache     float64
	exchangeRateCacheTime time.Time
	exchangeRateMu        sync.RWMutex
	exchangeRateCacheTTL  = time.Hour
	exchangeRateFallback  = 75.0
)

type exchangeRateAPIResponse struct {
	Rates map[string]float64 `json:"rates"`
}

type exchangeRateResponse struct {
	Rate      float64 `json:"rate"`
	UpdatedAt int64   `json:"updated_at"`
}

func (s *Server) handleExchangeRate(w http.ResponseWriter, r *http.Request) {
	rate, updatedAt := s.getExchangeRate()
	s.respondJSON(w, http.StatusOK, exchangeRateResponse{
		Rate:      rate,
		UpdatedAt: updatedAt,
	})
}

func (s *Server) getExchangeRate() (float64, int64) {
	exchangeRateMu.RLock()
	if time.Since(exchangeRateCacheTime) < exchangeRateCacheTTL && exchangeRateCache > 0 {
		rate := exchangeRateCache
		updatedAt := exchangeRateCacheTime.Unix()
		exchangeRateMu.RUnlock()
		return rate, updatedAt
	}
	exchangeRateMu.RUnlock()

	// Fetch new rate
	rate := s.fetchExchangeRate()

	exchangeRateMu.Lock()
	exchangeRateCache = rate
	exchangeRateCacheTime = time.Now()
	exchangeRateMu.Unlock()

	return rate, exchangeRateCacheTime.Unix()
}

func (s *Server) fetchExchangeRate() float64 {
	// Try primary API
	rate, err := s.fetchFromExchangeRateAPI()
	if err == nil {
		return rate
	}
	s.log.Warn().Err(err).Msg("Failed to fetch exchange rate from primary API")

	// Try alternative API
	rate, err = s.fetchFromOpenExchangeAPI()
	if err == nil {
		return rate
	}
	s.log.Warn().Err(err).Msg("Failed to fetch exchange rate from alternative API")

	// Return cached rate if available, otherwise fallback
	exchangeRateMu.RLock()
	if exchangeRateCache > 0 {
		rate = exchangeRateCache
		exchangeRateMu.RUnlock()
		return rate
	}
	exchangeRateMu.RUnlock()

	return exchangeRateFallback
}

func (s *Server) fetchFromExchangeRateAPI() (float64, error) {
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get("https://api.exchangerate-api.com/v4/latest/USD")
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	var data exchangeRateAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return 0, err
	}

	rate, ok := data.Rates["RUB"]
	if !ok || rate <= 0 {
		return 0, err
	}

	return rate, nil
}

func (s *Server) fetchFromOpenExchangeAPI() (float64, error) {
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get("https://open.er-api.com/v6/latest/USD")
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	var data exchangeRateAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return 0, err
	}

	rate, ok := data.Rates["RUB"]
	if !ok || rate <= 0 {
		return 0, err
	}

	return rate, nil
}
