---
title: COTA Tracker
description: Where's the bus?
date: 2016-05-14T16:00:00-04:00
tags: [cota, bus, maps, mashup]
layout: bus
---

<div style="margin: 0 20px;">
  <select id="option_list" style="margin: 10px 0; font-size: 16px;">
    <option value="">Select Route</option>
  </select>
  <div id="map_canvas"></div>
  <div id="marker_legend" style="margin: 10px 0;"></div>
</div>

## About

[COTA](http://www.cota.com/) provides real-time tracking of its buses.

This is a little mashup which takes that data from COTA and plots it
on [Google Maps](https://maps.google.com).  Choose a route from the
pull down menu below and markers will appear with the latest locations
of its buses.  Markers will update with the latest information from
COTA every 10 seconds.  Click on a stop marker to see predictions for
when the next vehicle will arrive.

On the backend is a small Go server that is populated with [COTA's
GTFS data](http://www.cota.com/data), and updates vehicle and
prediction data from COTA's GTFS-Realtime feed.  The front-end is the
same amateur 2009-era JavaScript that powers the [MBTA
edition](/mbta-bus).

If you have any questions or comments, please feel free to
[email](mailto:joe@joeshaw.org) or
[tweet](https://twitter.com/?status=@joeshaw%20) them to me.

This was based on a [similar mashup I made for the MBTA](/mbta-bus)
back in 2009.  The first version of this for COTA was released on 1
June 2016, shortly after COTA made the data available.

<script type="text/javascript" src="https://ajax.googleapis.com/ajax/libs/jquery/1.4.4/jquery.min.js"></script>
<script src="https://maps.google.com/maps/api/js?key=AIzaSyBuDLNN2zftYHZtrxnAwOcVYUF0zgJQukU&sensor=false" type="text/javascript"></script>
<script type="text/javascript" src="cota-bus.js"></script>

