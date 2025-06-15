package api

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/joeshaw/cota-bus/internal/filter"
	"github.com/joeshaw/cota-bus/internal/models"
)

// handleTrips handles the trips collection endpoint
func (s *Server) handleTrips(w http.ResponseWriter, r *http.Request) {
	options := filter.NewOptions(r.URL.Query())

	// Get all trips or apply initial filters
	var trips []*models.Trip

	if options.HasFilter("route") {
		// Filter by route is a common case, use optimized store method
		routeFilter := options.GetFilter("route")
		for _, routeID := range routeFilter {
			routeTrips := s.store.GetTripsByRoute(routeID)
			trips = append(trips, routeTrips...)
		}
	} else {
		// Get all trips
		trips = s.store.GetAllTrips()
	}

	// Apply additional filters
	if options.HasFilter("id") {
		idFilter := options.GetFilter("id")
		trips = filter.Filter(trips, func(trip *models.Trip) bool {
			for _, id := range idFilter {
				if trip.ID == id {
					return true
				}
			}
			return false
		})
	}

	// Convert to JSON:API resources
	resources := make([]Resource, len(trips))
	for i, trip := range trips {
		resources[i] = tripToResource(trip)
	}

	// Create response
	response := Response{
		Data: resources,
		Links: map[string]string{
			"self": "/trips",
		},
	}

	s.sendResponse(w, response)
}

// handleTrip handles the trip detail endpoint
func (s *Server) handleTrip(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	trip := s.store.GetTrip(id)
	if trip == nil {
		s.sendErrorResponse(w, http.StatusNotFound, "Trip not found")
		return
	}

	options := filter.NewOptions(r.URL.Query())

	// Create response
	response := Response{
		Data: tripToResource(trip),
		Links: map[string]string{
			"self": "/trips/" + id,
		},
	}

	// Include route if requested
	var included []Resource

	if options.HasInclude("route") {
		route := s.store.GetRoute(trip.RouteID)
		if route != nil {
			included = append(included, routeToResource(route))
		}
	}

	// Include stops if requested
	if options.HasInclude("stops") {
		stopTimes := s.store.GetStopTimesByTrip(trip.ID)
		for _, stopTime := range stopTimes {
			stop := s.store.GetStop(stopTime.StopID)
			if stop != nil {
				included = append(included, stopToResource(stop))
			}
		}
	}

	if len(included) > 0 {
		response.Included = included
	}

	s.sendResponse(w, response)
}

// tripToResource converts a Trip model to a JSON:API resource
func tripToResource(trip *models.Trip) Resource {
	return Resource{
		Type: "trip",
		ID:   trip.ID,
		Attributes: map[string]interface{}{
			"headsign":              trip.Headsign,
			"short_name":            trip.ShortName,
			"direction_id":          trip.DirectionID,
			"block_id":              trip.BlockID,
			"wheelchair_accessible": trip.WheelchairAccessible,
			"bikes_allowed":         trip.BikesAllowed,
		},
		Links: map[string]string{
			"self": "/trips/" + trip.ID,
		},
		Relationships: map[string]Relationship{
			"route": {
				Data: ResourceIdentifier{
					Type: "route",
					ID:   trip.RouteID,
				},
			},
		},
	}
}
