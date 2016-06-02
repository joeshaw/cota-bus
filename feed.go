package main

//go:generate protoc --gogo_out=import_path=main:. gtfs-realtime.proto

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
)

type agency struct {
	ID   string `db:"agency_id" json:"agency_id"`
	Name string `db:"agency_name" json:"name"`
	URL  string `db:"agency_url" json:"url"`
}

type route struct {
	ID        string `db:"route_id" json:"route_id"`
	LongName  string `db:"route_long_name" json:"long_name"`
	ShortName string `db:"route_short_name" json:"short_name"`
}

type stop struct {
	ID        string `db:"stop_id" json:"stop_id"`
	Name      string `db:"stop_name" json:"name"`
	Latitude  string `db:"stop_lat" json:"latitude"`
	Longitude string `db:"stop_lon" json:"longitude"`
}

type vehicle struct {
	ID           string  `db:"vehicle_id" json:"vehicle_id"`
	Name         string  `db:"vehicle_label" json:"name"`
	TripHeadsign string  `db:"trip_headsign" json:"trip_headsign"`
	RouteID      string  `db:"route_id" json:"route_id"`
	Latitude     float32 `db:"latitude" json:"latitude"`
	Longitude    float32 `db:"longitude" json:"longitude"`
}

type prediction struct {
	StopID       string `db:"stop_id" json:"stop_id"`
	RouteID      string `db:"route_id" json:"route_id"`
	TripHeadsign string `db:"trip_headsign" json:"trip_headsign"`
	ArrivalTime  uint64 `db:"arrival_time" json:"arrival_time"`
}

func fetchProtobuf(url string) (*FeedMessage, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, errors.New(resp.Status)
	}

	d, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var msg FeedMessage
	if err := proto.Unmarshal(d, &msg); err != nil {
		return nil, err
	}

	return &msg, nil
}

func updateVehiclePositions(db *sqlx.DB) error {
	msg, err := fetchProtobuf("http://realtime.cota.com/TMGTFSRealTimeWebService/Vehicle/VehiclePositions.pb")
	if err != nil {
		return err
	}

	tx, err := db.Beginx()
	if err != nil {
		return err
	}
	defer tx.Commit()

	if _, err := tx.Exec(`DELETE FROM vehicle_positions`); err != nil {
		tx.Rollback()
		return err
	}

	const q = `INSERT INTO vehicle_positions (
		       vehicle_id,
		       vehicle_label,
		       trip_id,
		       latitude,
		       longitude)
		   VALUES (?, ?, ?, ?, ?)`

	for _, ent := range msg.Entity {
		v := ent.Vehicle

		if _, err := tx.Exec(
			q,
			v.Vehicle.GetId(),
			v.Vehicle.GetLabel(),
			v.Trip.GetTripId(),
			v.Position.GetLatitude(),
			v.Position.GetLongitude(),
		); err != nil {
			tx.Rollback()
			return err
		}
	}

	return nil
}

func updateTripUpdates(db *sqlx.DB) error {
	msg, err := fetchProtobuf("http://realtime.cota.com/TMGTFSRealTimeWebService/TripUpdate/TripUpdates.pb")
	if err != nil {
		return err
	}

	tx, err := db.Beginx()
	if err != nil {
		return err
	}
	defer tx.Commit()

	if _, err := tx.Exec(`DELETE FROM stop_time_updates`); err != nil {
		tx.Rollback()
		return err
	}

	const q = `INSERT INTO stop_time_updates (
		       stop_id,
		       trip_id,
		       arrival_time,
		       vehicle_id)
		   VALUES (?, ?, ?, ?)`

	for _, ent := range msg.Entity {
		tu := ent.TripUpdate

		for _, u := range tu.StopTimeUpdate {
			if _, err := tx.Exec(
				q,
				u.GetStopId(),
				tu.Trip.GetTripId(),
				u.Arrival.GetTime(),
				tu.Vehicle.GetId(),
			); err != nil {
				tx.Rollback()
				return err
			}
		}
	}

	return nil
}

func updateRealtimeData(db *sqlx.DB) {
	for {
		updateVehiclePositions(db)
		updateTripUpdates(db)
		time.Sleep(60 * time.Second)
	}
}

func main() {
	db, err := sqlx.Open("sqlite3", "cota-gtfs.db")
	if err != nil {
		log.Fatal(err)
	}

	go updateRealtimeData(db)

	http.HandleFunc("/agencies", func(rw http.ResponseWriter, req *http.Request) {
		agencies := []agency{}
		err := db.Select(&agencies, "SELECT agency_id, agency_name, agency_url FROM agency")
		if err != nil {
			http.Error(rw, err.Error(), http.StatusInternalServerError)
			return
		}

		rw.Header().Set("Content-Type", "application/json")
		rw.Header().Set("Access-Control-Allow-Origin", "*")
		enc := json.NewEncoder(rw)
		enc.Encode(agencies)
	})

	http.HandleFunc("/cota/routes", func(rw http.ResponseWriter, req *http.Request) {
		routes := []route{}
		err := db.Select(&routes, "SELECT route_id, route_long_name, route_short_name FROM routes WHERE agency_id = 'COTA' ORDER BY route_long_name")
		if err != nil {
			http.Error(rw, err.Error(), http.StatusInternalServerError)
			return
		}

		rw.Header().Set("Content-Type", "application/json")
		rw.Header().Set("Access-Control-Allow-Origin", "*")
		enc := json.NewEncoder(rw)
		enc.Encode(routes)
	})

	http.HandleFunc("/cota/stops", func(rw http.ResponseWriter, req *http.Request) {
		stops := []stop{}

		q := "SELECT DISTINCT stops.stop_id, stops.stop_name, stops.stop_lat, stops.stop_lon FROM stops"

		var err error
		if route := req.FormValue("route"); route != "" {
			q += ` INNER JOIN stop_times ON stops.stop_id = stop_times.stop_id
			       INNER JOIN trips ON stop_times.trip_id = trips.trip_id
			       WHERE trips.route_id = ?`
			err = db.Select(&stops, q, route)
		} else {
			err = db.Select(&stops, q)
		}

		if err != nil {
			http.Error(rw, err.Error(), http.StatusInternalServerError)
			return
		}

		rw.Header().Set("Content-Type", "application/json")
		rw.Header().Set("Access-Control-Allow-Origin", "*")
		enc := json.NewEncoder(rw)
		enc.Encode(stops)

	})

	http.HandleFunc("/cota/vehicles", func(rw http.ResponseWriter, req *http.Request) {
		vehicles := []vehicle{}

		q := `SELECT vp.vehicle_id, vp.vehicle_label, trips.trip_headsign, trips.route_id, vp.latitude, vp.longitude
		      FROM vehicle_positions AS vp
		      INNER JOIN trips ON vp.trip_id = trips.trip_id`

		var err error
		if route := req.FormValue("route"); route != "" {
			q += ` WHERE trips.route_id = ?`
			err = db.Select(&vehicles, q, route)
		} else {
			err = db.Select(&vehicles, q)
		}

		if err != nil {
			http.Error(rw, err.Error(), http.StatusInternalServerError)
			return
		}

		rw.Header().Set("Content-Type", "application/json")
		rw.Header().Set("Access-Control-Allow-Origin", "*")
		enc := json.NewEncoder(rw)
		enc.Encode(vehicles)
	})

	http.HandleFunc("/cota/predictions", func(rw http.ResponseWriter, req *http.Request) {
		stop := req.FormValue("stop")
		if stop == "" {
			http.Error(rw, "Missing stop argument", http.StatusBadRequest)
			return
		}

		predictions := []prediction{}

		const q = `SELECT stu.stop_id, trips.trip_headsign, trips.route_id, min(stu.arrival_time)-? as arrival_time
			   FROM stop_time_updates AS stu
			   INNER JOIN trips ON stu.trip_id = trips.trip_id
			   WHERE stu.stop_id = ? AND stu.arrival_time >= ?
			   GROUP BY stu.stop_id, trips.route_id`
		now := time.Now().Unix()
		if err := db.Select(&predictions, q, now, stop, now); err != nil {
			http.Error(rw, err.Error(), http.StatusInternalServerError)
			return
		}

		rw.Header().Set("Content-Type", "application/json")
		rw.Header().Set("Access-Control-Allow-Origin", "*")
		enc := json.NewEncoder(rw)
		enc.Encode(predictions)
	})

	log.Println("Starting server on port 8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
