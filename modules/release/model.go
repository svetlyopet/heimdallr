package release

import (
	"github.com/google/uuid"
	"github.com/svetlyopet/heimdallr/internal/database/model"
	"gorm.io/gorm"
)

type Release struct {
	ID            uuid.UUID `gorm:"type:uuid;primary_key" json:"id"`
	ApplicationID uuid.UUID `gorm:"type:uuid;not null;index;uniqueIndex:idx_release_app_version" json:"application_id"`
	Application   string    `gorm:"type:varchar(255);not null" json:"application"`
	Version       string    `gorm:"type:varchar(255);not null;uniqueIndex:idx_release_app_version" json:"version"`
	CommitSHA     string    `gorm:"type:varchar(255);not null" json:"commit_sha"`
	PipelineURL   string    `gorm:"type:varchar(255);not null" json:"pipeline_url"`
	Branch        string    `gorm:"type:varchar(255);not null" json:"branch"`
	model.Timestamp
}

func (r *Release) BeforeCreate(tx *gorm.DB) error {
	if r.ID == uuid.Nil {
		r.ID = uuid.New()
	}

	return nil
}
