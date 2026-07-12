package job

import (
	"github.com/google/uuid"
	"github.com/svetlyopet/heimdallr/internal/database/model"
	"gorm.io/datatypes"
)

type Job struct {
	ID           string         `gorm:"type:varchar(255);primaryKey" json:"id"`
	Automation   string         `gorm:"type:varchar(255);not null;check:automation <> ''" json:"automation"`
	AutomationID uuid.UUID      `gorm:"type:uuid;primaryKey" json:"automation_id"`
	Provider     string         `gorm:"type:varchar(255);not null;check:provider <> ''" json:"provider"`
	ProviderID   uuid.UUID      `gorm:"type:uuid;not null;index;check:provider_id <> ''" json:"provider_id"`
	Status       string         `gorm:"type:varchar(255);not null;check:status <> ''" json:"status"`
	Location     string         `gorm:"type:varchar(255);not null;check:location <> ''" json:"location"`
	Url          string         `gorm:"type:varchar(255);not null;check:url <> ''" json:"url"`
	Metadata     datatypes.JSON `gorm:"type:json;not null" json:"metadata"`
	Output       string         `gorm:"type:text;not null" json:"output"`

	model.Timestamp
}
