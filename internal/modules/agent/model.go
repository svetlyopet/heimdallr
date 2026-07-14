package agent

import (
	"time"

	"github.com/google/uuid"
	"github.com/svetlyopet/heimdallr/internal/database/model"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type Agent struct {
	ID       uuid.UUID      `gorm:"type:uuid;primary_key" json:"id"`
	Name     string         `gorm:"type:varchar(255);not null;uniqueIndex;check:name <> ''" json:"name"`
	Type     string         `gorm:"type:varchar(255);not null" json:"type"`
	Metadata datatypes.JSON `gorm:"type:json;not null" json:"metadata"`
	model.Timestamp
}

type ServerAgent struct {
	ServerID  uuid.UUID `gorm:"type:uuid;primaryKey" json:"server_id"`
	AgentID   uuid.UUID `gorm:"type:uuid;primaryKey" json:"agent_id"`
	CreatedAt time.Time `gorm:"not null" json:"created_at"`
}

type AgentWithCount struct {
	Agent
	ServerCount int64 `gorm:"column:server_count"`
}

type LinkedServer struct {
	ID       uuid.UUID `gorm:"column:id"`
	Hostname string    `gorm:"column:hostname"`
}

func (Agent) TableName() string {
	return "agents"
}

func (ServerAgent) TableName() string {
	return "server_agents"
}

func (a *Agent) BeforeCreate(tx *gorm.DB) error {
	if a.ID == uuid.Nil {
		a.ID = uuid.New()
	}

	return nil
}
