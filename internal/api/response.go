package api

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
)

// Resource represents a JSON:API resource object
type Resource struct {
	Type          string                  `json:"type"`
	ID            string                  `json:"id"`
	Attributes    map[string]interface{}  `json:"attributes,omitempty"`
	Relationships map[string]Relationship `json:"relationships,omitempty"`
	Links         map[string]string       `json:"links,omitempty"`
}

// Relationship represents a JSON:API relationship object
type Relationship struct {
	Data  interface{}       `json:"data,omitempty"`
	Links map[string]string `json:"links,omitempty"`
}

// ResourceIdentifier represents a JSON:API resource identifier object
type ResourceIdentifier struct {
	Type string `json:"type"`
	ID   string `json:"id"`
}

// Response represents a JSON:API response document
type Response struct {
	Data     interface{}            `json:"data"`
	Included []Resource             `json:"included,omitempty"`
	Links    map[string]string      `json:"links,omitempty"`
	Meta     map[string]interface{} `json:"meta,omitempty"`
}

// ErrorResponse represents a JSON:API error response
type ErrorResponse struct {
	Errors []Error `json:"errors"`
}

// Error represents a JSON:API error object
type Error struct {
	Status string `json:"status,omitempty"`
	Title  string `json:"title,omitempty"`
	Detail string `json:"detail,omitempty"`
}

// sendResponse sends a JSON:API response
func (s *Server) sendResponse(w http.ResponseWriter, response Response) {
	w.Header().Set("Content-Type", "application/vnd.api+json")

	jsonData, err := json.Marshal(response)
	if err != nil {
		log.Printf("Error marshaling JSON: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Write(jsonData)
}

// sendErrorResponse sends a JSON:API error response
func (s *Server) sendErrorResponse(w http.ResponseWriter, statusCode int, message string) {
	w.Header().Set("Content-Type", "application/vnd.api+json")
	w.WriteHeader(statusCode)

	response := ErrorResponse{
		Errors: []Error{
			{
				Status: strconv.Itoa(statusCode),
				Title:  http.StatusText(statusCode),
				Detail: message,
			},
		},
	}

	jsonData, err := json.Marshal(response)
	if err != nil {
		log.Printf("Error marshaling JSON error response: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Write(jsonData)
}
