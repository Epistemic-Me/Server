package integration

import (
	"net/http"
	"time"
)

var customHttpClient = &http.Client{
	Timeout: 10 * time.Second,
}

// Add any common utility functions or variables here
