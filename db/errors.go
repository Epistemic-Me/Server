package db

import "errors"

var (
	// ErrNotFound is returned when a key is not found in the store
	ErrNotFound = errors.New("not found")
)
