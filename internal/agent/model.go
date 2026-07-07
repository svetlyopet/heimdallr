package agent

import (
	"github.com/google/uuid"
	"github.com/svetlyopet/heimdallr/internal/database/model"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type Agent struct {
	ID       uuid.UUID      `gorm:"type:uuid;primary_key" json:"id"`
	Server   string         `gorm:"type:varchar(255);not null;default:''" json:"server"`
	ServerID *uuid.UUID     `gorm:"type:uuid;index" json:"server_id"`
	Name     string         `gorm:"type:varchar(255);not null;check:name <> ''" json:"name"`
	Type     string         `gorm:"type:varchar(255);not null" json:"type"`
	Version  string         `gorm:"type:varchar(255);not null" json:"version"`
	Metadata datatypes.JSON `gorm:"type:json;not null" json:"metadata"`
	model.Timestamp
}

func (Agent) TableName() string {
	return "agents"
}

func (a *Agent) BeforeCreate(tx *gorm.DB) error {
	if a.ID == uuid.Nil {
		a.ID = uuid.New()
	}

	return nil
}
