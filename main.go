package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/joeshaw/cota-bus/internal/api"
	"github.com/joeshaw/cota-bus/internal/gtfs"
	"github.com/joeshaw/cota-bus/internal/store"
	"github.com/joeshaw/cota-bus/internal/updater"
)

var (
	listenAddr     = flag.String("listen", ":18080", "HTTP listen address")
	gtfsURL        = flag.String("gtfs-url", "https://www.cota.com/data/cota.gtfs.zip", "URL to GTFS static feed")
	tripUpdatesURL = flag.String("trip-updates-url", "https://gtfs-rt.cota.vontascloud.com/TMGTFSRealTimeWebService/TripUpdate/TripUpdates.pb", "URL to GTFS-realtime trip updates feed")
	vehiclesURL    = flag.String("vehicles-url", "https://gtfs-rt.cota.vontascloud.com/TMGTFSRealTimeWebService/Vehicle/VehiclePositions.pb", "URL to GTFS-realtime vehicle positions feed")
)

func main() {
	flag.Parse()

	// Create data store
	dataStore := store.NewStore()

	// Set up initial GTFS static data load
	gtfsLoader := gtfs.NewLoader(*gtfsURL, dataStore)
	if err := gtfsLoader.Load(); err != nil {
		log.Fatalf("Failed to load initial GTFS data: %v", err)
	}

	// Set up GTFS-realtime updaters
	tripUpdater := updater.NewTripUpdater(*tripUpdatesURL, dataStore)
	vehicleUpdater := updater.NewVehicleUpdater(*vehiclesURL, dataStore)

	// Initial realtime data fetch
	if err := tripUpdater.Update(); err != nil {
		log.Printf("Failed to load initial trip data: %v", err)
	}
	if err := vehicleUpdater.Update(); err != nil {
		log.Printf("Failed to load initial vehicle data: %v", err)
	}

	// Set up API server
	apiServer := api.NewServer(dataStore)
	server := &http.Server{
		Addr:    *listenAddr,
		Handler: apiServer.Router(),
	}

	// Handle graceful shutdown
	done := make(chan struct{})
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	var wg sync.WaitGroup

	// Start GTFS static data updater (daily)
	wg.Add(1)
	go func() {
		defer wg.Done()
		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				if err := gtfsLoader.Load(); err != nil {
					log.Printf("Failed to update GTFS data: %v", err)
				} else {
					log.Println("GTFS data updated successfully")
				}
			case <-done:
				return
			}
		}
	}()

	// Start GTFS-realtime updaters
	wg.Add(1)
	go func() {
		defer wg.Done()
		ticker := time.NewTicker(15 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				if err := tripUpdater.Update(); err != nil {
					log.Printf("Failed to update trip data: %v", err)
				}
				if err := vehicleUpdater.Update(); err != nil {
					log.Printf("Failed to update vehicle data: %v", err)
				}
			case <-done:
				return
			}
		}
	}()

	// Start server
	wg.Add(1)
	go func() {
		defer wg.Done()
		log.Printf("Server listening on %s", *listenAddr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// Wait for termination signal
	<-quit
	log.Println("Shutting down server...")

	// Signal all goroutines to stop
	close(done)

	// Gracefully shut down server
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	// Wait for all goroutines to complete
	wg.Wait()
	log.Println("Server exited properly")
}
