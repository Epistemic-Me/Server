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
	store map[string]map[string]storedValue // user -> key -> storedValue
	mu    sync.Mutex
}

// storedValue holds the JSON string and the type of the original object.
type storedValue struct {
	JsonData string
	Type     reflect.Type
}

// NewKeyValueStore initializes and returns a new KeyValueStore.
func NewKeyValueStore() *KeyValueStore {
	return &KeyValueStore{
		store: make(map[string]map[string]storedValue),
	}
}

// Store checks if all fields in the given struct have JSON tags and stores the struct as JSON.
func (kvs *KeyValueStore) Store(user, key string, value interface{}) error {
	log.Printf("Storing value of type %T for user %s with key %s", value, user, key)

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

	// Store in memory
	kvs.mu.Lock()
	defer kvs.mu.Unlock()

	if _, exists := kvs.store[user]; !exists {
		kvs.store[user] = make(map[string]storedValue)
	}
	kvs.store[user][key] = storedValue{JsonData: string(jsonData), Type: t}

	return nil
}

// Retrieve gets the stored value under the given user and key, and deserializes it into the original object type.
func (kvs *KeyValueStore) Retrieve(userID string, key string) (interface{}, error) {
	log.Printf("Retrieving key %s for user %s", key, userID)
	kvs.mu.Lock()
	defer kvs.mu.Unlock()

	userStore, userExists := kvs.store[userID]
	if !userExists {
		return nil, fmt.Errorf("user not found")
	}

	storedValue, keyExists := userStore[key]
	if !keyExists {
		return nil, fmt.Errorf("key not found")
	}

	// Create a new instance of the original type
	v := reflect.New(storedValue.Type).Interface()

	// Unmarshal the JSON data into the new instance
	err := json.Unmarshal([]byte(storedValue.JsonData), v)
	if err != nil {
		return nil, err
	}

	return v, nil
}

// ListByType lists all objects of a given type associated with a user.
func (kvs *KeyValueStore) ListByType(user string, objType reflect.Type) ([]interface{}, error) {
	kvs.mu.Lock()
	defer kvs.mu.Unlock()

	userStore, userExists := kvs.store[user]
	if !userExists {
		return nil, fmt.Errorf("user not found")
	}

	var result []interface{}
	for _, storedValue := range userStore {
		if storedValue.Type == objType {
			v := reflect.New(storedValue.Type).Interface()
			err := json.Unmarshal([]byte(storedValue.JsonData), v)
			if err != nil {
				return nil, err
			}
			result = append(result, v)
		}
	}

	return result, nil
}
