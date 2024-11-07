package main

import (
	"context"
	"log"
	"os"

	db "epistemic-me-backend/db"
	"epistemic-me-backend/server"
)

func main() {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		log.Fatalf("OPENAI_API_KEY environment variable not set")
	}

	kvStore, err := db.NewKeyValueStore("./epistemic_me.json")
	if err != nil {
		log.Printf("Warning: Failed to create KeyValueStore: %v", err)
		log.Println("Continuing with in-memory storage. Data will not be persisted.")
		kvStore, err = db.NewKeyValueStore("")
		if err != nil {
			log.Fatalf("Failed to create in-memory KeyValueStore: %v", err)
		}
	}
	log.Println("Successfully created KeyValueStore")

	srv, wg, _ := server.RunServer(kvStore, "8080") // Pass kvStore as a pointer
	wg.Wait()
	_ = srv.Shutdown(context.Background())
}
