package store

import (
	"strings"
	"sync"
	"time"

	"github.com/joeshaw/cota-bus/internal/models"
)

// Store provides thread-safe access to transit data
type Store struct {
	mu sync.RWMutex

	agencies      map[string]*models.Agency
	routes        map[string]*models.Route
	stops         map[string]*models.Stop
	trips         map[string]*models.Trip
	stopTimes     map[string]map[string]*models.StopTime // map[tripID]map[stopID]StopTime
	calendars     map[string]*models.Calendar
	calendarDates map[string]map[string]*models.CalendarDate // map[serviceID]map[date]CalendarDate
	shapes        map[string][]*models.Shape                 // map[shapeID][]Shape

	// Realtime data
	vehicles    map[string]*models.Vehicle
	predictions map[string]*models.Prediction

	// Additional indexes for faster lookups
	routesByAgency     map[string][]string                    // map[agencyID][]routeID
	stopsByRoute       map[string][]string                    // map[routeID][]stopID
	tripsByRoute       map[string][]string                    // map[routeID][]tripID
	stopTimesByStop    map[string]map[string]*models.StopTime // map[stopID]map[tripID]StopTime
	vehiclesByRoute    map[string][]string                    // map[routeID][]vehicleID
	vehiclesByTrip     map[string]string                      // map[tripID]vehicleID
	predictionsByStop  map[string][]string                    // map[stopID][]predictionID
	predictionsByRoute map[string][]string                    // map[routeID][]predictionID
	predictionsByTrip  map[string][]string                    // map[tripID][]predictionID

	lastStaticUpdate   time.Time
	lastRealtimeUpdate time.Time
}

// NewStore creates a new data store
func NewStore() *Store {
	return &Store{
		agencies:      make(map[string]*models.Agency),
		routes:        make(map[string]*models.Route),
		stops:         make(map[string]*models.Stop),
		trips:         make(map[string]*models.Trip),
		stopTimes:     make(map[string]map[string]*models.StopTime),
		calendars:     make(map[string]*models.Calendar),
		calendarDates: make(map[string]map[string]*models.CalendarDate),
		shapes:        make(map[string][]*models.Shape),

		vehicles:    make(map[string]*models.Vehicle),
		predictions: make(map[string]*models.Prediction),

		routesByAgency:     make(map[string][]string),
		stopsByRoute:       make(map[string][]string),
		tripsByRoute:       make(map[string][]string),
		stopTimesByStop:    make(map[string]map[string]*models.StopTime),
		vehiclesByRoute:    make(map[string][]string),
		vehiclesByTrip:     make(map[string]string),
		predictionsByStop:  make(map[string][]string),
		predictionsByRoute: make(map[string][]string),
		predictionsByTrip:  make(map[string][]string),
	}
}

// Clear removes all data from the store
func (s *Store) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.agencies = make(map[string]*models.Agency)
	s.routes = make(map[string]*models.Route)
	s.stops = make(map[string]*models.Stop)
	s.trips = make(map[string]*models.Trip)
	s.stopTimes = make(map[string]map[string]*models.StopTime)
	s.calendars = make(map[string]*models.Calendar)
	s.calendarDates = make(map[string]map[string]*models.CalendarDate)
	s.shapes = make(map[string][]*models.Shape)

	s.routesByAgency = make(map[string][]string)
	s.stopsByRoute = make(map[string][]string)
	s.tripsByRoute = make(map[string][]string)
	s.stopTimesByStop = make(map[string]map[string]*models.StopTime)
}

// ClearRealtimeData removes all realtime data from the store
func (s *Store) ClearRealtimeData() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.vehicles = make(map[string]*models.Vehicle)
	s.predictions = make(map[string]*models.Prediction)

	s.vehiclesByRoute = make(map[string][]string)
	s.vehiclesByTrip = make(map[string]string)
	s.predictionsByStop = make(map[string][]string)
	s.predictionsByRoute = make(map[string][]string)
	s.predictionsByTrip = make(map[string][]string)
}

// UpdateVehicles atomically replaces all vehicle data with the new data
func (s *Store) UpdateVehicles(vehicles map[string]*models.Vehicle) {
	// Build the indexes outside the lock
	vehiclesByRoute := make(map[string][]string)
	vehiclesByTrip := make(map[string]string)

	for id, vehicle := range vehicles {
		// Update vehiclesByRoute
		if vehicle.RouteID != "" {
			vehiclesByRoute[vehicle.RouteID] = append(vehiclesByRoute[vehicle.RouteID], id)
		}

		// Update vehiclesByTrip
		if vehicle.TripID != "" {
			vehiclesByTrip[vehicle.TripID] = id
		}
	}

	// Atomically swap in the new data
	s.mu.Lock()
	defer s.mu.Unlock()

	s.vehicles = vehicles
	s.vehiclesByRoute = vehiclesByRoute
	s.vehiclesByTrip = vehiclesByTrip
}

// UpdatePredictions atomically replaces all prediction data with the new data
func (s *Store) UpdatePredictions(predictions map[string]*models.Prediction) {
	// Build the indexes outside the lock
	predictionsByStop := make(map[string][]string)
	predictionsByRoute := make(map[string][]string)
	predictionsByTrip := make(map[string][]string)

	for id, prediction := range predictions {
		// Update predictionsByStop
		if prediction.StopID != "" {
			predictionsByStop[prediction.StopID] = append(predictionsByStop[prediction.StopID], id)
		}

		// Update predictionsByRoute
		if prediction.RouteID != "" {
			predictionsByRoute[prediction.RouteID] = append(predictionsByRoute[prediction.RouteID], id)
		}

		// Update predictionsByTrip
		if prediction.TripID != "" {
			predictionsByTrip[prediction.TripID] = append(predictionsByTrip[prediction.TripID], id)
		}
	}

	// Atomically swap in the new data
	s.mu.Lock()
	defer s.mu.Unlock()

	s.predictions = predictions
	s.predictionsByStop = predictionsByStop
	s.predictionsByRoute = predictionsByRoute
	s.predictionsByTrip = predictionsByTrip
}

// Agency methods
func (s *Store) AddAgency(agency *models.Agency) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.agencies[agency.ID] = agency
}

func (s *Store) GetAgency(id string) *models.Agency {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.agencies[id]
}

func (s *Store) GetAllAgencies() []*models.Agency {
	s.mu.RLock()
	defer s.mu.RUnlock()
	agencies := make([]*models.Agency, 0, len(s.agencies))
	for _, agency := range s.agencies {
		agencies = append(agencies, agency)
	}
	return agencies
}

// Route methods
func (s *Store) AddRoute(route *models.Route) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.routes[route.ID] = route

	// Update indexes
	if route.AgencyID != "" {
		s.routesByAgency[route.AgencyID] = append(s.routesByAgency[route.AgencyID], route.ID)
	}
}

func (s *Store) GetRoute(id string) *models.Route {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.routes[id]
}

func (s *Store) GetAllRoutes() []*models.Route {
	s.mu.RLock()
	defer s.mu.RUnlock()
	routes := make([]*models.Route, 0, len(s.routes))
	for _, route := range s.routes {
		routes = append(routes, route)
	}
	return routes
}

func (s *Store) GetRoutesByAgency(agencyID string) []*models.Route {
	s.mu.RLock()
	defer s.mu.RUnlock()
	routeIDs := s.routesByAgency[agencyID]
	routes := make([]*models.Route, 0, len(routeIDs))
	for _, id := range routeIDs {
		if route, ok := s.routes[id]; ok {
			routes = append(routes, route)
		}
	}
	return routes
}

// Stop methods
func (s *Store) AddStop(stop *models.Stop) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.stops[stop.ID] = stop

	// Initialize stopTimesByStop for this stop
	if _, ok := s.stopTimesByStop[stop.ID]; !ok {
		s.stopTimesByStop[stop.ID] = make(map[string]*models.StopTime)
	}
}

func (s *Store) GetStop(id string) *models.Stop {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.stops[id]
}

func (s *Store) GetAllStops() []*models.Stop {
	s.mu.RLock()
	defer s.mu.RUnlock()
	stops := make([]*models.Stop, 0, len(s.stops))
	for _, stop := range s.stops {
		stops = append(stops, stop)
	}
	return stops
}

func (s *Store) GetStopsByRoute(routeID string) []*models.Stop {
	s.mu.RLock()
	defer s.mu.RUnlock()
	stopIDs := s.stopsByRoute[routeID]
	stops := make([]*models.Stop, 0, len(stopIDs))
	for _, id := range stopIDs {
		if stop, ok := s.stops[id]; ok {
			stops = append(stops, stop)
		}
	}
	return stops
}

// Trip methods
func (s *Store) AddTrip(trip *models.Trip) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.trips[trip.ID] = trip

	// Update indexes
	s.tripsByRoute[trip.RouteID] = append(s.tripsByRoute[trip.RouteID], trip.ID)

	// Initialize stopTimes for this trip
	if _, ok := s.stopTimes[trip.ID]; !ok {
		s.stopTimes[trip.ID] = make(map[string]*models.StopTime)
	}
}

func (s *Store) GetTrip(id string) *models.Trip {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.trips[id]
}

func (s *Store) GetAllTrips() []*models.Trip {
	s.mu.RLock()
	defer s.mu.RUnlock()
	trips := make([]*models.Trip, 0, len(s.trips))
	for _, trip := range s.trips {
		trips = append(trips, trip)
	}
	return trips
}

func (s *Store) GetTripsByRoute(routeID string) []*models.Trip {
	s.mu.RLock()
	defer s.mu.RUnlock()
	tripIDs := s.tripsByRoute[routeID]
	trips := make([]*models.Trip, 0, len(tripIDs))
	for _, id := range tripIDs {
		if trip, ok := s.trips[id]; ok {
			trips = append(trips, trip)
		}
	}
	return trips
}

// StopTime methods
func (s *Store) AddStopTime(stopTime *models.StopTime) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Ensure maps exist
	if _, ok := s.stopTimes[stopTime.TripID]; !ok {
		s.stopTimes[stopTime.TripID] = make(map[string]*models.StopTime)
	}
	if _, ok := s.stopTimesByStop[stopTime.StopID]; !ok {
		s.stopTimesByStop[stopTime.StopID] = make(map[string]*models.StopTime)
	}

	// Add to maps
	s.stopTimes[stopTime.TripID][stopTime.StopID] = stopTime
	s.stopTimesByStop[stopTime.StopID][stopTime.TripID] = stopTime

	// Update route-stop index
	if trip, ok := s.trips[stopTime.TripID]; ok {
		found := false
		for _, id := range s.stopsByRoute[trip.RouteID] {
			if id == stopTime.StopID {
				found = true
				break
			}
		}
		if !found {
			s.stopsByRoute[trip.RouteID] = append(s.stopsByRoute[trip.RouteID], stopTime.StopID)
		}
	}
}

func (s *Store) GetStopTimesByTrip(tripID string) []*models.StopTime {
	s.mu.RLock()
	defer s.mu.RUnlock()

	tripStopTimes := s.stopTimes[tripID]
	stopTimes := make([]*models.StopTime, 0, len(tripStopTimes))
	for _, stopTime := range tripStopTimes {
		stopTimes = append(stopTimes, stopTime)
	}
	return stopTimes
}

func (s *Store) GetStopTimesByStop(stopID string) []*models.StopTime {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stopStopTimes := s.stopTimesByStop[stopID]
	stopTimes := make([]*models.StopTime, 0, len(stopStopTimes))
	for _, stopTime := range stopStopTimes {
		stopTimes = append(stopTimes, stopTime)
	}
	return stopTimes
}

// Calendar methods
func (s *Store) AddCalendar(calendar *models.Calendar) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calendars[calendar.ServiceID] = calendar
}

func (s *Store) GetCalendar(serviceID string) *models.Calendar {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.calendars[serviceID]
}

// CalendarDate methods
func (s *Store) AddCalendarDate(calendarDate *models.CalendarDate) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.calendarDates[calendarDate.ServiceID]; !ok {
		s.calendarDates[calendarDate.ServiceID] = make(map[string]*models.CalendarDate)
	}
	s.calendarDates[calendarDate.ServiceID][calendarDate.Date] = calendarDate
}

func (s *Store) GetCalendarDatesByService(serviceID string) []*models.CalendarDate {
	s.mu.RLock()
	defer s.mu.RUnlock()

	serviceDates := s.calendarDates[serviceID]
	calendarDates := make([]*models.CalendarDate, 0, len(serviceDates))
	for _, date := range serviceDates {
		calendarDates = append(calendarDates, date)
	}
	return calendarDates
}

// Shape methods
func (s *Store) AddShape(shape *models.Shape) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.shapes[shape.ID] = append(s.shapes[shape.ID], shape)
}

func (s *Store) GetShapesByID(shapeID string) []*models.Shape {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.shapes[shapeID]
}

func (s *Store) GetAllShapes() map[string][]*models.Shape {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Create a copy of the shapes map to avoid race conditions
	shapesCopy := make(map[string][]*models.Shape, len(s.shapes))
	for id, points := range s.shapes {
		shapesCopy[id] = points
	}

	return shapesCopy
}

// Vehicle methods
func (s *Store) AddVehicle(vehicle *models.Vehicle) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.vehicles[vehicle.ID] = vehicle

	// Update indexes
	s.vehiclesByTrip[vehicle.TripID] = vehicle.ID

	// Update vehiclesByRoute
	found := false
	for _, id := range s.vehiclesByRoute[vehicle.RouteID] {
		if id == vehicle.ID {
			found = true
			break
		}
	}
	if !found {
		s.vehiclesByRoute[vehicle.RouteID] = append(s.vehiclesByRoute[vehicle.RouteID], vehicle.ID)
	}
}

func (s *Store) GetVehicle(id string) *models.Vehicle {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.vehicles[id]
}

func (s *Store) GetAllVehicles() []*models.Vehicle {
	s.mu.RLock()
	defer s.mu.RUnlock()
	vehicles := make([]*models.Vehicle, 0, len(s.vehicles))
	for _, vehicle := range s.vehicles {
		vehicles = append(vehicles, vehicle)
	}
	return vehicles
}

func (s *Store) GetVehiclesByRoute(routeID string) []*models.Vehicle {
	s.mu.RLock()
	defer s.mu.RUnlock()
	vehicleIDs := s.vehiclesByRoute[routeID]
	vehicles := make([]*models.Vehicle, 0, len(vehicleIDs))
	for _, id := range vehicleIDs {
		if vehicle, ok := s.vehicles[id]; ok {
			vehicles = append(vehicles, vehicle)
		}
	}
	return vehicles
}

func (s *Store) GetVehicleByTrip(tripID string) *models.Vehicle {
	s.mu.RLock()
	defer s.mu.RUnlock()
	vehicleID, ok := s.vehiclesByTrip[tripID]
	if !ok {
		return nil
	}
	return s.vehicles[vehicleID]
}

// Prediction methods
func (s *Store) AddPrediction(prediction *models.Prediction) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.predictions[prediction.ID] = prediction

	// Update indexes
	s.predictionsByStop[prediction.StopID] = append(s.predictionsByStop[prediction.StopID], prediction.ID)
	s.predictionsByRoute[prediction.RouteID] = append(s.predictionsByRoute[prediction.RouteID], prediction.ID)
	s.predictionsByTrip[prediction.TripID] = append(s.predictionsByTrip[prediction.TripID], prediction.ID)
}

func (s *Store) GetPrediction(id string) *models.Prediction {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.predictions[id]
}

func (s *Store) GetAllPredictions() []*models.Prediction {
	s.mu.RLock()
	defer s.mu.RUnlock()
	predictions := make([]*models.Prediction, 0, len(s.predictions))
	for _, prediction := range s.predictions {
		predictions = append(predictions, prediction)
	}
	return predictions
}

func (s *Store) GetPredictionsByStop(stopID string) []*models.Prediction {
	s.mu.RLock()
	defer s.mu.RUnlock()
	predictionIDs := s.predictionsByStop[stopID]
	predictions := make([]*models.Prediction, 0, len(predictionIDs))
	for _, id := range predictionIDs {
		if prediction, ok := s.predictions[id]; ok {
			predictions = append(predictions, prediction)
		}
	}
	return predictions
}

func (s *Store) GetPredictionsByRoute(routeID string) []*models.Prediction {
	s.mu.RLock()
	defer s.mu.RUnlock()
	predictionIDs := s.predictionsByRoute[routeID]
	predictions := make([]*models.Prediction, 0, len(predictionIDs))
	for _, id := range predictionIDs {
		if prediction, ok := s.predictions[id]; ok {
			predictions = append(predictions, prediction)
		}
	}
	return predictions
}

func (s *Store) GetPredictionsByTrip(tripID string) []*models.Prediction {
	s.mu.RLock()
	defer s.mu.RUnlock()
	predictionIDs := s.predictionsByTrip[tripID]
	predictions := make([]*models.Prediction, 0, len(predictionIDs))
	for _, id := range predictionIDs {
		if prediction, ok := s.predictions[id]; ok {
			predictions = append(predictions, prediction)
		}
	}
	return predictions
}

// RemovePrediction removes a prediction from the store
func (s *Store) RemovePrediction(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	prediction, ok := s.predictions[id]
	if !ok {
		return
	}

	// Remove from main map
	delete(s.predictions, id)

	// Remove from predictionsByStop
	if prediction.StopID != "" {
		stopIDs := s.predictionsByStop[prediction.StopID]
		for i, stopID := range stopIDs {
			if stopID == id {
				s.predictionsByStop[prediction.StopID] = append(stopIDs[:i], stopIDs[i+1:]...)
				break
			}
		}
	}

	// Remove from predictionsByRoute
	if prediction.RouteID != "" {
		routeIDs := s.predictionsByRoute[prediction.RouteID]
		for i, routeID := range routeIDs {
			if routeID == id {
				s.predictionsByRoute[prediction.RouteID] = append(routeIDs[:i], routeIDs[i+1:]...)
				break
			}
		}
	}

	// Remove from predictionsByTrip
	if prediction.TripID != "" {
		tripIDs := s.predictionsByTrip[prediction.TripID]
		for i, tripID := range tripIDs {
			if tripID == id {
				s.predictionsByTrip[prediction.TripID] = append(tripIDs[:i], tripIDs[i+1:]...)
				break
			}
		}
	}
}

// Update time getters/setters
func (s *Store) SetLastStaticUpdate(t time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.lastStaticUpdate = t
}

func (s *Store) GetLastStaticUpdate() time.Time {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.lastStaticUpdate
}

func (s *Store) SetLastRealtimeUpdate(t time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.lastRealtimeUpdate = t
}

func (s *Store) GetLastRealtimeUpdate() time.Time {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.lastRealtimeUpdate
}

// BuildStopsByRoute builds the stopsByRoute index from existing data
// This should be called after all GTFS data is loaded
func (s *Store) BuildStopsByRoute() {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Clear existing index
	s.stopsByRoute = make(map[string][]string)

	// Iterate through all trips and their stop times
	for tripID, trip := range s.trips {
		if stopTimesForTrip, ok := s.stopTimes[tripID]; ok {
			for stopID := range stopTimesForTrip {
				// Check if this stop is already in the route's list
				found := false
				for _, existingStopID := range s.stopsByRoute[trip.RouteID] {
					if existingStopID == stopID {
						found = true
						break
					}
				}
				if !found {
					s.stopsByRoute[trip.RouteID] = append(s.stopsByRoute[trip.RouteID], stopID)
				}
			}
		}
	}
}

// BuildRouteDirections builds direction names and destinations for each route from trip headsigns
func (s *Store) BuildRouteDirections() {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Track direction headsigns by route and direction_id
	routeDirections := make(map[string]map[int]map[string]bool)

	// Collect all unique headsigns by route and direction
	for _, trip := range s.trips {
		if routeDirections[trip.RouteID] == nil {
			routeDirections[trip.RouteID] = make(map[int]map[string]bool)
		}
		if routeDirections[trip.RouteID][trip.DirectionID] == nil {
			routeDirections[trip.RouteID][trip.DirectionID] = make(map[string]bool)
		}
		if trip.Headsign != "" {
			routeDirections[trip.RouteID][trip.DirectionID][trip.Headsign] = true
		}
	}

	// Build direction arrays for each route
	for routeID, directions := range routeDirections {
		route := s.routes[routeID]
		if route == nil {
			continue
		}

		// Initialize arrays with empty strings
		route.DirectionNames = make([]string, 2)
		route.DirectionDestinations = make([]string, 2)

		// Set direction destinations from headsigns
		for directionID, headsigns := range directions {
			if directionID < 0 || directionID > 1 {
				continue
			}

			// Use the first headsign to extract destination info
			for headsign := range headsigns {
				// Look for "TO" in the headsign to extract destination
				if toIndex := strings.Index(headsign, " TO "); toIndex != -1 {
					destination := headsign[toIndex+4:] // Everything after " TO "
					route.DirectionDestinations[directionID] = destination
					route.DirectionNames[directionID] = "TO " + destination
				} else {
					// Fallback if no "TO" found
					route.DirectionDestinations[directionID] = headsign
					route.DirectionNames[directionID] = headsign
				}
				break // Use first headsign found
			}
		}
	}
}
