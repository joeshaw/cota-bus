package api

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/joeshaw/cota-bus/internal/filter"
	"github.com/joeshaw/cota-bus/internal/models"
)

// handleShapes handles the shapes collection endpoint
func (s *Server) handleShapes(w http.ResponseWriter, r *http.Request) {
	options := filter.NewOptions(r.URL.Query())
	
	// Get all shapes or apply filters
	var shapes map[string][]*models.Shape
	
	if options.HasFilter("id") {
		// Filter by shape IDs
		idFilter := options.GetFilter("id")
		filteredShapes := make(map[string][]*models.Shape)
		
		for _, id := range idFilter {
			if shapePoints := s.store.GetShapesByID(id); len(shapePoints) > 0 {
				filteredShapes[id] = shapePoints
			}
		}
		
		shapes = filteredShapes
	} else if options.HasFilter("route") {
		// Filter by route ID
		routeFilter := options.GetFilter("route")
		filteredShapes := make(map[string][]*models.Shape)
		
		for _, routeID := range routeFilter {
			// Get trips for the route
			trips := s.store.GetTripsByRoute(routeID)
			
			// Get shape IDs from trips
			for _, trip := range trips {
				if trip.ShapeID != "" {
					if shapePoints := s.store.GetShapesByID(trip.ShapeID); len(shapePoints) > 0 {
						filteredShapes[trip.ShapeID] = shapePoints
					}
				}
			}
		}
		
		shapes = filteredShapes
	} else {
		// Get all shape IDs from store
		shapes = s.store.GetAllShapes()
	}
	
	// Convert to JSON:API resources
	resources := make([]Resource, 0, len(shapes))
	for id, points := range shapes {
		resources = append(resources, shapeToResource(id, points))
	}
	
	// Create response
	response := Response{
		Data: resources,
		Links: map[string]string{
			"self": "/shapes",
		},
	}
	
	s.sendResponse(w, response)
}

// handleShape handles the shape detail endpoint
func (s *Server) handleShape(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	
	shapePoints := s.store.GetShapesByID(id)
	if len(shapePoints) == 0 {
		s.sendErrorResponse(w, http.StatusNotFound, "Shape not found")
		return
	}
	
	// Create response
	response := Response{
		Data: shapeToResource(id, shapePoints),
		Links: map[string]string{
			"self": "/shapes/" + id,
		},
	}
	
	s.sendResponse(w, response)
}

// shapeToResource converts a Shape model to a JSON:API resource
// It converts the shape points into a polyline format and includes the raw points
func shapeToResource(id string, points []*models.Shape) Resource {
	// Extract coordinates for polyline encoding
	coords := make([][2]float64, len(points))
	pointsData := make([]map[string]interface{}, len(points))
	
	for i, point := range points {
		coords[i] = [2]float64{point.Latitude, point.Longitude}
		
		// Create individual point data
		pointsData[i] = map[string]interface{}{
			"latitude":      point.Latitude,
			"longitude":     point.Longitude,
			"sequence":      point.Sequence,
			"dist_traveled": point.DistTraveled,
		}
	}
	
	// Encode the polyline
	polyline := encodePolyline(coords)
	
	return Resource{
		Type: "shape",
		ID:   id,
		Attributes: map[string]interface{}{
			"polyline": polyline,
			"points":   pointsData,
		},
		Links: map[string]string{
			"self": "/shapes/" + id,
		},
	}
}

// encodePolyline encodes a series of coordinates into a Google polyline format
// Polyline encoding algorithm: https://developers.google.com/maps/documentation/utilities/polylinealgorithm
func encodePolyline(coords [][2]float64) string {
	if len(coords) == 0 {
		return ""
	}
	
	result := make([]byte, 0, len(coords)*4)
	
	var prevLat, prevLng int
	for _, coord := range coords {
		lat5 := int(coord[0] * 1e5)
		lng5 := int(coord[1] * 1e5)
		
		// Encode latitude
		result = appendEncoded(result, lat5-prevLat)
		// Encode longitude
		result = appendEncoded(result, lng5-prevLng)
		
		prevLat, prevLng = lat5, lng5
	}
	
	return string(result)
}

// appendEncoded appends an encoded integer to the byte slice
func appendEncoded(result []byte, value int) []byte {
	value = value << 1
	if value < 0 {
		value = ^value
	}
	
	for value >= 0x20 {
		result = append(result, byte((0x20|(value&0x1f))+63))
		value >>= 5
	}
	
	result = append(result, byte(value+63))
	return result
}