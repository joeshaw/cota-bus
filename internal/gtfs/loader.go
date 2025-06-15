package gtfs

import (
	"archive/zip"
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/joeshaw/cota-bus/internal/models"
	"github.com/joeshaw/cota-bus/internal/store"
)

// Loader handles loading GTFS static data
type Loader struct {
	url   string
	store *store.Store
}

// NewLoader creates a new GTFS loader
func NewLoader(url string, store *store.Store) *Loader {
	return &Loader{
		url:   url,
		store: store,
	}
}

// Load downloads and processes GTFS data
func (l *Loader) Load() error {
	log.Println("Starting GTFS data load from", l.url)

	// Create a temporary file to store the ZIP
	tmpFile, err := os.CreateTemp("", "gtfs_*.zip")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	// Download the ZIP file
	resp, err := http.Get(l.url)
	if err != nil {
		return fmt.Errorf("failed to download GTFS data: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	// Copy the response body to the temp file
	_, err = io.Copy(tmpFile, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to write GTFS data to temp file: %v", err)
	}

	// Open the ZIP file
	zipReader, err := zip.OpenReader(tmpFile.Name())
	if err != nil {
		return fmt.Errorf("failed to open ZIP file: %v", err)
	}
	defer zipReader.Close()

	// Clear existing static data
	l.store.Clear()

	// Process each file in the ZIP
	for _, file := range zipReader.File {
		switch filepath.Base(file.Name) {
		case "agency.txt":
			if err := l.processAgency(file); err != nil {
				return fmt.Errorf("failed to process agency data: %v", err)
			}
		case "routes.txt":
			if err := l.processRoutes(file); err != nil {
				return fmt.Errorf("failed to process routes data: %v", err)
			}
		case "stops.txt":
			if err := l.processStops(file); err != nil {
				return fmt.Errorf("failed to process stops data: %v", err)
			}
		case "trips.txt":
			if err := l.processTrips(file); err != nil {
				return fmt.Errorf("failed to process trips data: %v", err)
			}
		case "stop_times.txt":
			if err := l.processStopTimes(file); err != nil {
				return fmt.Errorf("failed to process stop times data: %v", err)
			}
		case "calendar.txt":
			if err := l.processCalendar(file); err != nil {
				return fmt.Errorf("failed to process calendar data: %v", err)
			}
		case "calendar_dates.txt":
			if err := l.processCalendarDates(file); err != nil {
				return fmt.Errorf("failed to process calendar dates data: %v", err)
			}
		case "shapes.txt":
			if err := l.processShapes(file); err != nil {
				return fmt.Errorf("failed to process shapes data: %v", err)
			}
		}
	}

	// Build the stopsByRoute index from the loaded data
	l.store.BuildStopsByRoute()

	// Build direction information for routes
	l.store.BuildRouteDirections()

	// Update last update time
	l.store.SetLastStaticUpdate(time.Now())

	log.Println("GTFS data load completed successfully")
	return nil
}

// processAgency processes agency.txt
func (l *Loader) processAgency(file *zip.File) error {
	records, err := readCSV(file)
	if err != nil {
		return err
	}

	for _, record := range records {
		agency := &models.Agency{
			ID:       getString(record, "agency_id"),
			Name:     getString(record, "agency_name"),
			URL:      getString(record, "agency_url"),
			Timezone: getString(record, "agency_timezone"),
		}
		l.store.AddAgency(agency)
	}
	return nil
}

// processRoutes processes routes.txt
func (l *Loader) processRoutes(file *zip.File) error {
	records, err := readCSV(file)
	if err != nil {
		return err
	}

	for _, record := range records {
		route := &models.Route{
			ID:          getString(record, "route_id"),
			AgencyID:    getString(record, "agency_id"),
			ShortName:   getString(record, "route_short_name"),
			LongName:    getString(record, "route_long_name"),
			Description: getString(record, "route_desc"),
			Type:        getInt(record, "route_type"),
			Color:       getString(record, "route_color"),
			TextColor:   getString(record, "route_text_color"),
			SortOrder:   getInt(record, "route_sort_order"),
		}
		l.store.AddRoute(route)
	}
	return nil
}

// processStops processes stops.txt
func (l *Loader) processStops(file *zip.File) error {
	records, err := readCSV(file)
	if err != nil {
		return err
	}

	for _, record := range records {
		stop := &models.Stop{
			ID:                 getString(record, "stop_id"),
			Code:               getString(record, "stop_code"),
			Name:               getString(record, "stop_name"),
			Description:        getString(record, "stop_desc"),
			Latitude:           getFloat(record, "stop_lat"),
			Longitude:          getFloat(record, "stop_lon"),
			ZoneID:             getString(record, "zone_id"),
			URL:                getString(record, "stop_url"),
			LocationType:       getInt(record, "location_type"),
			ParentStation:      getString(record, "parent_station"),
			WheelchairBoarding: getInt(record, "wheelchair_boarding"),
		}
		l.store.AddStop(stop)
	}
	return nil
}

// processTrips processes trips.txt
func (l *Loader) processTrips(file *zip.File) error {
	records, err := readCSV(file)
	if err != nil {
		return err
	}

	for _, record := range records {
		trip := &models.Trip{
			ID:                   getString(record, "trip_id"),
			RouteID:              getString(record, "route_id"),
			ServiceID:            getString(record, "service_id"),
			Headsign:             getString(record, "trip_headsign"),
			ShortName:            getString(record, "trip_short_name"),
			DirectionID:          getInt(record, "direction_id"),
			BlockID:              getString(record, "block_id"),
			ShapeID:              getString(record, "shape_id"),
			WheelchairAccessible: getInt(record, "wheelchair_accessible"),
			BikesAllowed:         getInt(record, "bikes_allowed"),
		}
		l.store.AddTrip(trip)
	}
	return nil
}

// processStopTimes processes stop_times.txt
func (l *Loader) processStopTimes(file *zip.File) error {
	records, err := readCSV(file)
	if err != nil {
		return err
	}

	for _, record := range records {
		stopTime := &models.StopTime{
			TripID:            getString(record, "trip_id"),
			ArrivalTime:       getString(record, "arrival_time"),
			DepartureTime:     getString(record, "departure_time"),
			StopID:            getString(record, "stop_id"),
			StopSequence:      getInt(record, "stop_sequence"),
			StopHeadsign:      getString(record, "stop_headsign"),
			PickupType:        getInt(record, "pickup_type"),
			DropOffType:       getInt(record, "drop_off_type"),
			ShapeDistTraveled: getFloat(record, "shape_dist_traveled"),
			Timepoint:         getInt(record, "timepoint"),
		}
		l.store.AddStopTime(stopTime)
	}
	return nil
}

// processCalendar processes calendar.txt
func (l *Loader) processCalendar(file *zip.File) error {
	records, err := readCSV(file)
	if err != nil {
		return err
	}

	for _, record := range records {
		calendar := &models.Calendar{
			ServiceID: getString(record, "service_id"),
			Monday:    getInt(record, "monday"),
			Tuesday:   getInt(record, "tuesday"),
			Wednesday: getInt(record, "wednesday"),
			Thursday:  getInt(record, "thursday"),
			Friday:    getInt(record, "friday"),
			Saturday:  getInt(record, "saturday"),
			Sunday:    getInt(record, "sunday"),
			StartDate: getString(record, "start_date"),
			EndDate:   getString(record, "end_date"),
		}
		l.store.AddCalendar(calendar)
	}
	return nil
}

// processCalendarDates processes calendar_dates.txt
func (l *Loader) processCalendarDates(file *zip.File) error {
	records, err := readCSV(file)
	if err != nil {
		return err
	}

	for _, record := range records {
		calendarDate := &models.CalendarDate{
			ServiceID:     getString(record, "service_id"),
			Date:          getString(record, "date"),
			ExceptionType: getInt(record, "exception_type"),
		}
		l.store.AddCalendarDate(calendarDate)
	}
	return nil
}

// processShapes processes shapes.txt
func (l *Loader) processShapes(file *zip.File) error {
	records, err := readCSV(file)
	if err != nil {
		return err
	}

	for _, record := range records {
		shape := &models.Shape{
			ID:           getString(record, "shape_id"),
			Latitude:     getFloat(record, "shape_pt_lat"),
			Longitude:    getFloat(record, "shape_pt_lon"),
			Sequence:     getInt(record, "shape_pt_sequence"),
			DistTraveled: getFloat(record, "shape_dist_traveled"),
		}
		l.store.AddShape(shape)
	}
	return nil
}

// readCSV reads a CSV file from a ZIP entry and returns the data with headers
func readCSV(file *zip.File) ([]map[string]string, error) {
	fileReader, err := file.Open()
	if err != nil {
		return nil, err
	}
	defer fileReader.Close()

	csvReader := csv.NewReader(fileReader)

	// Read headers
	headers, err := csvReader.Read()
	if err != nil {
		return nil, err
	}

	// Read all records
	var records []map[string]string

	for {
		record, err := csvReader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		// Create a map of field name -> value
		fields := make(map[string]string)
		for i, header := range headers {
			if i < len(record) {
				fields[header] = record[i]
			}
		}

		records = append(records, fields)
	}

	return records, nil
}

// Helper functions for type conversion
func getString(record map[string]string, field string) string {
	return record[field]
}

func getInt(record map[string]string, field string) int {
	if val, ok := record[field]; ok && val != "" {
		if i, err := strconv.Atoi(val); err == nil {
			return i
		}
	}
	return 0
}

func getFloat(record map[string]string, field string) float64 {
	if val, ok := record[field]; ok && val != "" {
		if f, err := strconv.ParseFloat(val, 64); err == nil {
			return f
		}
	}
	return 0
}

func getBool(record map[string]string, field string) bool {
	if val, ok := record[field]; ok {
		val = strings.ToLower(val)
		return val == "1" || val == "true" || val == "t" || val == "yes" || val == "y"
	}
	return false
}
