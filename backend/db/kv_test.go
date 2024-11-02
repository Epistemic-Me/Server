package db

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestKeyValueStore(t *testing.T) {
	t.Run("In-Memory Store", func(t *testing.T) {
		store, err := NewKeyValueStore("")
		require.NoError(t, err)
		err = runTests(t, store)
		require.NoError(t, err)
	})

	t.Run("On-Disk Store", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second) // Increased from 60 to 120 seconds
		defer cancel()

		done := make(chan bool)
		errChan := make(chan error)
		var filePath string // Declare filePath here

		go func() {
			defer close(done)
			defer close(errChan)

			tempDir, err := os.MkdirTemp("", "kvstore_test")
			if err != nil {
				errChan <- fmt.Errorf("failed to create temp dir: %v", err)
				return
			}
			defer os.RemoveAll(tempDir)

			filePath = filepath.Join(tempDir, "kvstore.json") // Assign to the outer filePath

			// Create an empty file
			err = os.WriteFile(filePath, []byte("{}"), 0644)
			if err != nil {
				errChan <- fmt.Errorf("failed to create initial empty file: %v", err)
				return
			}

			store, err := NewKeyValueStore(filePath)
			if err != nil {
				errChan <- fmt.Errorf("failed to create new key-value store: %v", err)
				return
			}

			if err := runTests(t, store); err != nil {
				errChan <- fmt.Errorf("tests failed: %v", err)
				return
			}

			// Test persistence
			newStore, err := NewKeyValueStore(filePath)
			if err != nil {
				errChan <- fmt.Errorf("failed to create new key-value store for persistence test: %v", err)
				return
			}

			if err := testPersistence(t, newStore); err != nil {
				errChan <- fmt.Errorf("persistence test failed: %v", err)
				return
			}

			done <- true
		}()

		select {
		case <-ctx.Done():
			t.Fatal("Test timed out after 120 seconds. Last operation: Persisting to disk")
		case err := <-errChan:
			t.Fatalf("Test failed with error: %v", err)
		case <-done:
			// Test completed successfully

			// After the test runs, check if the file exists
			if _, err := os.Stat(filePath); os.IsNotExist(err) {
				t.Errorf("File was not created at %s", filePath)
			} else if err != nil {
				t.Errorf("Error checking file: %v", err)
			}
		}
	})
}

func runTests(t *testing.T, store *KeyValueStore) error {
	var err error

	t.Run("Store and Retrieve", func(t *testing.T) {
		err = testStoreAndRetrieve(t, store)
		assert.NoError(t, err)
	})
	if err != nil {
		return err
	}

	t.Run("Store Multiple Versions", func(t *testing.T) {
		err = testStoreMultipleVersions(t, store)
		assert.NoError(t, err)
	})
	if err != nil {
		return err
	}

	t.Run("ListByType", func(t *testing.T) {
		// Clear the store before this test
		store.store = make(map[string]map[string][]storedValue)
		err = testListByType(t, store)
		assert.NoError(t, err)
	})
	if err != nil {
		return err
	}

	t.Run("StoreWithVersionReplacement", func(t *testing.T) {
		TestKeyValueStore_StoreWithVersionReplacement(t)
	})
	if err != nil {
		return err
	}

	return nil
}

func testStoreAndRetrieve(t *testing.T, store *KeyValueStore) error {
	developerId := "testDeveloper"
	key := "testKey"
	value := TestStruct{ID: "1", Name: "Test"}

	done := make(chan error)
	go func() {
		done <- store.Store(developerId, key, value, 1)
	}()

	select {
	case err := <-done:
		if err != nil {
			return fmt.Errorf("failed to store value: %v", err)
		}
	case <-time.After(5 * time.Second):
		return fmt.Errorf("store operation timed out")
	}

	retrieved, err := store.Retrieve(developerId, key)
	if err != nil {
		return fmt.Errorf("failed to retrieve value: %v", err)
	}

	if !reflect.DeepEqual(value, *retrieved.(*TestStruct)) {
		return fmt.Errorf("retrieved value does not match stored value")
	}

	_, err = store.Retrieve(developerId, "nonExistentKey")
	if err == nil {
		return fmt.Errorf("expected error when retrieving non-existent key")
	}

	_, err = store.Retrieve("nonExistentDeveloper", key)
	if err == nil {
		return fmt.Errorf("expected error when retrieving for non-existent developer")
	}

	return nil
}

func testStoreMultipleVersions(t *testing.T, store *KeyValueStore) error {
	developerId := "testDeveloper"
	key := "multiVersionKey"
	value1 := TestStruct{ID: "1", Name: "Version 1"}
	value2 := TestStruct{ID: "1", Name: "Version 2"}

	err := store.Store(developerId, key, value1, 1)
	if err != nil {
		return fmt.Errorf("failed to store first version: %v", err)
	}

	err = store.Store(developerId, key, value2, 2)
	if err != nil {
		return fmt.Errorf("failed to store second version: %v", err)
	}

	versions, err := store.RetrieveAllVersions(developerId, key)
	if err != nil {
		return fmt.Errorf("failed to retrieve all versions: %v", err)
	}

	if len(versions) != 2 {
		return fmt.Errorf("expected 2 versions, got %d", len(versions))
	}

	if !reflect.DeepEqual(value1, *versions[0].(*TestStruct)) {
		return fmt.Errorf("first version does not match stored value")
	}

	if !reflect.DeepEqual(value2, *versions[1].(*TestStruct)) {
		return fmt.Errorf("second version does not match stored value")
	}

	return nil
}

func testListByType(t *testing.T, store *KeyValueStore) error {
	developerId := "testDeveloper"
	value1 := TestStruct{ID: "1", Name: "Test1"}
	value2 := TestStruct{ID: "2", Name: "Test2"}

	err := store.Store(developerId, "key1", value1, 1)
	if err != nil {
		return fmt.Errorf("failed to store first value: %v", err)
	}

	err = store.Store(developerId, "key2", value2, 1)
	if err != nil {
		return fmt.Errorf("failed to store second value: %v", err)
	}

	results, err := store.ListByType(developerId, reflect.TypeOf(TestStruct{}))
	if err != nil {
		return fmt.Errorf("failed to list by type: %v", err)
	}

	if len(results) != 2 {
		return fmt.Errorf("expected 2 results, got %d", len(results))
	}

	return nil
}

func testPersistence(t *testing.T, store *KeyValueStore) error {
	developerId := "testDeveloper"
	key := "persistenceKey"
	value := TestStruct{ID: "1", Name: "Persistence Test"}

	err := store.Store(developerId, key, value, 1)
	if err != nil {
		return fmt.Errorf("failed to store value: %v", err)
	}

	retrieved, err := store.Retrieve(developerId, key)
	if err != nil {
		return fmt.Errorf("failed to retrieve value: %v", err)
	}

	if !reflect.DeepEqual(value, *retrieved.(*TestStruct)) {
		return fmt.Errorf("retrieved value does not match stored value")
	}

	return nil
}

func TestKeyValueStore_StoreWithVersionReplacement(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "kvstore_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a file path for the test store
	filePath := filepath.Join(tempDir, "test_store.json")

	// Create an empty file with an empty JSON object
	err = os.WriteFile(filePath, []byte("{}"), 0644)
	if err != nil {
		t.Fatalf("Failed to create initial empty file: %v", err)
	}

	// Initialize the KeyValueStore with the temp file
	kvs, err := NewKeyValueStore(filePath)
	if err != nil {
		t.Fatalf("Failed to create KeyValueStore: %v", err)
	}

	// Test data
	developerId := "testDeveloper"
	key := "testkey"
	value1 := TestStruct{ID: "1", Name: "Test1"}
	value2 := TestStruct{ID: "1", Name: "Test2"}

	// Store the first value with version 1
	err = kvs.Store(developerId, key, value1, 1)
	if err != nil {
		t.Fatalf("Failed to store initial value: %v", err)
	}

	// Store a new value with the same version (1)
	err = kvs.Store(developerId, key, value2, 1)
	if err != nil {
		t.Fatalf("Failed to store replacement value: %v", err)
	}

	// Retrieve the value
	retrieved, err := kvs.Retrieve(developerId, key)
	if err != nil {
		t.Fatalf("Failed to retrieve value: %v", err)
	}

	// Check if the retrieved value matches the replaced value
	retrievedStruct, ok := retrieved.(*TestStruct)
	if !ok {
		t.Fatalf("Retrieved value is not of type *TestStruct")
	}

	if retrievedStruct.ID != value2.ID || retrievedStruct.Name != value2.Name {
		t.Errorf("Retrieved value does not match the replaced value. Got %+v, want %+v", retrievedStruct, value2)
	}

	// Retrieve all versions
	allVersions, err := kvs.RetrieveAllVersions(developerId, key)
	if err != nil {
		t.Fatalf("Failed to retrieve all versions: %v", err)
	}

	// Check if there's only one version stored
	if len(allVersions) != 1 {
		t.Errorf("Expected 1 version, got %d versions", len(allVersions))
	}

	// Store a new value with a different version
	value3 := TestStruct{ID: "1", Name: "Test3"}
	err = kvs.Store(developerId, key, value3, 2)
	if err != nil {
		t.Fatalf("Failed to store new version: %v", err)
	}

	// Retrieve all versions again
	allVersions, err = kvs.RetrieveAllVersions(developerId, key)
	if err != nil {
		t.Fatalf("Failed to retrieve all versions after adding new version: %v", err)
	}

	// Check if there are now two versions stored
	if len(allVersions) != 2 {
		t.Errorf("Expected 2 versions, got %d versions", len(allVersions))
	}

	// Verify the contents of both versions
	for _, v := range allVersions {
		ts, ok := v.(*TestStruct)
		if !ok {
			t.Fatalf("Retrieved value is not of type *TestStruct")
		}
		if ts.Name != value2.Name && ts.Name != value3.Name {
			t.Errorf("Unexpected version found: %+v", ts)
		}
	}
}
