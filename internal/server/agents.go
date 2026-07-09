package server

import (
	"context"

	"github.com/google/uuid"
	"github.com/svetlyopet/heimdallr/internal/server/api"
)

type AgentAttachmentService interface {
	AttachAgentIDs(ctx context.Context, serverID uuid.UUID, agentIDs []uuid.UUID) error
	CreateAgentsOnServer(ctx context.Context, serverID uuid.UUID, agents []api.AgentCreateRequest) error
}
