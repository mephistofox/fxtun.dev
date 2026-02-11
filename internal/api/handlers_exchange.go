package api

import (
	"net/http"

	"github.com/mephistofox/fxtun.dev/internal/exchange"
)

type exchangeRateResponse struct {
	Rate      float64 `json:"rate"`
	UpdatedAt int64   `json:"updated_at"`
}

func (s *Server) handleExchangeRate(w http.ResponseWriter, r *http.Request) {
	rate, updatedAt := exchange.GetRate()
	s.respondJSON(w, http.StatusOK, exchangeRateResponse{
		Rate:      rate,
		UpdatedAt: updatedAt,
	})
}
