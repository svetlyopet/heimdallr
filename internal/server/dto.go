package server

import (
	"encoding/json"

	"github.com/google/uuid"
)

type GetResponse struct {
	ID              uuid.UUID       `json:"id"`
	Hostname        string          `json:"hostname"`
	Metadata        json.RawMessage `json:"metadata"`
	OperatingSystem string          `json:"operating_system"`
	Hypervisor      string          `json:"hypervisor"`
	Location        string          `json:"location"`
}

type RelationSummary struct {
	AgentCount   int64 `json:"agent_count"`
	JobCount     int64 `json:"job_count"`
	ReleaseCount int64 `json:"release_count"`
}

type ListItemResponse struct {
	GetResponse
	Relations RelationSummary `json:"relations"`
}

type GetWithRelationsResponse struct {
	GetResponse
	Relations RelationSummary `json:"relations"`
}

type CreateRequest struct {
	Hostname        string                   `json:"hostname" binding:"required,min=1,max=255"`
	OperatingSystem string                   `json:"operating_system" binding:"omitempty,max=255"`
	Hypervisor      string                   `json:"hypervisor" binding:"omitempty,max=255"`
	Location        string                   `json:"location" binding:"omitempty,max=255"`
	Metadata        json.RawMessage          `json:"metadata" binding:"omitempty,json"`
	AgentIDs        []uuid.UUID              `json:"agent_ids" binding:"omitempty,dive"`
	Agents          []AgentRegistrationInput `json:"agents" binding:"omitempty,dive"`
}

type UpdateRequest struct {
	AgentIDs []uuid.UUID              `json:"agent_ids" binding:"omitempty,dive"`
	Agents   []AgentRegistrationInput `json:"agents" binding:"omitempty,dive"`
}

type JobAssociateRequest struct {
	JobID        string    `json:"job_id" binding:"required,min=1"`
	AutomationID uuid.UUID `json:"automation_id" binding:"required"`
}

type JobAssociationResponse struct {
	JobID        string    `json:"job_id"`
	AutomationID uuid.UUID `json:"automation_id"`
	Automation   string    `json:"automation"`
	Provider     string    `json:"provider"`
	Status       string    `json:"status"`
	Location     string    `json:"location"`
	URL          string    `json:"url"`
}

type ReleaseAssociateRequest struct {
	ReleaseID     uuid.UUID `json:"release_id" binding:"required"`
	ApplicationID uuid.UUID `json:"application_id" binding:"required"`
}

type ReleaseAssociationResponse struct {
	ReleaseID     uuid.UUID `json:"release_id"`
	ApplicationID uuid.UUID `json:"application_id"`
	Application   string    `json:"application"`
	Version       string    `json:"version"`
	CommitSHA     string    `json:"commit_sha"`
	Branch        string    `json:"branch"`
}
