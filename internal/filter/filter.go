package filter

import (
	"net/url"
	"strings"
)

// Options represents filter, sort, include, and fields options for an API request
type Options struct {
	Filters  map[string][]string
	Includes []string
	Fields   map[string][]string
	Sort     []string
}

// NewOptions parses query parameters and creates filter options
func NewOptions(query url.Values) *Options {
	options := &Options{
		Filters:  make(map[string][]string),
		Includes: []string{},
		Fields:   make(map[string][]string),
		Sort:     []string{},
	}

	// Parse filters
	for key, values := range query {
		if strings.HasPrefix(key, "filter[") && strings.HasSuffix(key, "]") {
			filterName := key[7 : len(key)-1]
			options.Filters[filterName] = values
		}
	}

	// Parse includes
	if includeParam, ok := query["include"]; ok && len(includeParam) > 0 {
		includes := strings.Split(includeParam[0], ",")
		for _, include := range includes {
			if include = strings.TrimSpace(include); include != "" {
				options.Includes = append(options.Includes, include)
			}
		}
	}

	// Parse fields
	for key, values := range query {
		if strings.HasPrefix(key, "fields[") && strings.HasSuffix(key, "]") {
			resourceType := key[7 : len(key)-1]
			if len(values) > 0 {
				fields := strings.Split(values[0], ",")
				for i, field := range fields {
					fields[i] = strings.TrimSpace(field)
				}
				options.Fields[resourceType] = fields
			}
		}
	}

	// Parse sorting
	if sortParam, ok := query["sort"]; ok && len(sortParam) > 0 {
		sortFields := strings.Split(sortParam[0], ",")
		for _, field := range sortFields {
			if field = strings.TrimSpace(field); field != "" {
				options.Sort = append(options.Sort, field)
			}
		}
	}

	return options
}

// HasFilter checks if a specific filter exists
func (o *Options) HasFilter(name string) bool {
	_, exists := o.Filters[name]
	return exists
}

// GetFilter returns the value(s) for a specific filter
func (o *Options) GetFilter(name string) []string {
	return o.Filters[name]
}

// HasInclude checks if a specific include is requested
func (o *Options) HasInclude(name string) bool {
	for _, include := range o.Includes {
		if include == name {
			return true
		}
	}
	return false
}

// GetFields returns the fields to include for a resource type
func (o *Options) GetFields(resourceType string) []string {
	return o.Fields[resourceType]
}

// ShouldIncludeField checks if a field should be included
func (o *Options) ShouldIncludeField(resourceType, field string) bool {
	fields, ok := o.Fields[resourceType]
	if !ok {
		// If no fields specified, include all
		return true
	}

	for _, f := range fields {
		if f == field {
			return true
		}
	}
	return false
}

// HasSort checks if sorting is requested
func (o *Options) HasSort() bool {
	return len(o.Sort) > 0
}

// GetSort returns the sort fields
func (o *Options) GetSort() []string {
	return o.Sort
}

// FilterFunc is a generic filter function type
type FilterFunc[T any] func(item T) bool

// Filter applies a filter function to a slice of items
func Filter[T any](items []T, fn FilterFunc[T]) []T {
	filtered := make([]T, 0, len(items))
	for _, item := range items {
		if fn(item) {
			filtered = append(filtered, item)
		}
	}
	return filtered
}
