package db

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"reflect"
	"sync"
)

// KeyValueStore holds the in-memory store and a mutex for thread-safe access.
type KeyValueStore struct {
	store    map[string]map[string][]storedValue // user -> key -> []storedValue (slice to hold different versions)
	mu       sync.RWMutex
	filePath string     // New field for persistence
	diskMu   sync.Mutex // New mutex for disk operations
}

// storedValue holds the JSON string, the type of the original object, and the version.
type storedValue struct {
	JsonData string
	Type     reflect.Type
	Version  int
}

// serializableStoredValue is a serializable version of storedValue
type serializableStoredValue struct {
	JsonData string
	Type     string
	Version  int
}

// NewKeyValueStore initializes and returns a new KeyValueStore.
// If filePath is provided, it attempts to load the store from the file.
func NewKeyValueStore(filePath string) (*KeyValueStore, error) {
	log.Printf("Creating new KeyValueStore with filePath: %s", filePath)
	kvs := &KeyValueStore{
		store:    make(map[string]map[string][]storedValue),
		filePath: filePath,
	}

	if filePath != "" {
		// Check if the file exists
		_, err := os.Stat(filePath)
		if os.IsNotExist(err) {
			// File doesn't exist, create it
			log.Printf("File %s doesn't exist. Creating it.", filePath)
			err = kvs.SaveToDisk()
			if err != nil {
				return nil, fmt.Errorf("failed to create initial file: %v", err)
			}
		} else if err != nil {
			// Some other error occurred
			return nil, fmt.Errorf("failed to check file status: %v", err)
		} else {
			// File exists, load it
			err = kvs.LoadFromDisk()
			if err != nil {
				return nil, fmt.Errorf("failed to load from disk: %v", err)
			}
		}
	}

	return kvs, nil
}

// SaveToDisk persists the current state of the store to the file specified by filePath.
func (kvs *KeyValueStore) SaveToDisk() error {
	if kvs.filePath == "" {
		return nil // No persistence requested
	}

	kvs.diskMu.Lock()
	defer kvs.diskMu.Unlock()

	kvs.mu.RLock()
	defer kvs.mu.RUnlock()

	// Create a snapshot of the store while holding the read lock
	snapshot := make(map[string]map[string][]storedValue)
	for user, userStore := range kvs.store {
		snapshot[user] = make(map[string][]storedValue)
		for key, values := range userStore {
			snapshot[user][key] = make([]storedValue, len(values))
			copy(snapshot[user][key], values)
		}
	}

	// Work with the snapshot to create the serializable store
	serializableStore := make(map[string]map[string][]serializableStoredValue)
	for user, userStore := range snapshot {
		serializableStore[user] = make(map[string][]serializableStoredValue)
		for key, values := range userStore {
			serializableValues := make([]serializableStoredValue, len(values))
			for i, v := range values {
				serializableValues[i] = serializableStoredValue{
					JsonData: v.JsonData,
					Type:     v.Type.String(),
					Version:  v.Version,
				}
			}
			serializableStore[user][key] = serializableValues
		}
	}

	// Marshal the serializable store to JSON
	jsonData, err := json.Marshal(serializableStore)
	if err != nil {
		return fmt.Errorf("failed to marshal store: %w", err)
	}

	// Write to the file
	err = os.WriteFile(kvs.filePath, jsonData, 0644)
	if err != nil {
		return fmt.Errorf("failed to write to file: %w", err)
	}

	return nil
}

// LoadFromDisk loads the store state from the file specified by filePath.
func (kvs *KeyValueStore) LoadFromDisk() error {
	data, err := os.ReadFile(kvs.filePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	var serializableStore map[string]map[string][]serializableStoredValue
	err = json.Unmarshal(data, &serializableStore)
	if err != nil {
		return fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	kvs.mu.Lock()
	defer kvs.mu.Unlock()

	kvs.store = make(map[string]map[string][]storedValue)
	for user, userStore := range serializableStore {
		kvs.store[user] = make(map[string][]storedValue)
		for key, values := range userStore {
			storedValues := make([]storedValue, len(values))
			for i, v := range values {
				t, err := getTypeFromName(v.Type)
				if err != nil {
					return fmt.Errorf("failed to get type from name: %w", err)
				}
				storedValues[i] = storedValue{
					JsonData: v.JsonData,
					Type:     t,
					Version:  v.Version,
				}
			}
			kvs.store[user][key] = storedValues
		}
	}

	return nil
}

// Store checks if all fields in the given struct have JSON tags and stores the struct as JSON.
// It stores the value with the specified version number.
func (kvs *KeyValueStore) Store(user, key string, value interface{}, version int) error {
	log.Printf("Storing value of type %T for user %s with key %s and version %d", value, user, key, version)

	v := reflect.ValueOf(value)
	if v.Kind() != reflect.Struct {
		return fmt.Errorf("value must be a struct")
	}

	// Check for JSON annotations
	t := v.Type()
	for i := 0; i < v.NumField(); i++ {
		field := t.Field(i)
		if _, ok := field.Tag.Lookup("json"); !ok {
			return fmt.Errorf("field %s does not have a json tag", field.Name)
		}
	}

	// Convert to JSON
	jsonData, err := json.Marshal(value)
	if err != nil {
		return err
	}
	kvs.mu.Lock()
	defer kvs.mu.Unlock()

	// Perform in-memory store operation
	if _, exists := kvs.store[user]; !exists {
		kvs.store[user] = make(map[string][]storedValue)
	}

	// Insert the value at the correct version position
	existingValues := kvs.store[user][key]

	// Check if the version already exists
	for i, storedVal := range existingValues {
		if storedVal.Version == version {
			// Replace the existing version
			kvs.store[user][key][i] = storedValue{
				JsonData: string(jsonData),
				Type:     reflect.TypeOf(value),
				Version:  version,
			}
			return nil
		}
	}

	kvs.store[user][key] = append(existingValues, storedValue{
		JsonData: string(jsonData),
		Type:     reflect.TypeOf(value),
		Version:  version,
	})

	// Sort by version (in case versions are added out of order)
	kvs.sortByVersion(user, key)

	// Create a copy of the data to be persisted
	var dataToPersist map[string]map[string][]storedValue
	if kvs.filePath != "" {
		dataToPersist = make(map[string]map[string][]storedValue)
		for u, userStore := range kvs.store {
			dataToPersist[u] = make(map[string][]storedValue)
			for k, values := range userStore {
				dataToPersist[u][k] = make([]storedValue, len(values))
				copy(dataToPersist[u][k], values)
			}
		}
	}

	// Perform disk persistence outside of the lock
	if kvs.filePath != "" {
		err := kvs.saveToDiskWithData(dataToPersist)
		if err != nil {
			return err
		}
	}

	return nil
}

func (kvs *KeyValueStore) saveToDiskWithData(data map[string]map[string][]storedValue) error {
	kvs.diskMu.Lock()
	defer kvs.diskMu.Unlock()

	// Convert to serializable format
	serializableStore := make(map[string]map[string][]serializableStoredValue)
	for user, userStore := range data {
		serializableStore[user] = make(map[string][]serializableStoredValue)
		for key, values := range userStore {
			serializableValues := make([]serializableStoredValue, len(values))
			for i, v := range values {
				serializableValues[i] = serializableStoredValue{
					JsonData: v.JsonData,
					Type:     v.Type.String(),
					Version:  v.Version,
				}
			}
			serializableStore[user][key] = serializableValues
		}
	}

	// Marshal and write to disk
	jsonData, err := json.Marshal(serializableStore)
	if err != nil {
		return fmt.Errorf("failed to marshal store: %w", err)
	}

	err = os.WriteFile(kvs.filePath, jsonData, 0644)
	if err != nil {
		return fmt.Errorf("failed to write to file: %w", err)
	}

	return nil
}

// sortByVersion sorts the versions of a key for a user in ascending order.
func (kvs *KeyValueStore) sortByVersion(user, key string) {
	values := kvs.store[user][key]
	for i := 1; i < len(values); i++ {
		for j := i; j > 0 && values[j-1].Version > values[j].Version; j-- {
			values[j-1], values[j] = values[j], values[j-1]
		}
	}
}

// Retrieve gets the latest stored value under the given user and key, and deserializes it into the original object type.
func (kvs *KeyValueStore) Retrieve(userID string, key string) (interface{}, error) {
	log.Printf("Retrieving key %s for user %s", key, userID)
	kvs.mu.RLock()
	defer kvs.mu.RUnlock()

	userStore, userExists := kvs.store[userID]
	if !userExists {
		return nil, fmt.Errorf("user not found")
	}

	storedValues, keyExists := userStore[key]
	if !keyExists || len(storedValues) == 0 {
		return nil, fmt.Errorf("key not found")
	}

	// Get the latest version
	latestValue := storedValues[len(storedValues)-1]

	// Create a new instance of the original type
	v := reflect.New(latestValue.Type).Interface()

	// Unmarshal the JSON data into the new instance
	err := json.Unmarshal([]byte(latestValue.JsonData), v)
	if err != nil {
		return nil, err
	}
	log.Printf("Retrieved value: %+v", v)

	return v, nil
}

// RetrieveAllVersions retrieves all versions of the stored value under the given user and key.
func (kvs *KeyValueStore) RetrieveAllVersions(userID string, key string) ([]interface{}, error) {
	log.Printf("Retrieving all versions for key %s for user %s", key, userID)
	kvs.mu.RLock()
	defer kvs.mu.RUnlock()

	userStore, userExists := kvs.store[userID]
	if !userExists {
		return nil, fmt.Errorf("user not found")
	}

	storedValues, keyExists := userStore[key]
	if !keyExists || len(storedValues) == 0 {
		return nil, fmt.Errorf("key not found")
	}

	// Retrieve all versions
	var result []interface{}
	for _, storedValue := range storedValues {
		// Create a new instance of the original type
		v := reflect.New(storedValue.Type).Interface()

		// Unmarshal the JSON data into the new instance
		err := json.Unmarshal([]byte(storedValue.JsonData), v)
		if err != nil {
			return nil, err
		}
		result = append(result, v)
	}

	return result, nil
}

// ListByType lists all objects of a given type associated with a user.
// It ensures that only the latest versions are returned.
func (kvs *KeyValueStore) ListByType(user string, objType reflect.Type) ([]interface{}, error) {
	kvs.mu.RLock()
	defer kvs.mu.RUnlock()

	userStore, userExists := kvs.store[user]
	if !userExists {
		return nil, fmt.Errorf("user not found")
	}

	var result []interface{}
	for _, storedValues := range userStore {
		if len(storedValues) > 0 {
			latestValue := storedValues[len(storedValues)-1]
			if latestValue.Type == objType {
				v := reflect.New(latestValue.Type).Interface()
				err := json.Unmarshal([]byte(latestValue.JsonData), v)
				if err != nil {
					return nil, err
				}
				result = append(result, v)
			}
		}
	}

	return result, nil
}

// ClearStore removes all data from the KeyValueStore
func (kvs *KeyValueStore) ClearStore() {
	kvs.store = make(map[string]map[string][]storedValue)
	kvs.SaveToDisk() // If you want to clear the persistent storage as well
}
