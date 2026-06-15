package provider

import (
	"github.com/google/uuid"
	"github.com/svetlyopet/heimdallr/internal/database/model"
	"gorm.io/gorm"
)

type Provider struct {
	ID   uuid.UUID `gorm:"type:uuid;primary_key" json:"id"`
	Name string    `gorm:"type:varchar(255);not null;check:name <> ''" json:"name"`
	Url  string    `gorm:"type:varchar(255);not null;check:url <> ''" json:"url"`

	model.Timestamp
}

func (p *Provider) BeforeCreate(tx *gorm.DB) error {
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}

	return nil
}
