package models

import (
	"time"
)

// Agency represents a transit agency
type Agency struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	URL      string `json:"url"`
	Timezone string `json:"timezone"`
}

// Route represents a transit route
type Route struct {
	ID                    string   `json:"id"`
	AgencyID              string   `json:"agency_id,omitzero"`
	ShortName             string   `json:"short_name"`
	LongName              string   `json:"long_name"`
	Description           string   `json:"description,omitzero"`
	Type                  int      `json:"type"`
	Color                 string   `json:"color,omitzero"`
	TextColor             string   `json:"text_color,omitzero"`
	SortOrder             int      `json:"sort_order,omitzero"`
	DirectionNames        []string `json:"direction_names,omitzero"`
	DirectionDestinations []string `json:"direction_destinations,omitzero"`
}

// Stop represents a transit stop
type Stop struct {
	ID                 string  `json:"id"`
	Code               string  `json:"code,omitzero"`
	Name               string  `json:"name"`
	Description        string  `json:"description,omitzero"`
	Latitude           float64 `json:"latitude"`
	Longitude          float64 `json:"longitude"`
	ZoneID             string  `json:"zone_id,omitzero"`
	URL                string  `json:"url,omitzero"`
	LocationType       int     `json:"location_type,omitzero"`
	ParentStation      string  `json:"parent_station,omitzero"`
	WheelchairBoarding int     `json:"wheelchair_boarding,omitzero"`
}

// Trip represents a transit trip
type Trip struct {
	ID                   string `json:"id"`
	RouteID              string `json:"route_id"`
	ServiceID            string `json:"service_id"`
	Headsign             string `json:"headsign,omitzero"`
	ShortName            string `json:"short_name,omitzero"`
	DirectionID          int    `json:"direction_id,omitzero"`
	BlockID              string `json:"block_id,omitzero"`
	ShapeID              string `json:"shape_id,omitzero"`
	WheelchairAccessible int    `json:"wheelchair_accessible,omitzero"`
	BikesAllowed         int    `json:"bikes_allowed,omitzero"`
}

// StopTime represents a scheduled stop time for a trip
type StopTime struct {
	TripID            string  `json:"trip_id"`
	ArrivalTime       string  `json:"arrival_time"`
	DepartureTime     string  `json:"departure_time"`
	StopID            string  `json:"stop_id"`
	StopSequence      int     `json:"stop_sequence"`
	StopHeadsign      string  `json:"stop_headsign,omitzero"`
	PickupType        int     `json:"pickup_type,omitzero"`
	DropOffType       int     `json:"drop_off_type,omitzero"`
	ShapeDistTraveled float64 `json:"shape_dist_traveled,omitzero"`
	Timepoint         int     `json:"timepoint,omitzero"`
}

// Calendar represents service dates
type Calendar struct {
	ServiceID string `json:"service_id"`
	Monday    int    `json:"monday"`
	Tuesday   int    `json:"tuesday"`
	Wednesday int    `json:"wednesday"`
	Thursday  int    `json:"thursday"`
	Friday    int    `json:"friday"`
	Saturday  int    `json:"saturday"`
	Sunday    int    `json:"sunday"`
	StartDate string `json:"start_date"`
	EndDate   string `json:"end_date"`
}

// CalendarDate represents exceptions to the calendar
type CalendarDate struct {
	ServiceID     string `json:"service_id"`
	Date          string `json:"date"`
	ExceptionType int    `json:"exception_type"`
}

// Shape represents a route shape
type Shape struct {
	ID           string  `json:"id"`
	Latitude     float64 `json:"latitude"`
	Longitude    float64 `json:"longitude"`
	Sequence     int     `json:"sequence"`
	DistTraveled float64 `json:"dist_traveled,omitzero"`
}

// Vehicle represents real-time vehicle position
type Vehicle struct {
	ID                  string    `json:"id"`
	TripID              string    `json:"trip_id"`
	RouteID             string    `json:"route_id"`
	DirectionID         int       `json:"direction_id,omitzero"`
	Latitude            float64   `json:"latitude"`
	Longitude           float64   `json:"longitude"`
	Bearing             float64   `json:"bearing,omitzero"`
	Speed               float64   `json:"speed,omitzero"`
	StopID              string    `json:"stop_id,omitzero"`
	UpdatedAt           time.Time `json:"updated_at"`
	VehicleLabel        string    `json:"vehicle_label,omitzero"`
	CurrentStopSequence int       `json:"current_stop_sequence,omitzero"`
	CurrentStatus       string    `json:"current_status,omitzero"`
	CongestionLevel     string    `json:"congestion_level,omitzero"`
	OccupancyStatus     string    `json:"occupancy_status,omitzero"`
}

// Prediction represents a real-time arrival/departure prediction
type Prediction struct {
	ID             string    `json:"id"`
	TripID         string    `json:"trip_id"`
	StopID         string    `json:"stop_id"`
	RouteID        string    `json:"route_id"`
	DirectionID    int       `json:"direction_id,omitzero"`
	ArrivalTime    time.Time `json:"arrival_time,omitzero"`
	DepartureTime  time.Time `json:"departure_time,omitzero"`
	Status         string    `json:"status,omitzero"`
	StopSequence   int       `json:"stop_sequence,omitzero"`
}

// JSONAPIObject represents a JSON:API formatted object
type JSONAPIObject struct {
	Type          string                  `json:"type"`
	ID            string                  `json:"id"`
	Attributes    map[string]interface{}  `json:"attributes"`
	Links         map[string]string       `json:"links,omitzero"`
	Relationships map[string]Relationship `json:"relationships,omitzero"`
}

// Relationship represents a JSON:API relationship
type Relationship struct {
	Data  interface{}       `json:"data"`
	Links map[string]string `json:"links,omitzero"`
}

// RelationshipData represents a JSON:API relationship data object
type RelationshipData struct {
	Type string `json:"type"`
	ID   string `json:"id"`
}
