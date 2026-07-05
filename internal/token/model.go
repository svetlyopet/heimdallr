package token

import (
	"github.com/google/uuid"
	"github.com/svetlyopet/heimdallr/internal/database/model"
	"gorm.io/gorm"
)

const (
	ScopeApplicationWrite = "application:write"
	ScopeAutomationWrite  = "automation:write"
	ScopeRead             = "read"
	ScopeAdmin            = "admin"
)

type APIToken struct {
	ID         uuid.UUID  `gorm:"type:uuid;primary_key" json:"id"`
	Name       string     `gorm:"type:varchar(255);not null;check:name <> ''" json:"name"`
	TokenHash  string     `gorm:"type:char(64);not null;uniqueIndex" json:"-"`
	Scopes     []string   `gorm:"serializer:json;type:text;not null" json:"scopes"`
	CreatedBy  *uuid.UUID `gorm:"type:uuid" json:"created_by"`
	model.Timestamp
}

func (t *APIToken) BeforeCreate(tx *gorm.DB) error {
	if t.ID == uuid.Nil {
		t.ID = uuid.New()
	}

	return nil
}

func (APIToken) TableName() string {
	return "api_tokens"
}
