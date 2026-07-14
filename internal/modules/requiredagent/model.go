package requiredagent

import (
	"github.com/google/uuid"
	"github.com/svetlyopet/heimdallr/internal/database/model"
	"gorm.io/gorm"
)

type RequiredAgent struct {
	ID        uuid.UUID `gorm:"type:uuid;primary_key" json:"id"`
	AgentName string    `gorm:"type:varchar(255);not null;check:agent_name <> ''" json:"agent_name"`
	AgentType string    `gorm:"type:varchar(255);not null" json:"agent_type"`
	model.Timestamp
}

func (RequiredAgent) TableName() string {
	return "required_agents"
}

func (r *RequiredAgent) BeforeCreate(tx *gorm.DB) error {
	if r.ID == uuid.Nil {
		r.ID = uuid.New()
	}

	return nil
}
