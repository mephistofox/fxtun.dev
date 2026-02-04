package exchange

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/rs/zerolog"
)

const (
	// FallbackRate is the default USD to RUB rate when APIs are unavailable
	FallbackRate = 75.0
	// CacheTTL is how long to cache the exchange rate
	CacheTTL = time.Hour
)

var (
	rateCache     float64
	rateCacheTime time.Time
	rateMu        sync.RWMutex
	globalService *Service
)

type apiResponse struct {
	Rates map[string]float64 `json:"rates"`
}

// Service provides exchange rate functionality
type Service struct {
	log zerolog.Logger
}

// New creates a new exchange rate service
func New(log zerolog.Logger) *Service {
	s := &Service{log: log}
	globalService = s
	return s
}

// GetRate returns the current USD to RUB exchange rate
func (s *Service) GetRate() (float64, int64) {
	rateMu.RLock()
	if time.Since(rateCacheTime) < CacheTTL && rateCache > 0 {
		rate := rateCache
		updatedAt := rateCacheTime.Unix()
		rateMu.RUnlock()
		return rate, updatedAt
	}
	rateMu.RUnlock()

	rate := s.fetchRate()

	rateMu.Lock()
	rateCache = rate
	rateCacheTime = time.Now()
	rateMu.Unlock()

	return rate, rateCacheTime.Unix()
}

// GetRate returns the exchange rate using the global service
func GetRate() (float64, int64) {
	if globalService != nil {
		return globalService.GetRate()
	}
	return FallbackRate, time.Now().Unix()
}

func (s *Service) fetchRate() float64 {
	// Try primary API
	rate, err := fetchFromAPI("https://api.exchangerate-api.com/v4/latest/USD")
	if err == nil {
		return rate
	}
	if s.log.Debug().Enabled() {
		s.log.Warn().Err(err).Msg("Failed to fetch exchange rate from primary API")
	}

	// Try alternative API
	rate, err = fetchFromAPI("https://open.er-api.com/v6/latest/USD")
	if err == nil {
		return rate
	}
	if s.log.Debug().Enabled() {
		s.log.Warn().Err(err).Msg("Failed to fetch exchange rate from alternative API")
	}

	// Return cached rate if available
	rateMu.RLock()
	if rateCache > 0 {
		rate = rateCache
		rateMu.RUnlock()
		return rate
	}
	rateMu.RUnlock()

	return FallbackRate
}

func fetchFromAPI(url string) (float64, error) {
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	var data apiResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return 0, err
	}

	rate, ok := data.Rates["RUB"]
	if !ok || rate <= 0 {
		return 0, err
	}

	return rate, nil
}

// ConvertUSDToRUB converts USD amount to RUB with nice rounding (to nearest 5)
func ConvertUSDToRUB(usd float64) float64 {
	rate, _ := GetRate()
	rub := usd * rate
	// Round to nearest 5 for nice pricing (e.g., 746.25 -> 745)
	return roundToNearest5(rub)
}

// roundToNearest5 rounds a number to the nearest multiple of 5
func roundToNearest5(n float64) float64 {
	return float64(int((n+2.5)/5) * 5)
}
