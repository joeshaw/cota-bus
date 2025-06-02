package main

//go:generate protoc --gogo_out=import_path=main:. gtfs-realtime.proto

import (
	"archive/zip"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"maps"
	"net/http"
	"os"
	"slices"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/gorilla/mux"
)

const (
	cotaGTFSURL             = "https://www.cota.com/data/cota.gtfs.zip"
	cotaVehiclePositionsURL = "https://gtfs-rt.cota.vontascloud.com/TMGTFSRealTimeWebService/Vehicle/VehiclePositions.pb"
	cotaTripUpdatesURL      = "https://gtfs-rt.cota.vontascloud.com/TMGTFSRealTimeWebService/TripUpdate/TripUpdates.pb"

	realtimeUpdateInterval = 30 * time.Second
	gtfsUpdateInterval     = 24 * time.Hour
)

// Data structures matching MBTA v3 API format
type APIResponse[T any] struct {
	Data T `json:"data"`
}

type Route struct {
	ID         string          `json:"id"`
	Type       string          `json:"type"`
	Attributes RouteAttributes `json:"attributes"`
}

type RouteAttributes struct {
	Color                 string   `json:"color,omitempty"`
	Description           string   `json:"description,omitempty"`
	LongName              string   `json:"long_name"`
	ShortName             string   `json:"short_name"`
	TextColor             string   `json:"text_color,omitempty"`
	Type                  int      `json:"type"`
	DirectionDestinations []string `json:"direction_destinations,omitempty"`
}

type Stop struct {
	ID         string         `json:"id"`
	Type       string         `json:"type"`
	Attributes StopAttributes `json:"attributes"`
}

type StopAttributes struct {
	Description        string  `json:"description,omitempty"`
	Latitude           float64 `json:"latitude"`
	Longitude          float64 `json:"longitude"`
	Name               string  `json:"name"`
	PlatformCode       string  `json:"platform_code,omitempty"`
	PlatformName       string  `json:"platform_name,omitempty"`
	WheelchairBoarding int     `json:"wheelchair_boarding"`
}

type Vehicle struct {
	ID            string                `json:"id"`
	Type          string                `json:"type"`
	Attributes    VehicleAttributes     `json:"attributes"`
	Relationships *VehicleRelationships `json:"relationships,omitempty"`
}

type VehicleRelationships struct {
	Trip *VehicleTrip `json:"trip,omitempty"`
	Stop *VehicleStop `json:"stop,omitempty"`
}

type VehicleTrip struct {
	Data *VehicleTripData `json:"data,omitempty"`
}

type VehicleTripData struct {
	ID   string `json:"id"`
	Type string `json:"type"`
}

type VehicleStop struct {
	Data *VehicleStopData `json:"data,omitempty"`
}

type VehicleStopData struct {
	ID   string `json:"id"`
	Type string `json:"type"`
}

type VehicleAttributes struct {
	Bearing             float32 `json:"bearing,omitempty"`
	CurrentStatus       string  `json:"current_status,omitempty"`
	CurrentStopSequence int     `json:"current_stop_sequence"`
	DirectionID         int     `json:"direction_id"`
	Label               string  `json:"label,omitempty"`
	Latitude            float32 `json:"latitude"`
	Longitude           float32 `json:"longitude"`
	Speed               float32 `json:"speed,omitempty"`
	UpdatedAt           string  `json:"updated_at"`
	RouteID             string  `json:"route_id,omitempty"`
	Headsign            string  `json:"headsign,omitempty"`
}

type Shape struct {
	ID         string          `json:"id"`
	Type       string          `json:"type"`
	Attributes ShapeAttributes `json:"attributes"`
}

type ShapeAttributes struct {
	Polyline string `json:"polyline"`
}

type Prediction struct {
	ID            string                   `json:"id"`
	Type          string                   `json:"type"`
	Attributes    PredictionAttributes     `json:"attributes"`
	Relationships *PredictionRelationships `json:"relationships,omitempty"`
}

type PredictionRelationships struct {
	Stop *PredictionStop `json:"stop,omitempty"`
}

type PredictionStop struct {
	Data *PredictionStopData `json:"data,omitempty"`
}

type PredictionStopData struct {
	ID   string `json:"id"`
	Type string `json:"type"`
}

type PredictionAttributes struct {
	ArrivalTime   *string `json:"arrival_time"`
	DepartureTime *string `json:"departure_time"`
	DirectionID   int     `json:"direction_id"`
	Schedule      string  `json:"schedule_relationship"`
	Status        string  `json:"status,omitempty"`
	StopSequence  int     `json:"stop_sequence"`
}

// Internal data structures
type GTFSData struct {
	routes     map[string]*Route
	stops      map[string]*Stop
	shapes     map[string]*Shape
	trips      map[string]*Trip
	stopTimes  map[string][]*StopTime
	tripStops  map[string][]*StopTime // trip_id -> stop_times (ordered by stop_sequence)
	routeStops map[string][]string    // route_id -> stop_ids
}

type Trip struct {
	ID        string
	RouteID   string
	ServiceID string
	ShapeID   string
	Headsign  string
	Direction int
}

type StopTime struct {
	TripID        string
	StopID        string
	StopSequence  int
	ArrivalTime   string
	DepartureTime string
}

type RealtimeData struct {
	vehicles        map[string]*Vehicle
	predictions     map[string][]*Prediction // stop_id -> predictions
	tripPredictions map[string][]*Prediction // trip_id -> predictions
	lastUpdate      time.Time
}

type Server struct {
	gtfsMu     sync.RWMutex
	gtfs       *GTFSData
	realtimeMu sync.RWMutex
	realtime   *RealtimeData
}

func NewServer() *Server {
	return &Server{
		gtfs: &GTFSData{
			routes:     make(map[string]*Route),
			stops:      make(map[string]*Stop),
			shapes:     make(map[string]*Shape),
			trips:      make(map[string]*Trip),
			stopTimes:  make(map[string][]*StopTime),
			routeStops: make(map[string][]string),
		},
		realtime: &RealtimeData{
			vehicles:    make(map[string]*Vehicle),
			predictions: make(map[string][]*Prediction),
		},
	}
}

func main() {
	server := NewServer()

	// Initial GTFS load
	log.Println("Loading initial GTFS data...")
	if err := server.loadGTFS(); err != nil {
		log.Printf("Failed to load GTFS data: %v", err)
	}

	// Initial realtime data load
	log.Println("Loading initial realtime data...")
	if err := server.updateRealtimeData(); err != nil {
		log.Printf("Failed to load realtime data: %v", err)
	}

	// Start background routines
	go server.gtfsUpdater()
	go server.realtimeUpdater()

	// Setup HTTP routes
	r := mux.NewRouter()

	// Enable CORS for all routes
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			w.Header().Set("Content-Type", "application/json")

			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}

			next.ServeHTTP(w, r)
		})
	})

	// API routes matching MBTA v3 style
	r.HandleFunc("/routes", server.handleRoutes).Methods("GET")
	r.HandleFunc("/routes/{id}", server.handleRoute).Methods("GET")
	r.HandleFunc("/stops", server.handleStops).Methods("GET")
	r.HandleFunc("/stops/{id}", server.handleStop).Methods("GET")
	r.HandleFunc("/vehicles", server.handleVehicles).Methods("GET")
	r.HandleFunc("/shapes", server.handleShapes).Methods("GET")
	r.HandleFunc("/shapes/{id}", server.handleShape).Methods("GET")
	r.HandleFunc("/predictions", server.handlePredictions).Methods("GET")

	log.Println("Starting server on :18080")
	log.Fatal(http.ListenAndServe(":18080", r))
}

func (s *Server) gtfsUpdater() {
	ticker := time.NewTicker(gtfsUpdateInterval)
	defer ticker.Stop()

	for range ticker.C {
		log.Println("Updating GTFS data...")
		if err := s.loadGTFS(); err != nil {
			log.Printf("Failed to update GTFS data: %v", err)
		}
	}
}

func (s *Server) realtimeUpdater() {
	ticker := time.NewTicker(realtimeUpdateInterval)
	defer ticker.Stop()

	for range ticker.C {
		if err := s.updateRealtimeData(); err != nil {
			log.Printf("Failed to update realtime data: %v", err)
		}
	}
}

func (s *Server) loadGTFS() error {
	// Try to download fresh GTFS data from COTA
	if err := s.downloadAndParseGTFS(); err != nil {
		log.Printf("Failed to download fresh GTFS data: %v", err)
		log.Println("Falling back to local GTFS file...")

		// Fall back to local file
		if _, err := os.Stat("cota.gtfs.zip"); err == nil {
			return s.parseGTFS("cota.gtfs.zip")
		} else {
			return fmt.Errorf("no GTFS data available: %w", err)
		}
	}
	return nil
}

func (s *Server) downloadAndParseGTFS() error {
	log.Printf("Trying to download GTFS from: %s", cotaGTFSURL)

	resp, err := http.Get(cotaGTFSURL)
	if err != nil {
		return fmt.Errorf("download error from %s: %w", cotaGTFSURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d from %s", resp.StatusCode, cotaGTFSURL)
	}

	// Create temporary file
	tmpFile, err := os.CreateTemp("", "cota-gtfs-*.zip")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	// Copy downloaded data to temp file
	if _, err := io.Copy(tmpFile, resp.Body); err != nil {
		return fmt.Errorf("failed to save GTFS file: %w", err)
	}

	// Try to parse the downloaded file
	if err := s.parseGTFS(tmpFile.Name()); err != nil {
		return fmt.Errorf("failed to parse GTFS from %s: %w", cotaGTFSURL, err)
	}

	// Success! Save a copy locally for future fallback
	if err := s.saveLocalCopy(tmpFile.Name()); err != nil {
		log.Printf("Failed to save local copy: %v", err)
	}

	log.Printf("Successfully downloaded and parsed GTFS from: %s", cotaGTFSURL)
	return nil
}

func (s *Server) saveLocalCopy(sourcePath string) error {
	source, err := os.Open(sourcePath)
	if err != nil {
		return err
	}
	defer source.Close()

	dest, err := os.Create("cota.gtfs.zip")
	if err != nil {
		return err
	}
	defer dest.Close()

	_, err = io.Copy(dest, source)
	return err
}

func (s *Server) parseGTFS(filename string) error {
	log.Printf("Opening GTFS file: %s", filename)
	reader, err := zip.OpenReader(filename)
	if err != nil {
		return fmt.Errorf("failed to open zip file: %w", err)
	}
	defer reader.Close()

	newGTFS := &GTFSData{
		routes:     make(map[string]*Route),
		stops:      make(map[string]*Stop),
		shapes:     make(map[string]*Shape),
		trips:      make(map[string]*Trip),
		stopTimes:  make(map[string][]*StopTime),
		tripStops:  make(map[string][]*StopTime),
		routeStops: make(map[string][]string),
	}

	log.Printf("Found %d files in GTFS zip", len(reader.File))
	for _, file := range reader.File {
		log.Printf("Processing file: %s", file.Name)
		switch file.Name {
		case "routes.txt":
			if err := s.parseRoutes(file, newGTFS); err != nil {
				return fmt.Errorf("failed to parse routes: %w", err)
			}
		case "stops.txt":
			if err := s.parseStops(file, newGTFS); err != nil {
				return fmt.Errorf("failed to parse stops: %w", err)
			}
		case "shapes.txt":
			if err := s.parseShapes(file, newGTFS); err != nil {
				return fmt.Errorf("failed to parse shapes: %w", err)
			}
		case "trips.txt":
			if err := s.parseTrips(file, newGTFS); err != nil {
				return fmt.Errorf("failed to parse trips: %w", err)
			}
		case "stop_times.txt":
			if err := s.parseStopTimes(file, newGTFS); err != nil {
				return fmt.Errorf("failed to parse stop times: %w", err)
			}
		}
	}

	s.buildRouteStopMappings(newGTFS)

	for _, route := range newGTFS.routes {
		directionDestinations := make([]string, 2)
		for _, trip := range newGTFS.trips {
			if trip.RouteID == route.ID {
				directionDestinations[trip.Direction] = cleanCOTADestination(trip.Headsign)
			}
		}
		route.Attributes.DirectionDestinations = directionDestinations
	}

	log.Printf("Parsed %d routes, %d stops, %d shapes - atomically swapping data",
		len(newGTFS.routes), len(newGTFS.stops), len(newGTFS.shapes))

	// Swap in the new data
	s.gtfsMu.Lock()
	s.gtfs = newGTFS
	s.gtfsMu.Unlock()

	log.Printf("Successfully loaded %d routes, %d stops, %d shapes",
		len(newGTFS.routes), len(newGTFS.stops), len(newGTFS.shapes))

	return nil
}

func (s *Server) parseRoutes(file *zip.File, gtfs *GTFSData) error {
	return s.parseCSV(file, func(record []string, headers map[string]int) error {
		route := &Route{
			ID:   record[headers["route_id"]],
			Type: "route",
			Attributes: RouteAttributes{
				ShortName: record[headers["route_short_name"]],
				LongName:  record[headers["route_long_name"]],
				Type:      3, // Bus
			},
		}

		if colorIdx, exists := headers["route_color"]; exists && colorIdx < len(record) {
			route.Attributes.Color = record[colorIdx]
		}
		if textColorIdx, exists := headers["route_text_color"]; exists && textColorIdx < len(record) {
			route.Attributes.TextColor = record[textColorIdx]
		}

		gtfs.routes[route.ID] = route
		return nil
	})
}

func (s *Server) parseStops(file *zip.File, gtfs *GTFSData) error {
	return s.parseCSV(file, func(record []string, headers map[string]int) error {
		lat, _ := strconv.ParseFloat(record[headers["stop_lat"]], 64)
		lon, _ := strconv.ParseFloat(record[headers["stop_lon"]], 64)

		stop := &Stop{
			ID:   record[headers["stop_id"]],
			Type: "stop",
			Attributes: StopAttributes{
				Name:      record[headers["stop_name"]],
				Latitude:  lat,
				Longitude: lon,
			},
		}

		if descIdx, exists := headers["stop_desc"]; exists && descIdx < len(record) {
			stop.Attributes.Description = record[descIdx]
		}

		gtfs.stops[stop.ID] = stop
		return nil
	})
}

func (s *Server) parseShapes(file *zip.File, gtfs *GTFSData) error {
	shapePoints := make(map[string][]ShapePoint)

	err := s.parseCSV(file, func(record []string, headers map[string]int) error {
		shapeID := record[headers["shape_id"]]
		lat, _ := strconv.ParseFloat(record[headers["shape_pt_lat"]], 64)
		lon, _ := strconv.ParseFloat(record[headers["shape_pt_lon"]], 64)
		seq, _ := strconv.Atoi(record[headers["shape_pt_sequence"]])

		shapePoints[shapeID] = append(shapePoints[shapeID], ShapePoint{
			Lat:      lat,
			Lon:      lon,
			Sequence: seq,
		})
		return nil
	})
	if err != nil {
		return err
	}

	// Convert shape points to polylines
	for shapeID, points := range shapePoints {
		// Sort by sequence
		sort.Slice(points, func(i, j int) bool {
			return points[i].Sequence < points[j].Sequence
		})

		// Generate polyline
		polyline := encodePolyline(points)

		gtfs.shapes[shapeID] = &Shape{
			ID:   shapeID,
			Type: "shape",
			Attributes: ShapeAttributes{
				Polyline: polyline,
			},
		}
	}

	return nil
}

func (s *Server) parseTrips(file *zip.File, gtfs *GTFSData) error {
	return s.parseCSV(file, func(record []string, headers map[string]int) error {
		directionID, _ := strconv.Atoi(record[headers["direction_id"]])

		trip := &Trip{
			ID:        record[headers["trip_id"]],
			RouteID:   record[headers["route_id"]],
			ServiceID: record[headers["service_id"]],
			Headsign:  record[headers["trip_headsign"]],
			Direction: directionID,
		}

		if shapeIdx, exists := headers["shape_id"]; exists && shapeIdx < len(record) {
			trip.ShapeID = record[shapeIdx]
		}

		gtfs.trips[trip.ID] = trip
		return nil
	})
}

func (s *Server) parseStopTimes(file *zip.File, gtfs *GTFSData) error {
	return s.parseCSV(file, func(record []string, headers map[string]int) error {
		stopSequence, _ := strconv.Atoi(record[headers["stop_sequence"]])

		stopTime := &StopTime{
			TripID:        record[headers["trip_id"]],
			StopID:        record[headers["stop_id"]],
			StopSequence:  stopSequence,
			ArrivalTime:   record[headers["arrival_time"]],
			DepartureTime: record[headers["departure_time"]],
		}

		// Store by stop ID (existing functionality)
		gtfs.stopTimes[stopTime.StopID] = append(gtfs.stopTimes[stopTime.StopID], stopTime)
		
		// Store by trip ID for vehicle stop relationship lookups
		gtfs.tripStops[stopTime.TripID] = append(gtfs.tripStops[stopTime.TripID], stopTime)
		
		return nil
	})
}

func (s *Server) buildRouteStopMappings(gtfs *GTFSData) {
	log.Println("Building route-stop mappings...")

	// First, build a trip -> stops mapping for efficiency
	tripStops := make(map[string][]string)
	for stopID, stopTimes := range gtfs.stopTimes {
		for _, st := range stopTimes {
			tripStops[st.TripID] = append(tripStops[st.TripID], stopID)
		}
	}

	log.Printf("Built trip-stop mappings for %d trips", len(tripStops))

	// Now build route -> stops mapping
	routeStops := make(map[string]map[string]bool)
	for tripID, trip := range gtfs.trips {
		if routeStops[trip.RouteID] == nil {
			routeStops[trip.RouteID] = make(map[string]bool)
		}

		// Add all stops for this trip to the route
		if stops, exists := tripStops[tripID]; exists {
			for _, stopID := range stops {
				routeStops[trip.RouteID][stopID] = true
			}
		}
	}

	for routeID, stopSet := range routeStops {
		// Use slices package to convert map keys to slice
		stopIDs := slices.Collect(maps.Keys(stopSet))
		gtfs.routeStops[routeID] = stopIDs
	}
	log.Printf("Route-stop mappings complete: %d routes", len(gtfs.routeStops))
}

type ShapePoint struct {
	Lat      float64
	Lon      float64
	Sequence int
}

func (s *Server) parseCSV(file *zip.File, rowHandler func([]string, map[string]int) error) error {
	rc, err := file.Open()
	if err != nil {
		return err
	}
	defer rc.Close()

	reader := csv.NewReader(rc)

	// Read headers
	headers, err := reader.Read()
	if err != nil {
		return err
	}

	// Create header index map
	headerMap := make(map[string]int)
	for i, header := range headers {
		headerMap[header] = i
	}

	// Read data rows
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		if err := rowHandler(record, headerMap); err != nil {
			return err
		}
	}

	return nil
}

func (s *Server) updateRealtimeData() error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Update vehicle positions
	if err := s.updateVehiclePositions(ctx); err != nil {
		log.Printf("Failed to update vehicle positions: %v", err)
	}

	// Update trip updates/predictions
	if err := s.updateTripUpdates(ctx); err != nil {
		log.Printf("Failed to update trip updates: %v", err)
	}

	s.realtimeMu.Lock()
	s.realtime.lastUpdate = time.Now()
	s.realtimeMu.Unlock()

	return nil
}

func (s *Server) updateVehiclePositions(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, "GET", cotaVehiclePositionsURL, nil)
	if err != nil {
		return err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var feed FeedMessage
	if err := proto.Unmarshal(data, &feed); err != nil {
		return err
	}

	vehicles := make(map[string]*Vehicle)

	// Process vehicle positions
	for _, entity := range feed.Entity {
		if entity.Vehicle == nil {
			continue
		}
		vp := entity.Vehicle
		if vp.Position == nil || vp.Vehicle == nil {
			continue
		}
		vehicle := &Vehicle{
			ID:   vp.Vehicle.GetId(),
			Type: "vehicle",
			Attributes: VehicleAttributes{
				Label:     vp.Vehicle.GetLabel(),
				Latitude:  vp.Position.GetLatitude(),
				Longitude: vp.Position.GetLongitude(),
				UpdatedAt: time.Unix(int64(feed.Header.GetTimestamp()), 0).Format(time.RFC3339),
			},
		}
		if vp.Position.Bearing != nil {
			vehicle.Attributes.Bearing = vp.Position.GetBearing()
		}
		if vp.Position.Speed != nil {
			vehicle.Attributes.Speed = vp.Position.GetSpeed()
		}
		if vp.CurrentStatus != nil {
			vehicle.Attributes.CurrentStatus = vp.CurrentStatus.String()
		}
		if vp.Trip != nil {
			tripID := vp.Trip.GetTripId()
			vehicle.Attributes.RouteID = vp.Trip.GetRouteId()

			// Initialize relationships
			vehicle.Relationships = &VehicleRelationships{}

			// Populate trip relationship data
			vehicle.Relationships.Trip = &VehicleTrip{
				Data: &VehicleTripData{
					ID:   tripID,
					Type: "trip",
				},
			}

			// Set direction_id from GTFS trip if available
			s.gtfsMu.RLock()
			if trip, ok := s.gtfs.trips[tripID]; ok {
				vehicle.Attributes.DirectionID = trip.Direction
			}
			s.gtfsMu.RUnlock()
		}

		// Populate stop relationship data if available
		stopID := vp.GetStopId()
		if stopID != "" {
			if vehicle.Relationships == nil {
				vehicle.Relationships = &VehicleRelationships{}
			}
			vehicle.Relationships.Stop = &VehicleStop{
				Data: &VehicleStopData{
					ID:   stopID,
					Type: "stop",
				},
			}
			log.Printf("Vehicle %s has stop relationship: %s", vehicle.ID, stopID)
		} else if vp.Trip != nil && vp.CurrentStopSequence != nil {
			// If no direct stop ID, try to derive it from trip and stop sequence
			tripID := vp.Trip.GetTripId()
			stopSequence := int(vp.GetCurrentStopSequence())
			
			s.gtfsMu.RLock()
			if stopTimes, exists := s.gtfs.tripStops[tripID]; exists && stopSequence > 0 && stopSequence <= len(stopTimes) {
				// Get the stop ID for this stop sequence (arrays are 0-indexed, sequences are 1-indexed)
				derivedStopID := stopTimes[stopSequence-1].StopID
				if vehicle.Relationships == nil {
					vehicle.Relationships = &VehicleRelationships{}
				}
				vehicle.Relationships.Stop = &VehicleStop{
					Data: &VehicleStopData{
						ID:   derivedStopID,
						Type: "stop",
					},
				}
				log.Printf("Vehicle %s derived stop relationship from trip %s sequence %d: %s", vehicle.ID, tripID, stopSequence, derivedStopID)
			}
			s.gtfsMu.RUnlock()
		}

		// Set current stop sequence if available
		if vp.CurrentStopSequence != nil {
			vehicle.Attributes.CurrentStopSequence = int(vp.GetCurrentStopSequence())
		}
		vehicles[vehicle.ID] = vehicle
	}

	// Swap in the new data
	s.realtimeMu.Lock()
	s.realtime.vehicles = vehicles
	s.realtimeMu.Unlock()

	return nil
}

func (s *Server) updateTripUpdates(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, "GET", cotaTripUpdatesURL, nil)
	if err != nil {
		return err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var feed FeedMessage
	if err := proto.Unmarshal(data, &feed); err != nil {
		return err
	}

	predictions := make(map[string][]*Prediction)
	tripPredictions := make(map[string][]*Prediction)

	// Process trip updates
	for _, entity := range feed.Entity {
		if entity.TripUpdate == nil {
			continue
		}

		tu := entity.TripUpdate
		if tu.Trip == nil {
			continue
		}

		tripID := tu.Trip.GetTripId()

		for _, stu := range tu.StopTimeUpdate {
			if stu.StopId == nil {
				continue
			}

			stopID := stu.GetStopId()

			// Get direction_id from GTFS trip data, fallback to realtime data
			// Highly dubious.
			directionID := int(tu.Trip.GetDirectionId())
			s.gtfsMu.RLock()
			if trip, ok := s.gtfs.trips[tripID]; ok {
				directionID = trip.Direction
			}
			s.gtfsMu.RUnlock()
			prediction := &Prediction{
				ID:   fmt.Sprintf("%s-%s", tripID, stopID),
				Type: "prediction",
				Attributes: PredictionAttributes{
					StopSequence: int(stu.GetStopSequence()),
					DirectionID:  directionID,
				},
				Relationships: &PredictionRelationships{
					Stop: &PredictionStop{
						Data: &PredictionStopData{
							ID:   stopID,
							Type: "stop",
						},
					},
				},
			}

			var predictionTime time.Time

			if stu.Arrival != nil && stu.Arrival.Time != nil {
				predictionTime = time.Unix(int64(stu.Arrival.GetTime()), 0)
				arrivalTime := predictionTime.Format(time.RFC3339)
				prediction.Attributes.ArrivalTime = &arrivalTime
			}

			if stu.Departure != nil && stu.Departure.Time != nil {
				departureTime := time.Unix(int64(stu.Departure.GetTime()), 0)
				departureTimeStr := departureTime.Format(time.RFC3339)
				prediction.Attributes.DepartureTime = &departureTimeStr
				// Use departure time if arrival time is not available, or if departure is later
				if predictionTime.IsZero() || departureTime.After(predictionTime) {
					predictionTime = departureTime
				}
			}

			// Filter out predictions that are more than the update interval in the past
			if !predictionTime.IsZero() {
				now := time.Now()
				if predictionTime.Before(now.Add(-realtimeUpdateInterval)) {
					continue // Skip this prediction as it's too old
				}
			}

			predictions[stopID] = append(predictions[stopID], prediction)
			tripPredictions[tripID] = append(tripPredictions[tripID], prediction)
		}
	}

	// Swap in the new data
	s.realtimeMu.Lock()
	s.realtime.predictions = predictions
	s.realtime.tripPredictions = tripPredictions
	s.realtimeMu.Unlock()

	return nil
}

// HTTP handlers implementing MBTA v3-like API
func (s *Server) handleRoutes(w http.ResponseWriter, r *http.Request) {
	s.gtfsMu.RLock()
	routes := slices.Collect(maps.Values(s.gtfs.routes))
	s.gtfsMu.RUnlock()

	slices.SortFunc(routes, func(a, b *Route) int {
		return strings.Compare(a.ID, b.ID)
	})

	response := APIResponse[[]*Route]{Data: routes}
	json.NewEncoder(w).Encode(response)
}

func (s *Server) handleRoute(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	routeID := vars["id"]

	s.gtfsMu.RLock()
	route, exists := s.gtfs.routes[routeID]
	s.gtfsMu.RUnlock()

	if !exists {
		http.Error(w, "Route not found", http.StatusNotFound)
		return
	}

	response := APIResponse[*Route]{Data: route}
	json.NewEncoder(w).Encode(response)
}

func (s *Server) handleStops(w http.ResponseWriter, r *http.Request) {
	routeFilter := r.URL.Query().Get("filter[route]")

	s.gtfsMu.RLock()
	defer s.gtfsMu.RUnlock()

	var stops []*Stop

	if routeFilter != "" {
		// Filter stops by route
		if stopIDs, exists := s.gtfs.routeStops[routeFilter]; exists {
			for _, stopID := range stopIDs {
				if stop, exists := s.gtfs.stops[stopID]; exists {
					stops = append(stops, stop)
				}
			}
		}
	} else {
		// Return all stops
		stops = make([]*Stop, 0, len(s.gtfs.stops))
		for _, stop := range s.gtfs.stops {
			stops = append(stops, stop)
		}
	}

	response := APIResponse[[]*Stop]{Data: stops}
	json.NewEncoder(w).Encode(response)
}

func (s *Server) handleStop(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	stopID := vars["id"]

	s.gtfsMu.RLock()
	stop, exists := s.gtfs.stops[stopID]
	s.gtfsMu.RUnlock()

	if !exists {
		http.Error(w, "Stop not found", http.StatusNotFound)
		return
	}

	response := APIResponse[*Stop]{Data: stop}
	json.NewEncoder(w).Encode(response)
}

func (s *Server) handleVehicles(w http.ResponseWriter, r *http.Request) {
	routeFilter := r.URL.Query().Get("filter[route]")

	s.realtimeMu.RLock()
	vehicles := make([]*Vehicle, 0, len(s.realtime.vehicles))

	if routeFilter != "" {
		// Filter vehicles by route
		for _, vehicle := range s.realtime.vehicles {
			if vehicle.Attributes.RouteID == routeFilter {
				vehicles = append(vehicles, vehicle)
			}
		}
	} else {
		for _, vehicle := range s.realtime.vehicles {
			vehicles = append(vehicles, vehicle)
		}
	}
	s.realtimeMu.RUnlock()

	response := APIResponse[[]*Vehicle]{Data: vehicles}
	json.NewEncoder(w).Encode(response)
}

func (s *Server) handleShapes(w http.ResponseWriter, r *http.Request) {
	routeFilter := r.URL.Query().Get("filter[route]")

	s.gtfsMu.RLock()
	defer s.gtfsMu.RUnlock()

	var shapes []*Shape

	if routeFilter != "" {
		// Find shapes for route via trips
		for _, trip := range s.gtfs.trips {
			if trip.RouteID == routeFilter && trip.ShapeID != "" {
				if shape, exists := s.gtfs.shapes[trip.ShapeID]; exists {
					shapes = append(shapes, shape)
				}
			}
		}
	} else {
		shapes = make([]*Shape, 0, len(s.gtfs.shapes))
		for _, shape := range s.gtfs.shapes {
			shapes = append(shapes, shape)
		}
	}

	response := APIResponse[[]*Shape]{Data: shapes}
	json.NewEncoder(w).Encode(response)
}

func (s *Server) handleShape(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	shapeID := vars["id"]

	s.gtfsMu.RLock()
	defer s.gtfsMu.RUnlock()

	shape, exists := s.gtfs.shapes[shapeID]
	if !exists {
		http.Error(w, "Shape not found", http.StatusNotFound)
		return
	}

	response := APIResponse[*Shape]{Data: shape}
	json.NewEncoder(w).Encode(response)
}

func (s *Server) handlePredictions(w http.ResponseWriter, r *http.Request) {
	routeFilter := r.URL.Query().Get("filter[route]")
	tripFilter := r.URL.Query().Get("filter[trip]")

	s.realtimeMu.RLock()
	defer s.realtimeMu.RUnlock()

	var predictions []*Prediction

	if tripFilter != "" {
		if preds, exists := s.realtime.tripPredictions[tripFilter]; exists {
			predictions = append(predictions, preds...)
		}
	} else if routeFilter != "" {
		s.gtfsMu.RLock()
		for _, trip := range s.gtfs.trips {
			if trip.RouteID == routeFilter {
				if preds, exists := s.realtime.tripPredictions[trip.ID]; exists {
					predictions = append(predictions, preds...)
				}
			}
		}
		s.gtfsMu.RUnlock()
	} else {
		for _, stopPreds := range s.realtime.predictions {
			predictions = append(predictions, stopPreds...)
		}
	}

	response := APIResponse[[]*Prediction]{Data: predictions}
	json.NewEncoder(w).Encode(response)
}

// Polyline encoding (Google's polyline algorithm)
func encodePolyline(points []ShapePoint) string {
	if len(points) == 0 {
		return ""
	}

	var encoded strings.Builder
	var prevLat, prevLon int32

	for _, point := range points {
		lat := int32(point.Lat * 1e5)
		lon := int32(point.Lon * 1e5)

		deltaLat := lat - prevLat
		deltaLon := lon - prevLon

		encoded.WriteString(encodeSignedNumber(deltaLat))
		encoded.WriteString(encodeSignedNumber(deltaLon))

		prevLat = lat
		prevLon = lon
	}

	return encoded.String()
}

func encodeSignedNumber(num int32) string {
	shifted := num << 1
	if num < 0 {
		shifted = ^shifted
	}
	return encodeUnsignedNumber(uint32(shifted))
}

func encodeUnsignedNumber(num uint32) string {
	var encoded strings.Builder

	for num >= 0x20 {
		encoded.WriteByte(byte((num&0x1F)|0x20) + 63)
		num >>= 5
	}
	encoded.WriteByte(byte(num) + 63)

	return encoded.String()
}

// cleanCOTADestination extracts the destination from COTA's verbose destination format
// Input: "10 EAST AND WEST BROAD TO WESTWOODS PARK AND RIDE"
// Output: "TO WESTWOODS PARK AND RIDE"
func cleanCOTADestination(destination string) string {
	// Look for " TO " separator (with spaces to avoid false matches)
	toIndex := strings.Index(destination, " TO ")
	if toIndex == -1 {
		// If no " TO " found, return the original string
		return destination
	}

	// Extract everything from " TO " onward
	cleaned := strings.TrimSpace(destination[toIndex:])
	if cleaned == "" {
		// If nothing after route prefix, return original
		return destination
	}

	return cleaned
}
