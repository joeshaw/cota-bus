package api

import (
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/joeshaw/cota-bus/internal/filter"
	"github.com/joeshaw/cota-bus/internal/models"
	"github.com/joeshaw/cota-bus/internal/store"
)

// handleVehicles handles the vehicles collection endpoint
func (s *Server) handleVehicles(w http.ResponseWriter, r *http.Request) {
	options := filter.NewOptions(r.URL.Query())

	// Get all vehicles
	vehicles := s.store.GetAllVehicles()

	// Apply filters
	if options.HasFilter("id") {
		idFilter := options.GetFilter("id")
		vehicles = filter.Filter(vehicles, func(vehicle *models.Vehicle) bool {
			for _, id := range idFilter {
				if vehicle.ID == id {
					return true
				}
			}
			return false
		})
	}

	if options.HasFilter("route") {
		routeFilter := options.GetFilter("route")
		filteredVehicles := []*models.Vehicle{}

		for _, routeID := range routeFilter {
			routeVehicles := s.store.GetVehiclesByRoute(routeID)
			filteredVehicles = append(filteredVehicles, routeVehicles...)
		}

		vehicles = filteredVehicles
	}

	if options.HasFilter("trip") {
		tripFilter := options.GetFilter("trip")
		filteredVehicles := []*models.Vehicle{}

		for _, tripID := range tripFilter {
			vehicle := s.store.GetVehicleByTrip(tripID)
			if vehicle != nil {
				filteredVehicles = append(filteredVehicles, vehicle)
			}
		}

		vehicles = filteredVehicles
	}

	// Convert to JSON:API resources
	resources := make([]Resource, len(vehicles))
	for i, vehicle := range vehicles {
		resources[i] = vehicleToResource(vehicle, s.store)
	}

	// Create response
	response := Response{
		Data: resources,
		Links: map[string]string{
			"self": "/vehicles",
		},
	}

	s.sendResponse(w, response)
}

// handleVehicle handles the vehicle detail endpoint
func (s *Server) handleVehicle(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	vehicle := s.store.GetVehicle(id)
	if vehicle == nil {
		s.sendErrorResponse(w, http.StatusNotFound, "Vehicle not found")
		return
	}

	options := filter.NewOptions(r.URL.Query())

	// Create response
	response := Response{
		Data: vehicleToResource(vehicle, s.store),
		Links: map[string]string{
			"self": "/vehicles/" + id,
		},
	}

	// Include related resources if requested
	var included []Resource

	if options.HasInclude("trip") && vehicle.TripID != "" {
		trip := s.store.GetTrip(vehicle.TripID)
		if trip != nil {
			included = append(included, tripToResource(trip))
		}
	}

	if options.HasInclude("route") && vehicle.RouteID != "" {
		route := s.store.GetRoute(vehicle.RouteID)
		if route != nil {
			included = append(included, routeToResource(route))
		}
	}

	if len(included) > 0 {
		response.Included = included
	}

	s.sendResponse(w, response)
}

// vehicleToResource converts a Vehicle model to a JSON:API resource
func vehicleToResource(vehicle *models.Vehicle, store *store.Store) Resource {
	// Work around COTA GTFS-realtime bug: direction_id values don't match static GTFS
	// Use the correct direction_id from the associated trip instead of the realtime value
	directionID := vehicle.DirectionID
	if vehicle.TripID != "" {
		if trip := store.GetTrip(vehicle.TripID); trip != nil {
			directionID = trip.DirectionID
		}
	}

	attributes := map[string]interface{}{
		"latitude":     vehicle.Latitude,
		"longitude":    vehicle.Longitude,
		"bearing":      vehicle.Bearing,
		"speed":        vehicle.Speed,
		"direction_id": directionID,
		"updated_at":   vehicle.UpdatedAt.Format(time.RFC3339),
		"label":        vehicle.VehicleLabel,
	}

	// Only include optional attributes if they have meaningful values
	if vehicle.CurrentStopSequence > 0 {
		attributes["current_stop_sequence"] = vehicle.CurrentStopSequence
	}
	if vehicle.CurrentStatus != "" {
		attributes["current_status"] = vehicle.CurrentStatus
	}
	if vehicle.CongestionLevel != "" {
		attributes["congestion_level"] = vehicle.CongestionLevel
	}
	if vehicle.OccupancyStatus != "" {
		attributes["occupancy_status"] = vehicle.OccupancyStatus
	}

	resource := Resource{
		Type:       "vehicle",
		ID:         vehicle.ID,
		Attributes: attributes,
		Links: map[string]string{
			"self": "/vehicles/" + vehicle.ID,
		},
		Relationships: map[string]Relationship{},
	}

	// Add trip relationship if available
	if vehicle.TripID != "" {
		resource.Relationships["trip"] = Relationship{
			Data: ResourceIdentifier{
				Type: "trip",
				ID:   vehicle.TripID,
			},
		}
	}

	// Add route relationship if available
	if vehicle.RouteID != "" {
		resource.Relationships["route"] = Relationship{
			Data: ResourceIdentifier{
				Type: "route",
				ID:   vehicle.RouteID,
			},
		}
	}

	return resource
}
