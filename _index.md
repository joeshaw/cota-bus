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
<script src="https://maps.google.com/maps/api/js?key=AIzaSyBuDLNN2zftYHZtrxnAwOcVYUF0zgJQukU&sensor=false" type="text/javascript"></script><script type="text/javascript">
  $(document).ready(function() {
    var useragent = navigator.userAgent;
    var map_canvas = document.getElementById("map_canvas");

    var mapOptions = {
      zoom: 12,
      center: new google.maps.LatLng(39.965912, -82.999939),
      mapTypeId: google.maps.MapTypeId.ROADMAP
    };

    if (useragent.indexOf('iPhone') != -1 || useragent.indexOf('Android') != -1 ) {
      map_canvas.style.width = '100%';
      map_canvas.style.height = '300px';
      mapOptions.gestureHandling = 'cooperative';
    } else {
      map_canvas.style.width = '100%';
      map_canvas.style.height = '600px';
      mapOptions.gestureHandling = 'greedy';
    }

    var map = new google.maps.Map(document.getElementById("map_canvas"), mapOptions);

    var base_url = "/cota-bus/api";

    var direction_data = [
      { icon: "/mbta-bus/images/red-dot.png",
        line_color: "#FF0000" },

      { icon: "/mbta-bus/images/blue-dot.png",
        line_color: "#0000FF" },

      { icon: "/mbta-bus/images/green-dot.png",
        line_color: "#00FF00" },

      { icon: "/mbta-bus/images/yellow-dot.png",
        line_color: "#FFFF00" },

      { icon: "/mbta-bus/images/orange-dot.png",
        line_color: "#FF7700" },

      { icon: "/mbta-bus/images/purple-dot.png",
        line_color: "#FF00FF" }
    ];

    // Some global variables
    var selected_route = "";
    var vehicle_markers = {};
    var stop_markers = [];
    var route_layer = null;
    var lines = [];
    var open_info_window = null;
    var updateIntervalID = 0;

    populateRouteList();

    // Update the markers any time the option box is changed, or
    // every 10 seconds as long as the window is visible.
    $("select").change(updateMarkers);
    if (!document.hidden) {
      updateIntervalID = setInterval(updateMarkers, 10000);
    }

    function handleVisibilityChange() {
      if (document.hidden && updateIntervalID) {
        clearInterval(updateIntervalID);
        updateIntervalID = 0;
      } else if (!document.hidden && !updateIntervalID) {
        updateMarkers();
        updateIntervalID = setInterval(updateMarkers, 10000);
      }
    }
    document.addEventListener("visibilitychange", handleVisibilityChange, false);

    function queryParams(qs) {
      qs = qs.split("+").join(" ");

      var params = {};
      var regexp = /[?&]?([^=]+)=([^&]*)/g;
      var tokens;
      while (tokens = regexp.exec(qs)) {
        params[decodeURIComponent(tokens[1])] = decodeURIComponent(tokens[2])
      }
      return params;
    }

    function populateRouteList() {
      $.getJSON(base_url + "/cota/routes",
        function(data) {
          for (var j = 0; j < data.length; j++) {
            var route = data[j]
            if (route.route_hide) {
              continue
            }

            $("#option_list").append('<option value="' + route.route_id + '">' + route.short_name + " – " + route.long_name + '</option>');
          }

          params = queryParams(document.location.search);
          if (params["route"]) {
            $("#option_list option[value=\"" + params["route"] + "\"]").attr('selected', 'selected');
            updateMarkers();
          }
        }
      );
    }

    function resetRouteMarkers() {
      for (var i = 0; i < stop_markers.length; i++) {
        stop_markers[i].setMap(null);
      }
      stop_markers = [];

      for (var i = 0; i < lines.length; i++) {
        lines[i].setMap(null);
      }
      lines = [];

      if (route_layer !== null) {
        route_layer.setMap(null);
        route_layer = null;
      }
    }

    function resetVehicleMarkers() {
      $("#marker_legend").empty();

      for (var vehicle_id in vehicle_markers) {
        vehicle_markers[vehicle_id].setMap(null);
      }
      vehicle_markers = {};
    }

    function updateMarkers() {
      var old_route = selected_route;
      selected_route = $("select option:selected").attr("value");

      if (selected_route != old_route) {
        resetRouteMarkers();
        resetVehicleMarkers();
      }

      if (selected_route == "") {
        return;
      }

      if (selected_route != old_route) {
        fetchRouteData(selected_route);
      }

      fetchVehicles(selected_route);
    }

    function fetchRouteData(route_id) {
      var stops_url = base_url + "/cota/stops?route=" + route_id;
      $.getJSON(stops_url, function(data) {
        var bounds = new google.maps.LatLngBounds();
        console.log(bounds);

        for (var i = 0; i < data.length; i++) {
          var stop = data[i];
          var latlong = placeStop(route_id, stop);
          bounds.extend(latlong);
        }

        route_layer = new google.maps.KmlLayer({
          url: "https://www.joeshaw.org/cota-bus/kml/" + route_id + ".kml",
          suppressInfoWindows: true,
          map: map
        });

        map.fitBounds(bounds)
      });
    }

    function placeStop(route_id, stop) {
      var latlong = new google.maps.LatLng(stop.latitude, stop.longitude);

      var marker = new google.maps.Marker({
        position: latlong,
        map: map,
        icon: "https://www.nextmuni.com/googleMap/images/stopMarkerRed.gif"
      });

      marker.stop_id = stop.stop_id;
      marker.infoContent = '<h3>' + stop.name + '</h3>';

      google.maps.event.addListener(marker, "click", function() {
        var info_window = new google.maps.InfoWindow({
          content: this.infoContent,
        });

        var prediction_url = base_url + "/cota/predictions?stop=" + stop.stop_id;
        $.getJSON(prediction_url, function(data) {
          var content = info_window.getContent();

          if (data.length == 0) {
            content += '<p>No vehicles expected.</p>';
          } else {
            content += '<p>Expected arrivals:';
            content += '<ul>';

            for (var i = 0; i < data.length; i++) {
              prediction = data[i];
              content += '<li>';
              if (prediction.arrival_time < 60) {
                content += prediction.arrival_time + ' seconds';
              } else {
                content += Math.floor(prediction.arrival_time/60) + ' minutes';
              }
              content += ': ' + prediction.trip_headsign;
              content += '</li>';
            }

            content += '</ul></p>';
          }

          info_window.setContent(content);
        });

        google.maps.event.addListener(info_window, "closeclick", function() {
          open_info_window = null;
        });

        if (open_info_window) {
          open_info_window.close();
        }
        open_info_window = info_window;

        info_window.open(map, this);
      });

      stop_markers.push(marker);
      return latlong;
    }

    function fetchVehicles(route_id) {
      var vehicle_url = base_url + "/cota/vehicles?route=" + route_id;
      $.getJSON(vehicle_url, function(data) {
        var new_markers = {}

        $("#marker_legend").empty();
        var trips = [];

        for (var i = 0; i < data.length; i++) {
          var vehicle = data[i];
          var latlong = new google.maps.LatLng(vehicle.latitude, vehicle.longitude);

          var trip_idx = trips.indexOf(vehicle.trip_headsign);
          if (trip_idx == -1) {
            trip_idx = trips.push(vehicle.trip_headsign) - 1;
            addLegend(direction_data[trip_idx].icon, vehicle.trip_headsign);
          }

          var marker = vehicle_markers[vehicle.vehicle_id];
          if (!marker) {
            var marker = new google.maps.Marker({
              position: latlong,
              map: map,
              icon: direction_data[trip_idx].icon
            });

            marker.infoContent = '<h3>' + vehicle.trip_headsign + '</h3>';

            google.maps.event.addListener(marker, "click", function() {
              var info_window = new google.maps.InfoWindow({
                content: this.infoContent,
              });

              google.maps.event.addListener(info_window, "closeclick", function() {
                open_info_window = null;
              });

              if (open_info_window) {
                open_info_window.close();
              }
              open_info_window = info_window;
              info_window.open(map, this);
            });
          } else {
            marker.setPosition(latlong);
            marker.setIcon(direction_data[trip_idx].icon);
          }

          new_markers[vehicle.vehicle_id] = marker;
          delete vehicle_markers[vehicle.vehicle_id];
        }

        // Buses no longer on the map
        for (var vehicle_id in vehicle_markers) {
          vehicle_markers[vehicle_id].setMap(null);
        }
        vehicle_markers = new_markers;
      });
    }

    function addLegend(icon, name) {
      $("#marker_legend").append('<img src="' + icon + '" style="display: inline">' + name);
    }
});
</script>

