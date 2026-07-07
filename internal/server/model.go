package server

import (
	"github.com/google/uuid"
	"github.com/svetlyopet/heimdallr/internal/database/model"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type Server struct {
	ID              uuid.UUID      `gorm:"type:uuid;primary_key" json:"id"`
	Hostname        string         `gorm:"type:varchar(255);not null;uniqueIndex;check:hostname <> ''" json:"hostname"`
	Metadata        datatypes.JSON `gorm:"type:json;not null" json:"metadata"`
	OperatingSystem string         `gorm:"type:varchar(255);not null" json:"operating_system"`
	Hypervisor      string         `gorm:"type:varchar(255);not null" json:"hypervisor"`
	Location        string         `gorm:"type:varchar(255);not null" json:"location"`
	model.Timestamp
}

func (s *Server) BeforeCreate(tx *gorm.DB) error {
	if s.ID == uuid.Nil {
		s.ID = uuid.New()
	}

	return nil
}

type ServerJob struct {
	ServerID     uuid.UUID `gorm:"type:uuid;primaryKey" json:"server_id"`
	JobID        string    `gorm:"type:varchar(255);primaryKey" json:"job_id"`
	AutomationID uuid.UUID `gorm:"type:uuid;primaryKey" json:"automation_id"`
}

func (ServerJob) TableName() string {
	return "server_jobs"
}

type ServerRelease struct {
	ServerID      uuid.UUID `gorm:"type:uuid;primaryKey" json:"server_id"`
	ReleaseID     uuid.UUID `gorm:"type:uuid;primaryKey" json:"release_id"`
	ApplicationID uuid.UUID `gorm:"type:uuid;not null" json:"application_id"`
}

func (ServerRelease) TableName() string {
	return "server_releases"
}
