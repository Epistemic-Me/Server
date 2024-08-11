package db

import (
	"encoding/json"
	"fmt"
	"log"
	"reflect"
	"sync"
)

// KeyValueStore holds the in-memory store and a mutex for thread-safe access.
type KeyValueStore struct {
	store map[string]map[string][]storedValue // user -> key -> []storedValue (slice to hold different versions)
	mu    sync.Mutex
}

// storedValue holds the JSON string, the type of the original object, and the version.
type storedValue struct {
	JsonData string
	Type     reflect.Type
	Version  int
}

// NewKeyValueStore initializes and returns a new KeyValueStore.
func NewKeyValueStore() *KeyValueStore {
	return &KeyValueStore{
		store: make(map[string]map[string][]storedValue),
	}
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

	// Store in memory with the given version
	kvs.mu.Lock()
	defer kvs.mu.Unlock()

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
				Type:     t,
				Version:  version,
			}
			return nil
		}
	}

	// If the version does not exist, append it to the slice
	kvs.store[user][key] = append(existingValues, storedValue{
		JsonData: string(jsonData),
		Type:     t,
		Version:  version,
	})

	// Sort by version (in case versions are added out of order)
	kvs.sortByVersion(user, key)

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
	kvs.mu.Lock()
	defer kvs.mu.Unlock()

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

	return v, nil
}

// RetrieveAllVersions retrieves all versions of the stored value under the given user and key.
func (kvs *KeyValueStore) RetrieveAllVersions(userID string, key string) ([]interface{}, error) {
	log.Printf("Retrieving all versions for key %s for user %s", key, userID)
	kvs.mu.Lock()
	defer kvs.mu.Unlock()

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
	kvs.mu.Lock()
	defer kvs.mu.Unlock()

	userStore, userExists := kvs.store[user]
	if !userExists {
		return nil, fmt.Errorf("user not found")
	}

	var result []interface{}
	for _, storedValues := range userStore {
		if len(storedValues) > 0 && storedValues[len(storedValues)-1].Type == objType {
			// Get the latest version
			latestValue := storedValues[len(storedValues)-1]
			v := reflect.New(latestValue.Type).Interface()
			err := json.Unmarshal([]byte(latestValue.JsonData), v)
			if err != nil {
				return nil, err
			}
			result = append(result, v)
		}
	}

	return result, nil
}
