package api

import (
	"net/http"
	"time"
)

// handleIndex handles the index route
func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	response := Response{
		Data: map[string]interface{}{
			"version": "1.0.0",
			"name":    "COTA API",
			"time":    time.Now().Format(time.RFC3339),
		},
		Links: map[string]string{
			"routes":      "/routes",
			"stops":       "/stops",
			"trips":       "/trips",
			"vehicles":    "/vehicles",
			"predictions": "/predictions",
			"shapes":      "/shapes",
		},
	}

	s.sendResponse(w, response)
}