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

## Updating

These are mostly notes for myself, as I never remember what I need to do to update things.

COTA occasionally updates its GTFS static feed.
That can be fetched from https://www.cota.com/data/.
Save the zip file and unzip into the `cota-gtfs` directory.

Regenerate the KML files by deleting the files in the `kml` directory and running `go run ../tools/route-kml.go ../cota.gtfs.zip`.

If the gtfs-realtime.proto file is updated (unlikely at this point),regenerate the protobuf with `go generate`.
Make sure gogoprotobuf is installed.

On the server side, pull the latest code.
If necessary, rebuild the server with `go install`.
Build a new `cota-gtfs.db` by running `gtfs-load.sh`.
Stop the server, move the old DB out of the way, move the new one into place, and restart the server.

This module is pulled into my blog via git submodules.

## Possible TODOs

The server is super hacky, but it was written so that it could
conceivably be generalized to any transit system that uses
GTFS-realtime.

I'd like to rewrite this to run on Fastly Compute using the KV store.
