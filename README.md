# COTA Google Maps Mashup

A Google Maps mashup with real-time bus information from COTA.

Visit https://joeshaw.org/cota-bus to see it in action.

This draws heavily from my [MBTA
mashup](https://github.com/joeshaw/mbta-bus) from 2009.  The MBTA
provides a JSON API which removes the need for any server side
components, but COTA's real-time data is only available via
[GTFS-realtime](https://developers.google.com/transit/gtfs-realtime/),
a protobuf-based API.  A Go server takes the (static, occasionally
updated) GTFS data for COTA and updates it with periodic fetches of
the GTFS-realtime data.

This module is pulled into my blog via git submodules.

## Running the server

To build:

```bash
go build .
```

To run:
```bash
./cota-bus`
```

### Configuration Options

The server supports the following command-line flags:

- `-listen`: HTTP listen address (default: `:18080`)
- `-gtfs-url`: URL to GTFS static feed (default: `https://www.cota.com/data/cota.gtfs.zip`)
- `-trip-updates-url`: URL to GTFS-realtime trip updates feed (default: `https://gtfs-rt.cota.vontascloud.com/TMGTFSRealTimeWebService/TripUpdate/TripUpdates.pb`)
- `-vehicles-url`: URL to GTFS-realtime vehicle positions feed (default: `https://gtfs-rt.cota.vontascloud.com/TMGTFSRealTimeWebService/Vehicle/VehiclePositions.pb`)

### Development

If you need to regenerate code for an updated GTFS-realtime protobuf file, you can do so with:

```bash
go generate ./internal/realtime
```

This requires the `protoc` compiler and `proto-gen-go` plugin to be installed.

Run the tests with:

```bash
go test ./...
```
