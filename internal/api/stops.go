package api

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/joeshaw/cota-bus/internal/filter"
	"github.com/joeshaw/cota-bus/internal/models"
)

// handleStops handles the stops collection endpoint
func (s *Server) handleStops(w http.ResponseWriter, r *http.Request) {
	options := filter.NewOptions(r.URL.Query())

	// Get all stops
	stops := s.store.GetAllStops()

	// Apply filters
	if options.HasFilter("id") {
		idFilter := options.GetFilter("id")
		stops = filter.Filter(stops, func(stop *models.Stop) bool {
			for _, id := range idFilter {
				if stop.ID == id {
					return true
				}
			}
			return false
		})
	}

	if options.HasFilter("route") {
		routeFilter := options.GetFilter("route")
		filteredStops := []*models.Stop{}

		for _, routeID := range routeFilter {
			routeStops := s.store.GetStopsByRoute(routeID)
			filteredStops = append(filteredStops, routeStops...)
		}

		// If we also have an ID filter, apply it to the route-filtered stops
		if options.HasFilter("id") {
			idFilter := options.GetFilter("id")
			stops = filter.Filter(filteredStops, func(stop *models.Stop) bool {
				for _, id := range idFilter {
					if stop.ID == id {
						return true
					}
				}
				return false
			})
		} else {
			stops = filteredStops
		}
	}

	// Convert to JSON:API resources
	resources := make([]Resource, len(stops))
	for i, stop := range stops {
		resources[i] = stopToResource(stop)
	}

	// Create response
	response := Response{
		Data: resources,
		Links: map[string]string{
			"self": "/stops",
		},
	}

	s.sendResponse(w, response)
}

// handleStop handles the stop detail endpoint
func (s *Server) handleStop(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	stop := s.store.GetStop(id)
	if stop == nil {
		s.sendErrorResponse(w, http.StatusNotFound, "Stop not found")
		return
	}

	// Create response
	response := Response{
		Data: stopToResource(stop),
		Links: map[string]string{
			"self": "/stops/" + id,
		},
	}

	s.sendResponse(w, response)
}

// stopToResource converts a Stop model to a JSON:API resource
func stopToResource(stop *models.Stop) Resource {
	return Resource{
		Type: "stop",
		ID:   stop.ID,
		Attributes: map[string]interface{}{
			"name":                stop.Name,
			"description":         stop.Description,
			"latitude":            stop.Latitude,
			"longitude":           stop.Longitude,
			"location_type":       stop.LocationType,
			"wheelchair_boarding": stop.WheelchairBoarding,
		},
		Links: map[string]string{
			"self": "/stops/" + stop.ID,
		},
	}
}
