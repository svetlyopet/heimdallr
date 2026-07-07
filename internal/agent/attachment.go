package agent

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/svetlyopet/heimdallr/internal/logger"
	"github.com/svetlyopet/heimdallr/internal/server"
	"gorm.io/gorm"
)

type attachmentService struct {
	repository          Repository
	serverLookupService server.LookupService
	logger              *logger.Logger
}

func NewAttachmentService(repository Repository, serverLookupService server.LookupService, appLogger *logger.Logger) server.AgentAttachmentService {
	if appLogger == nil {
		appLogger = logger.Default()
	}

	return &attachmentService{
		repository:          repository,
		serverLookupService: serverLookupService,
		logger:              appLogger,
	}
}

func (s attachmentService) AttachAgentIDs(ctx context.Context, serverID uuid.UUID, agentIDs []uuid.UUID) error {
	if len(agentIDs) == 0 {
		return nil
	}

	serverEntity, err := s.serverLookupService.GetById(ctx, serverID.String())
	if err != nil {
		if errors.Is(err, server.ErrServerNotFound) {
			return server.ErrServerNotFound
		}

		return server.ErrAttachAgents
	}

	if err := s.repository.AttachToServer(ctx, serverID, serverEntity.Hostname, agentIDs); err != nil {
		if errors.Is(err, ErrAgentAlreadyAssigned) {
			return server.ErrAgentAlreadyAssigned
		}

		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrAgentNotFound
		}

		return server.ErrAttachAgents
	}

	return nil
}

func (s attachmentService) CreateAgentsOnServer(ctx context.Context, serverID uuid.UUID, agents []server.AgentRegistrationInput) error {
	for _, input := range agents {
		serverIDCopy := serverID
		agent := Agent{
			ID:       uuid.New(),
			ServerID: &serverIDCopy,
			Name:     input.Name,
			Type:     input.Type,
			Version:  input.Version,
			Metadata: normalizeMetadata(input.Metadata),
		}

		if _, err := s.repository.CreateOnServer(ctx, agent); err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return server.ErrServerNotFound
			}

			return server.ErrAttachAgents
		}
	}

	return nil
}
