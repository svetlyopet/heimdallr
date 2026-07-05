package report

import (
	"github.com/google/uuid"
	"github.com/svetlyopet/heimdallr/internal/database/model"
	"gorm.io/datatypes"
)

type Report struct {
	ID            string         `gorm:"type:varchar(255);primaryKey" json:"id"`
	ReleaseID     uuid.UUID      `gorm:"type:uuid;primaryKey" json:"release_id"`
	ApplicationID uuid.UUID      `gorm:"type:uuid;not null;index" json:"application_id"`
	Application   string         `gorm:"type:varchar(255);not null" json:"application"`
	Version       string         `gorm:"type:varchar(255);not null" json:"version"`
	Type          string         `gorm:"type:varchar(255);not null" json:"type"`
	Status        string         `gorm:"type:varchar(255);not null" json:"status"`
	Location      string         `gorm:"type:varchar(255);not null" json:"location"`
	URL           string         `gorm:"type:varchar(255);not null" json:"url"`
	Metadata      datatypes.JSON `gorm:"type:json;not null" json:"metadata"`
	Output        string         `gorm:"type:text;not null" json:"output"`
	model.Timestamp
}
