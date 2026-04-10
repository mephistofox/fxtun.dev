package exchange

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
)

const (
	defaultRate     = 80.0
	cbrAPIURL       = "https://www.cbr-xml-daily.ru/daily_json.js"
	refreshInterval = 6 * time.Hour
	requestTimeout  = 10 * time.Second
)

var (
	mu           sync.RWMutex
	currentRate  float64
	fallbackRate float64
	lastUpdated  time.Time
	stopCh       chan struct{}
)

// Init sets the fallback USD to RUB exchange rate from config,
// fetches the live rate from CBR API, and starts a background
// refresh goroutine that updates the rate every 6 hours.
func Init(configRate float64) {
	mu.Lock()
	if configRate > 0 {
		fallbackRate = configRate
	} else {
		fallbackRate = defaultRate
	}
	currentRate = fallbackRate
	mu.Unlock()

	// Try to fetch live rate immediately
	if r, err := fetchCBRRate(); err == nil {
		mu.Lock()
		currentRate = r
		lastUpdated = time.Now()
		mu.Unlock()
		log.Info().Float64("rate", r).Msg("Exchange rate fetched from CBR")
	} else {
		log.Warn().Err(err).Float64("fallback_rate", fallbackRate).
			Msg("Failed to fetch CBR rate, using config fallback")
	}

	// Start background refresh
	stopCh = make(chan struct{})
	go refreshLoop()
}

// Stop terminates the background rate refresh goroutine.
func Stop() {
	if stopCh != nil {
		close(stopCh)
	}
}

// GetRate returns the current USD to RUB exchange rate and the
// unix timestamp of the last successful CBR update (0 if never fetched).
func GetRate() (float64, int64) {
	mu.RLock()
	defer mu.RUnlock()

	rate := currentRate
	if rate <= 0 {
		rate = defaultRate
	}

	var ts int64
	if !lastUpdated.IsZero() {
		ts = lastUpdated.Unix()
	}

	return rate, ts
}

// ConvertUSDToRUB converts USD amount to RUB with nice rounding (to nearest 5).
func ConvertUSDToRUB(usd float64) float64 {
	r, _ := GetRate()
	return roundToNearest5(usd * r)
}

func roundToNearest5(n float64) float64 {
	return float64(int((n+2.5)/5) * 5)
}

// cbrResponse represents the relevant fields from the CBR daily JSON API.
type cbrResponse struct {
	Valute struct {
		USD struct {
			Value float64 `json:"Value"`
		} `json:"USD"`
	} `json:"Valute"`
}

func fetchCBRRate() (float64, error) {
	client := &http.Client{Timeout: requestTimeout}

	resp, err := client.Get(cbrAPIURL)
	if err != nil {
		return 0, fmt.Errorf("fetch CBR rate: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("CBR API returned status %d", resp.StatusCode)
	}

	var data cbrResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return 0, fmt.Errorf("parse CBR response: %w", err)
	}

	if data.Valute.USD.Value <= 0 {
		return 0, fmt.Errorf("invalid USD rate from CBR: %f", data.Valute.USD.Value)
	}

	return data.Valute.USD.Value, nil
}

func refreshLoop() {
	ticker := time.NewTicker(refreshInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if r, err := fetchCBRRate(); err == nil {
				mu.Lock()
				currentRate = r
				lastUpdated = time.Now()
				mu.Unlock()
				log.Debug().Float64("rate", r).Msg("Exchange rate refreshed from CBR")
			} else {
				log.Warn().Err(err).Msg("Failed to refresh CBR exchange rate")
			}
		case <-stopCh:
			return
		}
	}
}
