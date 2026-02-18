package server

import (
	"strings"

	"github.com/sethvargo/go-diceware/diceware"
)

// generateUniqueSubdomain tries 1-word subdomain first, then 2-word on collision.
func (s *Server) generateUniqueSubdomain() string {
	// Try 1 word (7776 options)
	for i := 0; i < 5; i++ {
		candidate := generateWords(1)
		if s.httpRouter.GetTunnel(candidate) == nil {
			return candidate
		}
	}
	// Collisions on 1 word — use 2 words (60M options)
	for i := 0; i < 5; i++ {
		candidate := generateWords(2)
		if s.httpRouter.GetTunnel(candidate) == nil {
			return candidate
		}
	}
	// Should never happen, but fallback to hex
	return generateShortID()
}

// generateWords returns n random EFF diceware words (3-7 chars each) joined by hyphens.
func generateWords(n int) string {
	result := make([]string, 0, n)
	for len(result) < n {
		words, err := diceware.Generate(1)
		if err != nil {
			return generateShortID()
		}
		w := strings.ToLower(words[0])
		if len(w) >= 3 && len(w) <= 7 {
			result = append(result, w)
		}
	}
	return strings.Join(result, "-")
}
