package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/joeshaw/cota-bus/internal/filter"
	"github.com/joeshaw/cota-bus/internal/models"
	"github.com/joeshaw/cota-bus/internal/store"
)

// TestIndexEndpoint tests the index endpoint
func TestIndexEndpoint(t *testing.T) {
	// Create a test store
	testStore := store.NewStore()
	
	// Create the API server
	server := NewServer(testStore)
	
	// Create a test request
	req, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}
	
	// Create a response recorder
	rr := httptest.NewRecorder()
	
	// Serve the request
	handler := server.Router()
	handler.ServeHTTP(rr, req)
	
	// Check the status code
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}
	
	// Check the content type
	contentType := rr.Header().Get("Content-Type")
	if contentType != "application/vnd.api+json" {
		t.Errorf("handler returned wrong content type: got %v want %v", contentType, "application/vnd.api+json")
	}
	
	// Parse the response
	var response Response
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Errorf("error parsing response: %v", err)
	}
	
	// Check that the response has the expected links
	if response.Links == nil {
		t.Errorf("response links missing")
	} else {
		expectedLinks := []string{"routes", "stops", "trips", "vehicles", "predictions"}
		for _, link := range expectedLinks {
			if _, ok := response.Links[link]; !ok {
				t.Errorf("missing link: %s", link)
			}
		}
	}
}

// TestRoutesEndpoint tests the routes endpoint
func TestRoutesEndpoint(t *testing.T) {
	// Create a test store
	testStore := store.NewStore()
	
	// Add a test route
	testRoute := &models.Route{
		ID:        "test-route",
		ShortName: "1",
		LongName:  "Test Route",
		Type:      3,
	}
	testStore.AddRoute(testRoute)
	
	// Create the API server
	server := NewServer(testStore)
	
	// Create a test request
	req, err := http.NewRequest("GET", "/routes", nil)
	if err != nil {
		t.Fatal(err)
	}
	
	// Create a response recorder
	rr := httptest.NewRecorder()
	
	// Serve the request
	handler := server.Router()
	handler.ServeHTTP(rr, req)
	
	// Check the status code
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}
	
	// Check the content type
	contentType := rr.Header().Get("Content-Type")
	if contentType != "application/vnd.api+json" {
		t.Errorf("handler returned wrong content type: got %v want %v", contentType, "application/vnd.api+json")
	}
	
	// Parse the response
	var response Response
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Errorf("error parsing response: %v", err)
	}
	
	// Check that the response has the expected data
	data, ok := response.Data.([]interface{})
	if !ok {
		t.Fatalf("response data is not an array")
	}
	
	if len(data) != 1 {
		t.Errorf("unexpected number of routes: got %d want 1", len(data))
	}
	
	route, ok := data[0].(map[string]interface{})
	if !ok {
		t.Fatalf("route data is not an object")
	}
	
	if route["id"] != "test-route" {
		t.Errorf("unexpected route ID: got %v want %v", route["id"], "test-route")
	}
	
	if route["type"] != "route" {
		t.Errorf("unexpected resource type: got %v want %v", route["type"], "route")
	}
}

// TestFilterFunction tests the filter.Filter function
func TestFilterFunction(t *testing.T) {
	// Create some test data
	numbers := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	
	// Filter for even numbers
	evenNumbers := filter.Filter(numbers, func(n int) bool {
		return n%2 == 0
	})
	
	// Check the result
	expectedEven := []int{2, 4, 6, 8, 10}
	if len(evenNumbers) != len(expectedEven) {
		t.Errorf("unexpected number of even numbers: got %d want %d", len(evenNumbers), len(expectedEven))
	}
	
	for i, n := range evenNumbers {
		if n != expectedEven[i] {
			t.Errorf("unexpected value at index %d: got %d want %d", i, n, expectedEven[i])
		}
	}
	
	// Filter for numbers greater than 5
	greaterThanFive := filter.Filter(numbers, func(n int) bool {
		return n > 5
	})
	
	// Check the result
	expectedGreaterThanFive := []int{6, 7, 8, 9, 10}
	if len(greaterThanFive) != len(expectedGreaterThanFive) {
		t.Errorf("unexpected number of numbers greater than 5: got %d want %d", len(greaterThanFive), len(expectedGreaterThanFive))
	}
	
	for i, n := range greaterThanFive {
		if n != expectedGreaterThanFive[i] {
			t.Errorf("unexpected value at index %d: got %d want %d", i, n, expectedGreaterThanFive[i])
		}
	}
}