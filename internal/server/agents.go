package server

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
)

type AgentAttachmentService interface {
	AttachAgentIDs(ctx context.Context, serverID uuid.UUID, agentIDs []uuid.UUID) error
	CreateAgentsOnServer(ctx context.Context, serverID uuid.UUID, agents []AgentRegistrationInput) error
}

type AgentRegistrationInput struct {
	Name     string          `json:"name" binding:"required,min=1,max=255"`
	Type     string          `json:"type" binding:"omitempty,max=255"`
	Version  string          `json:"version" binding:"omitempty,max=255"`
	Metadata json.RawMessage `json:"metadata" binding:"omitempty,json"`
}
