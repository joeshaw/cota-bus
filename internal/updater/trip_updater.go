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

// TripUpdater handles updating trip predictions
type TripUpdater struct {
	url   string
	store *store.Store
}

// NewTripUpdater creates a new trip updater
func NewTripUpdater(url string, store *store.Store) *TripUpdater {
	return &TripUpdater{
		url:   url,
		store: store,
	}
}

// Update fetches and processes the GTFS-realtime trip updates feed
func (u *TripUpdater) Update() error {
	log.Println("Updating trip predictions from", u.url)

	// Download the protobuf feed
	resp, err := http.Get(u.url)
	if err != nil {
		return fmt.Errorf("failed to download trip updates feed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	// Read the response body
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read trip updates feed: %v", err)
	}

	// Parse the protobuf message
	feed := &realtime.FeedMessage{}
	if err := proto.Unmarshal(data, feed); err != nil {
		return fmt.Errorf("failed to parse trip updates feed: %v", err)
	}

	// Process the feed
	predictions, count := u.processFeed(feed)

	// Atomically swap in the new predictions
	u.store.UpdatePredictions(predictions)

	log.Printf("Processed %d trip updates", count)

	// Update the last update time
	u.store.SetLastRealtimeUpdate(time.Now())

	return nil
}

// processFeed processes a GTFS-realtime feed message and returns the new predictions
func (u *TripUpdater) processFeed(feed *realtime.FeedMessage) (map[string]*models.Prediction, int) {
	// Create a new map to hold all predictions
	predictions := make(map[string]*models.Prediction)
	count := 0

	// Process each entity
	for _, entity := range feed.Entity {
		if entity.TripUpdate == nil {
			continue
		}

		tripUpdate := entity.TripUpdate

		// Check that we have the required trip ID
		if tripUpdate.Trip == nil || tripUpdate.Trip.TripId == nil {
			continue
		}

		tripID := *tripUpdate.Trip.TripId
		routeID := ""
		if tripUpdate.Trip.RouteId != nil {
			routeID = *tripUpdate.Trip.RouteId
		}

		// Get the route ID from the trip if not provided
		if routeID == "" {
			trip := u.store.GetTrip(tripID)
			if trip != nil {
				routeID = trip.RouteID
			}
		}

		// Process stop time updates
		for _, stopTimeUpdate := range tripUpdate.StopTimeUpdate {
			if stopTimeUpdate.StopId == nil {
				continue
			}

			stopID := *stopTimeUpdate.StopId

			predictionID := fmt.Sprintf("%s-%s", tripID, stopID)
			prediction := &models.Prediction{
				ID:      predictionID,
				TripID:  tripID,
				StopID:  stopID,
				RouteID: routeID,
			}

			// Set direction ID if available
			if tripUpdate.Trip.DirectionId != nil {
				prediction.DirectionID = int(*tripUpdate.Trip.DirectionId)
			}

			// Set stop sequence if available
			if stopTimeUpdate.StopSequence != nil {
				prediction.StopSequence = int(*stopTimeUpdate.StopSequence)
			}

			// Set arrival time if available
			if stopTimeUpdate.Arrival != nil && stopTimeUpdate.Arrival.Time != nil {
				prediction.ArrivalTime = time.Unix(*stopTimeUpdate.Arrival.Time, 0)
			}

			// Set departure time if available
			if stopTimeUpdate.Departure != nil && stopTimeUpdate.Departure.Time != nil {
				prediction.DepartureTime = time.Unix(*stopTimeUpdate.Departure.Time, 0)
			}

			// Set schedule relationship if available
			if stopTimeUpdate.ScheduleRelationship != nil {
				switch *stopTimeUpdate.ScheduleRelationship {
				case realtime.TripUpdate_StopTimeUpdate_SCHEDULED:
					prediction.Status = "SCHEDULED"
				case realtime.TripUpdate_StopTimeUpdate_SKIPPED:
					prediction.Status = "SKIPPED"
				case realtime.TripUpdate_StopTimeUpdate_NO_DATA:
					prediction.Status = "NO_DATA"
				case realtime.TripUpdate_StopTimeUpdate_UNSCHEDULED:
					prediction.Status = "UNSCHEDULED"
				}
			}

			// Add the prediction to our map
			predictions[predictionID] = prediction
			count++
		}
	}

	return predictions, count
}
