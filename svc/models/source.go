package models

import (
	pbmodels "epistemic-me-core/pb/models"
)

type SourceType int32

const (
	SourceTypeInvalid SourceType = iota
	SourceTypeUser
	SourceTypeSystem
	SourceTypeDocument
	SourceTypeSensor
)

type ResourceType int32

const (
	ResourceTypeInvalid ResourceType = iota
	ResourceTypeChatLog
	ResourceTypeScientificPaper
	ResourceTypeSurveyResponse
	ResourceTypeMeasurementData
)

type Resource struct {
	ID       string            `json:"id"`
	Type     ResourceType      `json:"type"`
	Content  string            `json:"content"`
	SourceID string            `json:"source_id"`
	ActionID string            `json:"action_id"`
	Metadata map[string]string `json:"metadata"`
}

type Source struct {
	ID   string     `json:"id"`
	Type SourceType `json:"type"`
	Name string     `json:"name"`
}

func (s Source) ToProto() *pbmodels.Source {
	// Implement the conversion
	return &pbmodels.Source{}
}

func sourcesToProto(sources []*Source) []*pbmodels.Source {
	result := make([]*pbmodels.Source, len(sources))
	for i, s := range sources {
		result[i] = s.ToProto()
	}
	return result
}

func (r *Resource) ToProto() *pbmodels.Resource {
	return &pbmodels.Resource{
		Id:       r.ID,
		Type:     pbmodels.ResourceType(r.Type),
		Content:  r.Content,
		SourceId: r.SourceID,
		Metadata: r.Metadata,
	}
}
