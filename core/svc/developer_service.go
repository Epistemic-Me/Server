package svc

import (
	ai "epistemic-me-core/ai"
	db "epistemic-me-core/db"
	"epistemic-me-core/svc/models"
	"errors"
	"fmt"
	"reflect"
	"time"

	"github.com/google/uuid"
)

type DeveloperService struct {
	kvStore *db.KeyValueStore
	ai      *ai.AIHelper
}

func NewDeveloperService(kvStore *db.KeyValueStore, ai *ai.AIHelper) *DeveloperService {
	return &DeveloperService{
		kvStore: kvStore,
		ai:      ai,
	}
}

func (s *DeveloperService) CreateDeveloper(input *models.CreateDeveloperInput) (*models.CreateDeveloperOutput, error) {
	developer := models.Developer{
		ID:        "dev_" + uuid.New().String(), // Use a UUID instead of the name
		Name:      input.Name,
		Email:     input.Email,
		APIKeys:   []string{uuid.New().String()}, // Generate a random API key
		CreatedAt: time.Now().UnixMilli(),
		UpdatedAt: time.Now().UnixMilli(),
	}

	err := s.kvStore.Store(developer.ID, "developer", developer, 1)
	if err != nil {
		return nil, err
	}

	return &models.CreateDeveloperOutput{
		Developer: developer,
	}, nil
}

func (s *DeveloperService) GetDeveloper(input *models.GetDeveloperInput) (*models.Developer, error) {
	if input.ID == "" {
		return nil, fmt.Errorf("developer ID is required")
	}

	// Get the developer from the KV store
	result, err := s.kvStore.Retrieve(input.ID, "developer")
	if err != nil {
		return nil, fmt.Errorf("failed to get developer: %w", err)
	}
	if result == nil {
		return nil, fmt.Errorf("developer not found")
	}

	// Type assert the result to *models.Developer
	developer, ok := result.(*models.Developer)
	if !ok {
		return nil, fmt.Errorf("invalid developer data type")
	}

	return developer, nil
}

// Update the GetDeveloperByAPIKey method

func (s *DeveloperService) GetDeveloperByAPIKey(apiKey string) (*models.Developer, error) {
	developers, err := s.kvStore.ListAllByType(reflect.TypeOf(models.Developer{}))
	if err != nil {
		return nil, err
	}

	for _, dev := range developers {
		developer, ok := dev.(*models.Developer)
		if !ok {
			continue
		}
		for _, key := range developer.APIKeys {
			if key == apiKey {
				return developer, nil
			}
		}
	}

	return nil, errors.New("developer not found for the given API key")
}

// Add other methods as needed (e.g., GetDeveloper, UpdateDeveloper, DeleteDeveloper)
