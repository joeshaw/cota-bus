package updater

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/joeshaw/cota-bus/internal/models"
	"github.com/joeshaw/cota-bus/internal/realtime"
	"github.com/joeshaw/cota-bus/internal/store"
	"google.golang.org/protobuf/proto"
)

// VehicleUpdater handles updating vehicle positions
type VehicleUpdater struct {
	url   string
	store *store.Store
}

// NewVehicleUpdater creates a new vehicle updater
func NewVehicleUpdater(url string, store *store.Store) *VehicleUpdater {
	return &VehicleUpdater{
		url:   url,
		store: store,
	}
}

// Update fetches and processes the GTFS-realtime vehicle positions feed
func (u *VehicleUpdater) Update() error {
	log.Println("Updating vehicle positions from", u.url)

	// Download the protobuf feed
	resp, err := http.Get(u.url)
	if err != nil {
		return fmt.Errorf("failed to download vehicle positions feed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	// Read the response body
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read vehicle positions feed: %v", err)
	}

	// Parse the protobuf message
	feed := &realtime.FeedMessage{}
	if err := proto.Unmarshal(data, feed); err != nil {
		return fmt.Errorf("failed to parse vehicle positions feed: %v", err)
	}

	// Process the feed
	vehicles, count := u.processFeed(feed)

	// Atomically swap in the new vehicle positions
	u.store.UpdateVehicles(vehicles)

	log.Printf("Processed %d vehicle positions", count)

	// Update the last update time is set in the trip updater

	return nil
}

// processFeed processes a GTFS-realtime feed message and returns the new vehicles
func (u *VehicleUpdater) processFeed(feed *realtime.FeedMessage) (map[string]*models.Vehicle, int) {
	// Create a new map to hold all vehicles
	vehicles := make(map[string]*models.Vehicle)
	count := 0

	// Process each entity
	for _, entity := range feed.Entity {
		if entity.Vehicle == nil {
			continue
		}

		vehiclePosition := entity.Vehicle

		// Check that we have a vehicle descriptor
		if vehiclePosition.Vehicle == nil || vehiclePosition.Vehicle.Id == nil {
			continue
		}

		// Get the vehicle ID
		vehicleID := *vehiclePosition.Vehicle.Id

		// Create a new vehicle object
		vehicle := &models.Vehicle{
			ID:        vehicleID,
			UpdatedAt: time.Now(),
		}

		// Set vehicle label if available
		if vehiclePosition.Vehicle.Label != nil {
			vehicle.VehicleLabel = *vehiclePosition.Vehicle.Label
		}

		// Set trip information if available
		if vehiclePosition.Trip != nil {
			if vehiclePosition.Trip.TripId != nil {
				vehicle.TripID = *vehiclePosition.Trip.TripId
			}

			if vehiclePosition.Trip.RouteId != nil {
				vehicle.RouteID = *vehiclePosition.Trip.RouteId
			}

			if vehiclePosition.Trip.DirectionId != nil {
				vehicle.DirectionID = int(*vehiclePosition.Trip.DirectionId)
			}
		}

		// Get the route ID from the trip if not provided
		if vehicle.RouteID == "" && vehicle.TripID != "" {
			trip := u.store.GetTrip(vehicle.TripID)
			if trip != nil {
				vehicle.RouteID = trip.RouteID
			}
		}

		// Set position information if available
		if vehiclePosition.Position != nil {
			if vehiclePosition.Position.Latitude != nil {
				vehicle.Latitude = float64(*vehiclePosition.Position.Latitude)
			}

			if vehiclePosition.Position.Longitude != nil {
				vehicle.Longitude = float64(*vehiclePosition.Position.Longitude)
			}

			if vehiclePosition.Position.Bearing != nil {
				vehicle.Bearing = float64(*vehiclePosition.Position.Bearing)
			}

			if vehiclePosition.Position.Speed != nil {
				vehicle.Speed = float64(*vehiclePosition.Position.Speed)
			}
		}

		// Set stop information if available
		if vehiclePosition.StopId != nil {
			vehicle.StopID = *vehiclePosition.StopId
		}

		// Set current stop sequence if available
		if vehiclePosition.CurrentStopSequence != nil {
			vehicle.CurrentStopSequence = int(*vehiclePosition.CurrentStopSequence)
		}

		// Set current status if available
		if vehiclePosition.CurrentStatus != nil {
			switch *vehiclePosition.CurrentStatus {
			case realtime.VehiclePosition_INCOMING_AT:
				vehicle.CurrentStatus = "INCOMING_AT"
			case realtime.VehiclePosition_STOPPED_AT:
				vehicle.CurrentStatus = "STOPPED_AT"
			case realtime.VehiclePosition_IN_TRANSIT_TO:
				vehicle.CurrentStatus = "IN_TRANSIT_TO"
			}
		}

		// Set congestion level if available
		if vehiclePosition.CongestionLevel != nil {
			switch *vehiclePosition.CongestionLevel {
			case realtime.VehiclePosition_UNKNOWN_CONGESTION_LEVEL:
				vehicle.CongestionLevel = "UNKNOWN_CONGESTION_LEVEL"
			case realtime.VehiclePosition_RUNNING_SMOOTHLY:
				vehicle.CongestionLevel = "RUNNING_SMOOTHLY"
			case realtime.VehiclePosition_STOP_AND_GO:
				vehicle.CongestionLevel = "STOP_AND_GO"
			case realtime.VehiclePosition_CONGESTION:
				vehicle.CongestionLevel = "CONGESTION"
			case realtime.VehiclePosition_SEVERE_CONGESTION:
				vehicle.CongestionLevel = "SEVERE_CONGESTION"
			}
		}

		// Set occupancy status if available
		if vehiclePosition.OccupancyStatus != nil {
			switch *vehiclePosition.OccupancyStatus {
			case realtime.VehiclePosition_EMPTY:
				vehicle.OccupancyStatus = "EMPTY"
			case realtime.VehiclePosition_MANY_SEATS_AVAILABLE:
				vehicle.OccupancyStatus = "MANY_SEATS_AVAILABLE"
			case realtime.VehiclePosition_FEW_SEATS_AVAILABLE:
				vehicle.OccupancyStatus = "FEW_SEATS_AVAILABLE"
			case realtime.VehiclePosition_STANDING_ROOM_ONLY:
				vehicle.OccupancyStatus = "STANDING_ROOM_ONLY"
			case realtime.VehiclePosition_CRUSHED_STANDING_ROOM_ONLY:
				vehicle.OccupancyStatus = "CRUSHED_STANDING_ROOM_ONLY"
			case realtime.VehiclePosition_FULL:
				vehicle.OccupancyStatus = "FULL"
			case realtime.VehiclePosition_NOT_ACCEPTING_PASSENGERS:
				vehicle.OccupancyStatus = "NOT_ACCEPTING_PASSENGERS"
			}
		}

		// Add the vehicle to our map
		vehicles[vehicleID] = vehicle
		count++
	}

	return vehicles, count
}
