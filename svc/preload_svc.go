package svc

import (
	"context"
	"encoding/json"
	"epistemic-me-core/svc/models"
	"io/ioutil"
	"log"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
)

type PreloadService struct {
	sms               *SelfModelService
	perspectiveEpiSvc *PerspectiveTakingEpistemology
	philosophiesPath  string
}

func NewPreloadSvc(sms *SelfModelService, epiSvc *PerspectiveTakingEpistemology, philosophiesPath string) *PreloadService {
	return &PreloadService{
		sms:               sms,
		perspectiveEpiSvc: epiSvc,
		philosophiesPath:  philosophiesPath,
	}
}

type Philosophy struct {
	SelfId       string `json:"self_id"`
	Strategy     string `json:"strategy"`
	File         string `json:"file"`
	MetaStrategy string `json:"meta_strategy_used"`
}

func (ps *PreloadService) loadPhilosophies() ([]Philosophy, error) {
	var philosophies []Philosophy

	// Walk through each author's directory
	authorDirs, err := ioutil.ReadDir(ps.philosophiesPath)
	if err != nil {
		return nil, err
	}

	for _, authorDir := range authorDirs {
		if !authorDir.IsDir() {
			continue
		}

		// Read each JSON file in the author's directory
		authorPath := filepath.Join(ps.philosophiesPath, authorDir.Name())
		files, err := ioutil.ReadDir(authorPath)
		if err != nil {
			log.Printf("Error reading author directory %s: %v", authorPath, err)
			continue
		}

		for _, file := range files {
			if !file.IsDir() && filepath.Ext(file.Name()) == ".json" {
				filePath := filepath.Join(authorPath, file.Name())
				data, err := ioutil.ReadFile(filePath)
				if err != nil {
					log.Printf("Error reading file %s: %v", filePath, err)
					continue
				}

				var philosophy Philosophy
				if err := json.Unmarshal(data, &philosophy); err != nil {
					log.Printf("Error parsing JSON from file %s: %v", filePath, err)
					continue
				}

				log.Printf("Loaded philosophy from %s: %+v", filePath, philosophy)
				philosophies = append(philosophies, philosophy)
			}
		}
	}

	return philosophies, nil
}

func (ps *PreloadService) RunPreload(ctx context.Context) error {
	// Load philosophies from filesystem
	philosophies, err := ps.loadPhilosophies()
	if err != nil {
		return err
	}

	resources := make([]models.Resource, 0)

	// create a resource for each philosophy
	for _, philosophy := range philosophies {
		resources = append(resources, models.Resource{
			ID:       uuid.New().String(),
			Type:     models.ResourceTypePhilosophy,
			Content:  philosophy.Strategy,
			SourceID: uuid.New().String(), // Generate a source ID for tracking
			Metadata: map[string]string{
				"selfModelId":  philosophy.SelfId,
				"file":         philosophy.File,
				"metaStrategy": philosophy.MetaStrategy,
				"authorPath":   filepath.Base(filepath.Dir(philosophy.File)), // Add the author's name
			},
		})
	}

	for _, resource := range resources {
		selfModelID := resource.Metadata["selfModelId"]

		// check if the self model already exists
		_, err := ps.sms.GetSelfModel(ctx, &models.GetSelfModelInput{
			SelfModelID: selfModelID,
		})

		if err != nil {
			// If it's not the "invalid self model data" error, just return
			// todo: fix whether this should be "developer" or "self model"
			if !strings.Contains(err.Error(), "developer not found") {
				return err
			}

			// Create the Self Model (which will initialize its own belief system)
			if _, err := ps.sms.CreateSelfModel(ctx, &models.CreateSelfModelInput{
				ID: selfModelID,
			}); err != nil {
				return err
			}
		}

		event := &models.PerspectiveTakingEpistemicEvent{
			Resource: resource,
		}

		// Process the resource for the self model
		_, err = ps.perspectiveEpiSvc.Process(event, false, selfModelID)
		if err != nil {
			return err
		}
	}

	return nil
}
