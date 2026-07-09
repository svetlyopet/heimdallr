package agent

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/svetlyopet/heimdallr/internal/logger"
	"github.com/svetlyopet/heimdallr/internal/server"
	serverapi "github.com/svetlyopet/heimdallr/internal/server/api"
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

	if _, err := s.serverLookupService.GetById(ctx, serverID.String()); err != nil {
		if errors.Is(err, server.ErrServerNotFound) {
			return server.ErrServerNotFound
		}

		return server.ErrAttachAgents
	}

	if err := s.repository.AttachToServer(ctx, serverID, agentIDs); err != nil {
		if errors.Is(err, ErrAgentAlreadyLinked) {
			return server.ErrAgentAlreadyLinked
		}

		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrAgentNotFound
		}

		return server.ErrAttachAgents
	}

	return nil
}

func (s attachmentService) CreateAgentsOnServer(ctx context.Context, serverID uuid.UUID, agents []serverapi.AgentCreateRequest) error {
	for _, input := range agents {
		if _, err := s.repository.FindByName(ctx, input.Name); err == nil {
			return server.ErrAgentAlreadyExists
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return server.ErrAttachAgents
		}

		agent := Agent{
			ID:       uuid.New(),
			Name:     input.Name,
			Type:     stringValue(input.Type),
			Version:  stringValue(input.Version),
			Metadata: metadataToEntity(convertServerMetadata(input.Metadata)),
		}

		if _, err := s.repository.CreateOnServer(ctx, serverID, agent); err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return server.ErrServerNotFound
			}

			if isUniqueViolation(err) {
				return server.ErrAgentAlreadyExists
			}

			return server.ErrAttachAgents
		}
	}

	return nil
}
