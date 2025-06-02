---
title: COTA Tracker
description: Where's the bus?
date: 2016-05-14T16:00:00-04:00
tags: [cota, bus, maps, mashup]
layout: bus
---

<div style="margin: 0 20px;">
  <select id="route-select" style="margin: 10px 0; font-size: 16px;">
    <option value="">Select Route</option>
  </select>
  <div id="map-canvas"></div>
  <div id="directions-container" style="margin: 10px 0;"></div>
</div>

## About

[COTA](http://www.cota.com/) provides real-time tracking of its buses.

This is a little mashup which takes that data from COTA and plots it
on [Google Maps](https://maps.google.com).  Choose a route from the
pull down menu below and markers will appear with the latest locations
of its buses.  Markers will update with the latest information from
COTA every 10 seconds.

On the backend is a small Go server that is populated with [COTA's
GTFS data](http://www.cota.com/data), and updates vehicle and
prediction data from COTA's GTFS-Realtime feed.  It exposes an API
that mimics the MBTA v3 API.  The front-end uses modern JavaScript
and the Google Maps Javascript API.

If you have any questions or comments, please feel free to
[email](mailto:joe@joeshaw.org) me.

This was based on a [similar mashup I made for the MBTA](/mbta-bus)
back in 2009.  This is version 2, released 1 June 2025.  The first
version of this was released on 1 June 2016, shortly after COTA made
the data available.

<script async src="https://maps.googleapis.com/maps/api/js?key=AIzaSyBuDLNN2zftYHZtrxnAwOcVYUF0zgJQukU&libraries=geometry,marker&loading=async&callback=initMap"></script>
<script type="text/javascript" src="cota-bus.js"></script>

