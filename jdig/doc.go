// Package jdig provides a simple way to deal with JSON.
//
// This package assume only basic unmarshaled JSON types (map[string]any, []any, string, float64, bool, nil) are used.
// Using other types may cause unexpected behavior.
//
// # Query keys
//
// For keys parameters, only string and int are allowed.
// All query function will return zero value if the value at keys path is not the expected type, or the path does not exist.
//
// # Mutation
//
// All mutating functions will use in-place mutation, and return the mutated value.
// Make copy if you don't want to mutate the original value.
package jdig
