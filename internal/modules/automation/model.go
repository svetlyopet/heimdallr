package automation

import (
	"github.com/google/uuid"
	"github.com/svetlyopet/heimdallr/internal/database/model"
	"gorm.io/gorm"
)

type Automation struct {
	ID          uuid.UUID `gorm:"type:uuid;primary_key" json:"id"`
	Name        string    `gorm:"type:varchar(255);not null;check:name <> ''" json:"name"`
	Url         string    `gorm:"type:varchar(255);not null;check:url <> ''" json:"url"`
	Provider    string    `gorm:"type:varchar(255);not null;check:provider <> ''" json:"provider"`
	ProviderID  uuid.UUID `gorm:"type:uuid;not null;index;check:provider_id <> ''" json:"provider_id"`
	CostSavings float64   `gorm:"type:float" json:"cost_savings"`

	model.Timestamp
}

func (a *Automation) BeforeCreate(tx *gorm.DB) error {
	if a.ID == uuid.Nil {
		a.ID = uuid.New()
	}

	return nil
}
