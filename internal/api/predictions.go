package api

import (
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/joeshaw/cota-bus/internal/filter"
	"github.com/joeshaw/cota-bus/internal/models"
	"github.com/joeshaw/cota-bus/internal/store"
)

// handlePredictions handles the predictions collection endpoint
func (s *Server) handlePredictions(w http.ResponseWriter, r *http.Request) {
	options := filter.NewOptions(r.URL.Query())

	// Filter is required for predictions
	if !options.HasFilter("route") && !options.HasFilter("trip") && !options.HasFilter("stop") {
		s.sendErrorResponse(w, http.StatusBadRequest, "At least one filter (route, trip, or stop) is required")
		return
	}

	// Get predictions based on filters
	var predictions []*models.Prediction
	var candidatePredictions []*models.Prediction

	// Start with the first available filter to get candidate predictions
	if options.HasFilter("route") {
		routeFilter := options.GetFilter("route")
		for _, routeID := range routeFilter {
			routePredictions := s.store.GetPredictionsByRoute(routeID)
			candidatePredictions = append(candidatePredictions, routePredictions...)
		}
	} else if options.HasFilter("trip") {
		tripFilter := options.GetFilter("trip")
		for _, tripID := range tripFilter {
			tripPredictions := s.store.GetPredictionsByTrip(tripID)
			candidatePredictions = append(candidatePredictions, tripPredictions...)
		}
	} else if options.HasFilter("stop") {
		stopFilter := options.GetFilter("stop")
		for _, stopID := range stopFilter {
			stopPredictions := s.store.GetPredictionsByStop(stopID)
			candidatePredictions = append(candidatePredictions, stopPredictions...)
		}
	}

	// Now filter the candidates based on remaining filters
	for _, prediction := range candidatePredictions {
		matches := true

		// Check route filter
		if options.HasFilter("route") {
			routeFilter := options.GetFilter("route")
			routeMatches := false
			for _, routeID := range routeFilter {
				if prediction.RouteID == routeID {
					routeMatches = true
					break
				}
			}
			if !routeMatches {
				matches = false
			}
		}

		// Check trip filter
		if matches && options.HasFilter("trip") {
			tripFilter := options.GetFilter("trip")
			tripMatches := false
			for _, tripID := range tripFilter {
				if prediction.TripID == tripID {
					tripMatches = true
					break
				}
			}
			if !tripMatches {
				matches = false
			}
		}

		// Check stop filter
		if matches && options.HasFilter("stop") {
			stopFilter := options.GetFilter("stop")
			stopMatches := false
			for _, stopID := range stopFilter {
				if prediction.StopID == stopID {
					stopMatches = true
					break
				}
			}
			if !stopMatches {
				matches = false
			}
		}

		if matches {
			predictions = append(predictions, prediction)
		}
	}

	// Convert to JSON:API resources
	resources := make([]Resource, len(predictions))
	for i, prediction := range predictions {
		resources[i] = predictionToResource(prediction, s.store)
	}

	// Create response
	response := Response{
		Data: resources,
		Links: map[string]string{
			"self": r.URL.String(),
		},
	}

	// Include related resources if requested
	var included []Resource
	includedMap := make(map[string]bool) // Track included resources to avoid duplicates

	if options.HasInclude("stop") {
		for _, prediction := range predictions {
			key := "stop-" + prediction.StopID
			if !includedMap[key] {
				stop := s.store.GetStop(prediction.StopID)
				if stop != nil {
					included = append(included, stopToResource(stop))
					includedMap[key] = true
				}
			}
		}
	}

	if options.HasInclude("trip") {
		for _, prediction := range predictions {
			key := "trip-" + prediction.TripID
			if !includedMap[key] {
				trip := s.store.GetTrip(prediction.TripID)
				if trip != nil {
					included = append(included, tripToResource(trip))
					includedMap[key] = true
				}
			}
		}
	}

	if options.HasInclude("route") {
		for _, prediction := range predictions {
			key := "route-" + prediction.RouteID
			if !includedMap[key] {
				route := s.store.GetRoute(prediction.RouteID)
				if route != nil {
					included = append(included, routeToResource(route))
					includedMap[key] = true
				}
			}
		}
	}

	if len(included) > 0 {
		response.Included = included
	}

	s.sendResponse(w, response)
}

// handlePrediction handles the prediction detail endpoint
func (s *Server) handlePrediction(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	prediction := s.store.GetPrediction(id)
	if prediction == nil {
		s.sendErrorResponse(w, http.StatusNotFound, "Prediction not found")
		return
	}

	options := filter.NewOptions(r.URL.Query())

	// Create response
	response := Response{
		Data: predictionToResource(prediction, s.store),
		Links: map[string]string{
			"self": "/predictions/" + id,
		},
	}

	// Include related resources if requested
	var included []Resource

	if options.HasInclude("trip") {
		trip := s.store.GetTrip(prediction.TripID)
		if trip != nil {
			included = append(included, tripToResource(trip))
		}
	}

	if options.HasInclude("stop") {
		stop := s.store.GetStop(prediction.StopID)
		if stop != nil {
			included = append(included, stopToResource(stop))
		}
	}

	if options.HasInclude("route") {
		route := s.store.GetRoute(prediction.RouteID)
		if route != nil {
			included = append(included, routeToResource(route))
		}
	}

	if len(included) > 0 {
		response.Included = included
	}

	s.sendResponse(w, response)
}

// predictionToResource converts a Prediction model to a JSON:API resource
func predictionToResource(prediction *models.Prediction, store *store.Store) Resource {
	// Work around COTA GTFS-realtime bug: direction_id values don't match static GTFS
	// Use the correct direction_id from the associated trip instead of the realtime value
	directionID := prediction.DirectionID
	if prediction.TripID != "" {
		if trip := store.GetTrip(prediction.TripID); trip != nil {
			directionID = trip.DirectionID
		}
	}

	resource := Resource{
		Type: "prediction",
		ID:   prediction.ID,
		Attributes: map[string]interface{}{
			"status":        prediction.Status,
			"direction_id":  directionID,
			"stop_sequence": prediction.StopSequence,
		},
		Links: map[string]string{
			"self": "/predictions/" + prediction.ID,
		},
		Relationships: map[string]Relationship{
			"trip": {
				Data: ResourceIdentifier{
					Type: "trip",
					ID:   prediction.TripID,
				},
			},
			"stop": {
				Data: ResourceIdentifier{
					Type: "stop",
					ID:   prediction.StopID,
				},
			},
			"route": {
				Data: ResourceIdentifier{
					Type: "route",
					ID:   prediction.RouteID,
				},
			},
		},
	}

	// Add arrival time if available
	if !prediction.ArrivalTime.IsZero() {
		resource.Attributes["arrival_time"] = prediction.ArrivalTime.Format(time.RFC3339)
	}

	// Add departure time if available
	if !prediction.DepartureTime.IsZero() {
		resource.Attributes["departure_time"] = prediction.DepartureTime.Format(time.RFC3339)
	}

	return resource
}
