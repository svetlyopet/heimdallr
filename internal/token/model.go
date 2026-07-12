package token

import (
	"time"

	"github.com/google/uuid"
	"github.com/svetlyopet/heimdallr/internal/database/model"
	"gorm.io/gorm"
)

const (
	TokenKindSession = "session"
	TokenKindAPI     = "api"
)

type APIToken struct {
	ID        uuid.UUID  `gorm:"type:uuid;primary_key" json:"id"`
	Name      string     `gorm:"type:varchar(255);not null;check:name <> ''" json:"name"`
	TokenHash string     `gorm:"type:char(64);not null;uniqueIndex" json:"-"`
	Scopes    []string   `gorm:"serializer:json;type:text;not null" json:"scopes"`
	Kind      string     `gorm:"type:varchar(32);not null;default:api" json:"kind"`
	ExpiresAt *time.Time `gorm:"not null" json:"expires_at,omitempty"`
	CreatedBy *uuid.UUID `gorm:"type:uuid" json:"created_by"`
	model.Timestamp
}

func (t *APIToken) BeforeCreate(tx *gorm.DB) error {
	if t.ID == uuid.Nil {
		t.ID = uuid.New()
	}

	if t.Kind == "" {
		t.Kind = TokenKindAPI
	}

	return nil
}

func (APIToken) TableName() string {
	return "api_tokens"
}
