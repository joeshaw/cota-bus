package api

import (
	"net/http"
	"sort"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/joeshaw/cota-bus/internal/filter"
	"github.com/joeshaw/cota-bus/internal/models"
)

// handleRoutes handles the routes collection endpoint
func (s *Server) handleRoutes(w http.ResponseWriter, r *http.Request) {
	options := filter.NewOptions(r.URL.Query())

	// Get all routes
	routes := s.store.GetAllRoutes()

	// Apply filters
	if options.HasFilter("id") {
		idFilter := options.GetFilter("id")
		routes = filter.Filter(routes, func(route *models.Route) bool {
			for _, id := range idFilter {
				if route.ID == id {
					return true
				}
			}
			return false
		})
	}

	if options.HasFilter("type") {
		typeFilter := options.GetFilter("type")
		routes = filter.Filter(routes, func(route *models.Route) bool {
			for _, typeStr := range typeFilter {
				if typeVal, err := strconv.Atoi(typeStr); err == nil && route.Type == typeVal {
					return true
				}
			}
			return false
		})
	}

	// Sort routes by ID
	sort.Slice(routes, func(i, j int) bool {
		return routes[i].ID < routes[j].ID
	})

	// Convert to JSON:API resources
	resources := make([]Resource, len(routes))
	for i, route := range routes {
		resources[i] = routeToResource(route)
	}

	// Create response
	response := Response{
		Data: resources,
		Links: map[string]string{
			"self": "/routes",
		},
	}

	s.sendResponse(w, response)
}

// handleRoute handles the route detail endpoint
func (s *Server) handleRoute(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	route := s.store.GetRoute(id)
	if route == nil {
		s.sendErrorResponse(w, http.StatusNotFound, "Route not found")
		return
	}

	options := filter.NewOptions(r.URL.Query())

	// Create response
	response := Response{
		Data: routeToResource(route),
		Links: map[string]string{
			"self": "/routes/" + id,
		},
	}

	// Include stops if requested
	if options.HasInclude("stops") {
		stops := s.store.GetStopsByRoute(route.ID)
		included := make([]Resource, len(stops))
		for i, stop := range stops {
			included[i] = stopToResource(stop)
		}
		response.Included = included
	}

	s.sendResponse(w, response)
}

// routeToResource converts a Route model to a JSON:API resource
func routeToResource(route *models.Route) Resource {
	attributes := map[string]interface{}{
		"short_name":  route.ShortName,
		"long_name":   route.LongName,
		"description": route.Description,
		"type":        route.Type,
		"color":       route.Color,
		"text_color":  route.TextColor,
		"sort_order":  route.SortOrder,
	}

	// Only include direction attributes if they have data
	if len(route.DirectionNames) > 0 {
		attributes["direction_names"] = route.DirectionNames
	}
	if len(route.DirectionDestinations) > 0 {
		attributes["direction_destinations"] = route.DirectionDestinations
	}

	return Resource{
		Type:       "route",
		ID:         route.ID,
		Attributes: attributes,
		Links: map[string]string{
			"self": "/routes/" + route.ID,
		},
	}
}
