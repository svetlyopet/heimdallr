package agent

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/svetlyopet/heimdallr/internal/database"
	"github.com/svetlyopet/heimdallr/internal/logger"
	"github.com/svetlyopet/heimdallr/internal/server"
	serverapi "github.com/svetlyopet/heimdallr/internal/server/api"
	"gorm.io/gorm"
)

type attachmentService struct {
	repository          Repository
	serverLookupService server.LookupService
	db                  *gorm.DB
	logger              *logger.Logger
}

func NewAttachmentService(
	repository Repository,
	serverLookupService server.LookupService,
	db *gorm.DB,
	appLogger *logger.Logger,
) server.AgentAttachmentService {
	if appLogger == nil {
		appLogger = logger.Default()
	}

	return &attachmentService{
		repository:          repository,
		serverLookupService: serverLookupService,
		db:                  db,
		logger:              appLogger,
	}
}

func (s attachmentService) AttachAgentIDs(ctx context.Context, serverID uuid.UUID, agentIDs []uuid.UUID, tx *gorm.DB) error {
	if len(agentIDs) == 0 {
		return nil
	}

	if tx != nil {
		return s.attachAgentIDs(ctx, serverID, agentIDs, s.repository.WithTx(tx), true)
	}

	return database.WithTransaction(ctx, s.db, func(innerTx *gorm.DB) error {
		return s.attachAgentIDs(ctx, serverID, agentIDs, s.repository.WithTx(innerTx), false)
	})
}

func (s attachmentService) attachAgentIDs(ctx context.Context, serverID uuid.UUID, agentIDs []uuid.UUID, repo Repository, skipServerLookup bool) error {
	if !skipServerLookup {
		if _, err := s.serverLookupService.GetById(ctx, serverID.String()); err != nil {
			if errors.Is(err, server.ErrServerNotFound) {
				return server.ErrServerNotFound
			}

			return server.ErrAttachAgents
		}
	}

	if err := repo.AttachToServer(ctx, serverID, agentIDs); err != nil {
		if errors.Is(err, ErrAgentAlreadyLinked) {
			return server.ErrAgentAlreadyLinked
		}

		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrAgentNotFound
		}

		if database.IsUniqueViolation(err) {
			return server.ErrAgentAlreadyLinked
		}

		return server.ErrAttachAgents
	}

	return nil
}

func (s attachmentService) CreateAgentsOnServer(ctx context.Context, serverID uuid.UUID, agents []serverapi.AgentCreateRequest, tx *gorm.DB) error {
	if len(agents) == 0 {
		return nil
	}

	if tx != nil {
		return s.createAgentsOnServer(ctx, serverID, agents, s.repository.WithTx(tx))
	}

	return database.WithTransaction(ctx, s.db, func(innerTx *gorm.DB) error {
		return s.createAgentsOnServer(ctx, serverID, agents, s.repository.WithTx(innerTx))
	})
}

func (s attachmentService) createAgentsOnServer(ctx context.Context, serverID uuid.UUID, agents []serverapi.AgentCreateRequest, repo Repository) error {
	for _, input := range agents {
		if _, err := repo.FindByName(ctx, input.Name); err == nil {
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

		if _, err := repo.CreateOnServer(ctx, serverID, agent); err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return server.ErrServerNotFound
			}

			if database.IsUniqueViolation(err) {
				return server.ErrAgentAlreadyExists
			}

			return server.ErrAttachAgents
		}
	}

	return nil
}
