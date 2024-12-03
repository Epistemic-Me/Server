package main

import (
	"fmt"
	"log"
	"os"

	"epistemic-me-core/db"
	fixture_models "epistemic-me-core/db/fixtures"
)

// This script is used to import the belief system from the fixture
// into the persistent KeyValueStore.
// Its used to inspect the persistent KeyValueStore data in the local development environment.
func main() {
	kvStorePath := os.Getenv("KV_STORE_PATH")
	if kvStorePath == "" {
		log.Fatal("KV_STORE_PATH environment variable is not set")
	}

	// Create a new persistent KeyValueStore
	kvStore, err := db.NewKeyValueStore(kvStorePath)
	if err != nil {
		log.Fatalf("Error creating KeyValueStore: %v", err)
	}

	import_err := fixture_models.ImportFixtures(kvStore)
	if import_err != nil {
		log.Fatalf("Failed to import fixtures: %v", err)
	}

	fmt.Println("Fixture belief system has been successfully imported into the persistent KeyValueStore!")
}
