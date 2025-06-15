package api

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/joeshaw/cota-bus/internal/store"
)

// Server represents the API server
type Server struct {
	store *store.Store
}

// NewServer creates a new API server
func NewServer(store *store.Store) *Server {
	return &Server{
		store: store,
	}
}

// Router creates and returns the HTTP router
func (s *Server) Router() http.Handler {
	r := mux.NewRouter()

	// Register routes
	r.HandleFunc("/", s.handleIndex).Methods("GET")
	r.HandleFunc("/routes", s.handleRoutes).Methods("GET")
	r.HandleFunc("/routes/{id}", s.handleRoute).Methods("GET")
	r.HandleFunc("/stops", s.handleStops).Methods("GET")
	r.HandleFunc("/stops/{id}", s.handleStop).Methods("GET")
	r.HandleFunc("/trips", s.handleTrips).Methods("GET")
	r.HandleFunc("/trips/{id}", s.handleTrip).Methods("GET")
	r.HandleFunc("/vehicles", s.handleVehicles).Methods("GET")
	r.HandleFunc("/vehicles/{id}", s.handleVehicle).Methods("GET")
	r.HandleFunc("/predictions", s.handlePredictions).Methods("GET")
	r.HandleFunc("/predictions/{id}", s.handlePrediction).Methods("GET")
	r.HandleFunc("/shapes", s.handleShapes).Methods("GET")
	r.HandleFunc("/shapes/{id}", s.handleShape).Methods("GET")

	// Add CORS middleware
	return s.corsMiddleware(r)
}

// corsMiddleware adds CORS headers to all responses
func (s *Server) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}
