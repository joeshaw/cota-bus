#!/bin/bash
sqlite3 -echo cota-gtfs.db <<EOF
.mode csv

.import cota-gtfs/agency.txt agency
.import cota-gtfs/calendar.txt calendar
.import cota-gtfs/calendar_dates.txt calendar_dates
.import cota-gtfs/fare_attributes.txt fare_attributes
.import cota-gtfs/fare_rules.txt fare_rules
.import cota-gtfs/routes.txt routes
.import cota-gtfs/shapes.txt shapes
.import cota-gtfs/stop_times.txt stop_times
.import cota-gtfs/stops.txt stops
.import cota-gtfs/trips.txt trips

CREATE INDEX agency_id_idx ON agency (agency_id);
CREATE INDEX routes_agency_id_idx ON routes (agency_id);
CREATE INDEX stops_id_idx ON stops (stop_id);
CREATE INDEX stop_times_stop_id_idx ON stop_times (stop_id);
CREATE INDEX stop_times_trip_id_idx ON stop_times (trip_id);
CREATE INDEX trips_id_idx ON trips (trip_id);
CREATE INDEX trips_route_id_idx ON trips (route_id);

CREATE TABLE vehicle_positions (
    vehicle_id string PRIMARY KEY,
    vehicle_label string,
    trip_id string,
    latitude string,
    longitude string
);

CREATE INDEX vehicle_positions_trip_id_idx ON vehicle_positions (trip_id);

CREATE TABLE stop_time_updates (
    stop_id string,
    trip_id string,
    arrival_time string,
    vehicle_id string
);

CREATE INDEX stop_time_updates_stop_id_idx ON stop_time_updates (stop_id);
CREATE INDEX stop_time_updates_trip_id_idx ON stop_time_updates (trip_id);
CREATE INDEX stop_time_updates_vehicle_id_idx ON stop_time_updates (vehicle_id);

EOF
