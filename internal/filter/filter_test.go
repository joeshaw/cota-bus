package filter

import (
	"net/url"
	"testing"
)

func TestNewOptions(t *testing.T) {
	// Test with empty query
	emptyQuery := url.Values{}
	options := NewOptions(emptyQuery)
	if len(options.Filters) != 0 {
		t.Errorf("expected empty filters, got %v", options.Filters)
	}
	if len(options.Includes) != 0 {
		t.Errorf("expected empty includes, got %v", options.Includes)
	}
	if len(options.Fields) != 0 {
		t.Errorf("expected empty fields, got %v", options.Fields)
	}
	if len(options.Sort) != 0 {
		t.Errorf("expected empty sort, got %v", options.Sort)
	}

	// Test with filters
	query := url.Values{}
	query.Add("filter[id]", "1,2,3")
	query.Add("filter[type]", "3")
	query.Add("include", "stops,trips")
	query.Add("fields[route]", "short_name,long_name")
	query.Add("sort", "name,-id")

	options = NewOptions(query)

	// Check filters
	if len(options.Filters) != 2 {
		t.Errorf("expected 2 filters, got %d", len(options.Filters))
	}
	if _, ok := options.Filters["id"]; !ok {
		t.Errorf("expected filter[id], not found")
	}
	if _, ok := options.Filters["type"]; !ok {
		t.Errorf("expected filter[type], not found")
	}
	if options.Filters["id"][0] != "1,2,3" {
		t.Errorf("expected filter[id]=1,2,3, got %s", options.Filters["id"][0])
	}
	if options.Filters["type"][0] != "3" {
		t.Errorf("expected filter[type]=3, got %s", options.Filters["type"][0])
	}

	// Check includes
	if len(options.Includes) != 2 {
		t.Errorf("expected 2 includes, got %d", len(options.Includes))
	}
	if options.Includes[0] != "stops" {
		t.Errorf("expected includes[0]=stops, got %s", options.Includes[0])
	}
	if options.Includes[1] != "trips" {
		t.Errorf("expected includes[1]=trips, got %s", options.Includes[1])
	}

	// Check fields
	if len(options.Fields) != 1 {
		t.Errorf("expected 1 field, got %d", len(options.Fields))
	}
	if _, ok := options.Fields["route"]; !ok {
		t.Errorf("expected fields[route], not found")
	}
	if len(options.Fields["route"]) != 2 {
		t.Errorf("expected 2 fields for route, got %d", len(options.Fields["route"]))
	}
	if options.Fields["route"][0] != "short_name" {
		t.Errorf("expected fields[route][0]=short_name, got %s", options.Fields["route"][0])
	}
	if options.Fields["route"][1] != "long_name" {
		t.Errorf("expected fields[route][1]=long_name, got %s", options.Fields["route"][1])
	}

	// Check sort
	if len(options.Sort) != 2 {
		t.Errorf("expected 2 sort fields, got %d", len(options.Sort))
	}
	if options.Sort[0] != "name" {
		t.Errorf("expected sort[0]=name, got %s", options.Sort[0])
	}
	if options.Sort[1] != "-id" {
		t.Errorf("expected sort[1]=-id, got %s", options.Sort[1])
	}
}

func TestFilterFunction(t *testing.T) {
	// Create some test data
	numbers := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	
	// Filter for even numbers
	evenNumbers := Filter(numbers, func(n int) bool {
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
	greaterThanFive := Filter(numbers, func(n int) bool {
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

func TestHelperMethods(t *testing.T) {
	// Create test options
	query := url.Values{}
	query.Add("filter[id]", "1,2,3")
	query.Add("include", "stops,routes")
	query.Add("fields[route]", "short_name,long_name")
	query.Add("sort", "name,-id")

	options := NewOptions(query)

	// Test HasFilter
	if !options.HasFilter("id") {
		t.Errorf("expected HasFilter(id) to be true")
	}
	if options.HasFilter("nonexistent") {
		t.Errorf("expected HasFilter(nonexistent) to be false")
	}

	// Test GetFilter
	idFilter := options.GetFilter("id")
	if len(idFilter) != 1 || idFilter[0] != "1,2,3" {
		t.Errorf("expected GetFilter(id)=[1,2,3], got %v", idFilter)
	}
	nonExistentFilter := options.GetFilter("nonexistent")
	if len(nonExistentFilter) != 0 {
		t.Errorf("expected GetFilter(nonexistent) to be empty, got %v", nonExistentFilter)
	}

	// Test HasInclude
	if !options.HasInclude("stops") {
		t.Errorf("expected HasInclude(stops) to be true")
	}
	if !options.HasInclude("routes") {
		t.Errorf("expected HasInclude(routes) to be true")
	}
	if options.HasInclude("nonexistent") {
		t.Errorf("expected HasInclude(nonexistent) to be false")
	}

	// Test GetFields
	routeFields := options.GetFields("route")
	if len(routeFields) != 2 || routeFields[0] != "short_name" || routeFields[1] != "long_name" {
		t.Errorf("expected GetFields(route)=[short_name,long_name], got %v", routeFields)
	}
	nonExistentFields := options.GetFields("nonexistent")
	if len(nonExistentFields) != 0 {
		t.Errorf("expected GetFields(nonexistent) to be empty, got %v", nonExistentFields)
	}

	// Test ShouldIncludeField
	if !options.ShouldIncludeField("route", "short_name") {
		t.Errorf("expected ShouldIncludeField(route, short_name) to be true")
	}
	if options.ShouldIncludeField("route", "nonexistent") {
		t.Errorf("expected ShouldIncludeField(route, nonexistent) to be false")
	}
	if !options.ShouldIncludeField("nonexistent", "field") {
		t.Errorf("expected ShouldIncludeField(nonexistent, field) to be true")
	}

	// Test HasSort
	if !options.HasSort() {
		t.Errorf("expected HasSort() to be true")
	}

	// Test GetSort
	sort := options.GetSort()
	if len(sort) != 2 || sort[0] != "name" || sort[1] != "-id" {
		t.Errorf("expected GetSort()=[name,-id], got %v", sort)
	}
}