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
