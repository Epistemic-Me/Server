package svc

import (
	"context"
	"epistemic-me-core/svc/models"
)

type PreloadService struct {
	sms               *SelfModelService
	perspectiveEpiSvc *PerspectiveTakingEpistemology
}

func NewPreloadSvc(sms *SelfModelService, epiSvc *PerspectiveTakingEpistemology) *PreloadService {
	return &PreloadService{
		sms:               sms,
		perspectiveEpiSvc: epiSvc,
	}
}

type Philosophy struct {
	Self_id  string `json:"self_id"`
	Strategy string `json:"strategy"`
}

func (ps *PreloadService) RunPreload(ctx context.Context) error {

	// todo: @deen preseed philosophies
	// handle in part 3 of perspective taker
	philosophies := make([]Philosophy, 0)

	resources := make([]models.Resource, 0)

	// create a resource for each philosophy
	for _, philosophy := range philosophies {
		resources = append(resources, models.Resource{
			Type:    models.ResourceTypePhilosophy,
			Content: philosophy.Strategy,
			Metadata: map[string]string{
				"selfModelId": philosophy.Self_id,
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
			if err.Error() != "invalid self model data" {
				return err
			}

			// Otherwise, create the Self Model
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
