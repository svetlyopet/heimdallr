package application

import (
	"github.com/google/uuid"
	"github.com/svetlyopet/heimdallr/internal/database/model"
	"gorm.io/gorm"
)

type Application struct {
	ID            uuid.UUID `gorm:"type:uuid;primary_key" json:"id"`
	Name          string    `gorm:"type:varchar(255);not null;uniqueIndex;check:name <> ''" json:"name"`
	Description   string    `gorm:"type:text;not null" json:"description"`
	RepositoryURL string    `gorm:"type:varchar(255);not null" json:"repository_url"`
	model.Timestamp
}

func (a *Application) BeforeCreate(tx *gorm.DB) error {
	if a.ID == uuid.Nil {
		a.ID = uuid.New()
	}

	return nil
}
