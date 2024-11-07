package svc

import (
	ai "epistemic-me-core/ai"
	db "epistemic-me-core/db"
	"epistemic-me-core/svc/models"
	"time"

	"github.com/google/uuid"
)

type UserService struct {
	kvStore *db.KeyValueStore
	ai      *ai.AIHelper
}

func NewUserService(kvStore *db.KeyValueStore, ai *ai.AIHelper) *UserService {
	return &UserService{
		kvStore: kvStore,
		ai:      ai,
	}
}

func (s *UserService) CreateUser(input *models.CreateUserInput) (*models.CreateUserOutput, error) {
	user := models.User{
		ID:          "user_" + uuid.New().String(),
		DeveloperID: input.DeveloperID,
		Name:        input.Name,
		Email:       input.Email,
		CreatedAt:   time.Now().UnixMilli(),
		UpdatedAt:   time.Now().UnixMilli(),
	}

	err := s.kvStore.Store(input.DeveloperID, "user_"+user.ID, user, 1)
	if err != nil {
		return nil, err
	}

	return &models.CreateUserOutput{
		User: user,
	}, nil
}

// Add other methods as needed (e.g., GetUser, UpdateUser, DeleteUser)
