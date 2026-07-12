package agent

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"

	"github.com/google/uuid"
	"github.com/svetlyopet/heimdallr/internal/database"
	"github.com/svetlyopet/heimdallr/internal/logger"
	"github.com/svetlyopet/heimdallr/internal/modules/agent/api"
	server2 "github.com/svetlyopet/heimdallr/internal/modules/server"
	serverapi "github.com/svetlyopet/heimdallr/internal/modules/server/api"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type Service interface {
	GetAll(ctx context.Context, serverID string, page int, limit int) ([]api.Agent, int64, error)
	GetById(ctx context.Context, agentID string, serverID string) (api.Agent, error)
	CreateOnServer(ctx context.Context, serverID string, req api.ServerAgentRequest) (api.Agent, error)
	Detach(ctx context.Context, serverID string, agentID string) error

	ListGlobal(ctx context.Context, filter ListFilters, page int, limit int) ([]api.Agent, int64, error)
	GetByIdGlobal(ctx context.Context, agentID string) (api.AgentDetail, error)
	ListServers(ctx context.Context, agentID string, page int, limit int) ([]api.AgentServer, int64, error)
	CreateUnassigned(ctx context.Context, req api.AgentCreateRequest) (api.Agent, error)
	DeleteGlobal(ctx context.Context, agentID string) error
}

type service struct {
	repository          Repository
	serverLookupService server2.LookupService
	db                  *gorm.DB
	logger              *logger.Logger
}

func (s service) GetAll(ctx context.Context, serverID string, page int, limit int) ([]api.Agent, int64, error) {
	if _, err := uuid.Parse(serverID); err != nil {
		return nil, 0, ErrInvalidServerID
	}

	if _, err := s.serverLookupService.GetById(ctx, serverID); err != nil {
		if errors.Is(err, server2.ErrServerNotFound) {
			return nil, 0, server2.ErrServerNotFound
		}

		return nil, 0, ErrListAgents
	}

	offset := (page - 1) * limit

	agents, total, err := s.repository.FindAll(ctx, serverID, limit, offset)
	if err != nil {
		s.logger.ErrorWithStack(ctx, "failed to list agents", err,
			slog.String("server_id", serverID),
			slog.Int("page", page),
			slog.Int("limit", limit),
		)
		return nil, 0, ErrListAgents
	}

	responses := make([]api.Agent, 0, len(agents))
	for _, agent := range agents {
		item, err := mapEntityToResponse(ctx, agent, 1, ErrListAgents, s.logger)
		if err != nil {
			return nil, 0, err
		}
		responses = append(responses, item)
	}

	return responses, total, nil
}

func (s service) GetById(ctx context.Context, agentID string, serverID string) (api.Agent, error) {
	agent, err := s.repository.FindById(ctx, agentID, serverID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return api.Agent{}, ErrAgentNotFound
		}

		s.logger.ErrorWithStack(ctx, "failed to find agent by id", err,
			slog.String("agent_id", agentID),
			slog.String("server_id", serverID),
		)
		return api.Agent{}, ErrGetAgent
	}

	return mapEntityToResponse(ctx, agent, 1, ErrGetAgent, s.logger)
}

func (s service) CreateOnServer(ctx context.Context, serverID string, req api.ServerAgentRequest) (api.Agent, error) {
	parsedServerID, err := uuid.Parse(serverID)
	if err != nil {
		return api.Agent{}, ErrInvalidServerID
	}

	if _, err := s.serverLookupService.GetById(ctx, serverID); err != nil {
		if errors.Is(err, server2.ErrServerNotFound) {
			return api.Agent{}, server2.ErrServerNotFound
		}

		s.logger.ErrorWithStack(ctx, "failed to find server before creating agent", err,
			slog.String("server_id", serverID),
		)
		return api.Agent{}, ErrCreateAgent
	}

	if req.AgentId != nil {
		var agent Agent

		err := database.WithTransaction(ctx, s.db, func(tx *gorm.DB) error {
			txRepo := s.repository.WithTx(tx)

			if err := txRepo.AttachToServer(ctx, parsedServerID, []uuid.UUID{*req.AgentId}); err != nil {
				return err
			}

			found, err := txRepo.FindById(ctx, req.AgentId.String(), serverID)
			if err != nil {
				return err
			}

			agent = found
			return nil
		})
		if err != nil {
			if errors.Is(err, ErrAgentAlreadyLinked) {
				return api.Agent{}, ErrAgentAlreadyLinked
			}

			if errors.Is(err, gorm.ErrRecordNotFound) {
				return api.Agent{}, ErrAgentNotFound
			}

			if database.IsUniqueViolation(err) {
				return api.Agent{}, ErrAgentAlreadyLinked
			}

			return api.Agent{}, ErrCreateAgent
		}

		return mapEntityToResponse(ctx, agent, 1, ErrGetAgent, s.logger)
	}

	name := stringValue(req.Name)
	if name == "" {
		return api.Agent{}, ErrCreateAgent
	}

	if err := s.ensureAgentNameAvailable(ctx, name); err != nil {
		return api.Agent{}, err
	}

	metadata, err := metadataToEntity(req.Metadata)
	if err != nil {
		return api.Agent{}, ErrCreateAgent
	}

	agent := Agent{
		ID:       uuid.New(),
		Name:     name,
		Type:     stringValue(req.Type),
		Version:  stringValue(req.Version),
		Metadata: metadata,
	}

	var created Agent

	err = database.WithTransaction(ctx, s.db, func(tx *gorm.DB) error {
		var createErr error
		created, createErr = s.repository.WithTx(tx).CreateOnServer(ctx, parsedServerID, agent)
		return createErr
	})
	if err != nil {
		if database.IsUniqueViolation(err) {
			return api.Agent{}, ErrAgentAlreadyExists
		}

		s.logger.ErrorWithStack(ctx, "failed to create agent on server", err,
			slog.String("server_id", serverID),
			slog.String("agent_name", name),
		)
		return api.Agent{}, ErrCreateAgent
	}

	return mapEntityToResponse(ctx, created, 1, ErrGetAgent, s.logger)
}

func (s service) Detach(ctx context.Context, serverID string, agentID string) error {
	if _, err := s.serverLookupService.GetById(ctx, serverID); err != nil {
		if errors.Is(err, server2.ErrServerNotFound) {
			return server2.ErrServerNotFound
		}

		return ErrDeleteAgent
	}

	if err := s.repository.DetachFromServer(ctx, serverID, agentID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrAgentNotFound
		}

		s.logger.ErrorWithStack(ctx, "failed to detach agent", err,
			slog.String("server_id", serverID),
			slog.String("agent_id", agentID),
		)
		return ErrDeleteAgent
	}

	return nil
}

func (s service) ListGlobal(ctx context.Context, filter ListFilters, page int, limit int) ([]api.Agent, int64, error) {
	offset := (page - 1) * limit

	agents, total, err := s.repository.FindAllGlobal(ctx, filter, limit, offset)
	if err != nil {
		s.logger.ErrorWithStack(ctx, "failed to list global agents", err,
			slog.Bool("unassigned_only", filter.UnassignedOnly),
			slog.String("server_id", filter.ServerID),
			slog.String("agent_id", filter.AgentID),
			slog.Int("page", page),
			slog.Int("limit", limit),
		)
		return nil, 0, ErrListAgents
	}

	responses := make([]api.Agent, 0, len(agents))
	for _, agent := range agents {
		item, err := mapAgentWithCountToResponse(ctx, agent, ErrListAgents, s.logger)
		if err != nil {
			return nil, 0, err
		}
		responses = append(responses, item)
	}

	return responses, total, nil
}

func (s service) GetByIdGlobal(ctx context.Context, agentID string) (api.AgentDetail, error) {
	if _, err := uuid.Parse(agentID); err != nil {
		return api.AgentDetail{}, ErrInvalidAgentID
	}

	agent, err := s.repository.FindByIdGlobal(ctx, agentID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return api.AgentDetail{}, ErrAgentNotFound
		}

		s.logger.ErrorWithStack(ctx, "failed to find agent by id", err,
			slog.String("agent_id", agentID),
		)
		return api.AgentDetail{}, ErrGetAgent
	}

	servers, _, err := s.repository.FindServersByAgent(ctx, agentID, 0, 0)
	if err != nil {
		return api.AgentDetail{}, ErrGetAgent
	}

	base, err := mapAgentWithCountToResponse(ctx, agent, ErrGetAgent, s.logger)
	if err != nil {
		return api.AgentDetail{}, err
	}

	return api.AgentDetail{
		Id:          base.Id,
		Name:        base.Name,
		Type:        base.Type,
		Version:     base.Version,
		Metadata:    base.Metadata,
		ServerCount: base.ServerCount,
		Servers:     mapLinkedServersToResponses(servers),
	}, nil
}

func (s service) ListServers(ctx context.Context, agentID string, page int, limit int) ([]api.AgentServer, int64, error) {
	if _, err := uuid.Parse(agentID); err != nil {
		return nil, 0, ErrInvalidAgentID
	}

	if _, err := s.repository.FindByIdGlobal(ctx, agentID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, 0, ErrAgentNotFound
		}

		return nil, 0, ErrGetAgent
	}

	offset := (page - 1) * limit

	servers, total, err := s.repository.FindServersByAgent(ctx, agentID, limit, offset)
	if err != nil {
		return nil, 0, ErrGetAgent
	}

	return mapLinkedServersToResponses(servers), total, nil
}

func (s service) CreateUnassigned(ctx context.Context, req api.AgentCreateRequest) (api.Agent, error) {
	if err := s.ensureAgentNameAvailable(ctx, req.Name); err != nil {
		return api.Agent{}, err
	}

	metadata, err := metadataToEntity(req.Metadata)
	if err != nil {
		return api.Agent{}, ErrCreateAgent
	}

	agent := Agent{
		ID:       uuid.New(),
		Name:     req.Name,
		Type:     stringValue(req.Type),
		Version:  stringValue(req.Version),
		Metadata: metadata,
	}

	created, err := s.repository.CreateUnassigned(ctx, agent)
	if err != nil {
		if database.IsUniqueViolation(err) {
			return api.Agent{}, ErrAgentAlreadyExists
		}

		s.logger.ErrorWithStack(ctx, "failed to create unassigned agent", err,
			slog.String("agent_name", req.Name),
		)
		return api.Agent{}, ErrCreateAgent
	}

	return mapEntityToResponse(ctx, created, 0, ErrGetAgent, s.logger)
}

func (s service) DeleteGlobal(ctx context.Context, agentID string) error {
	if _, err := uuid.Parse(agentID); err != nil {
		return ErrInvalidAgentID
	}

	if err := s.repository.DeleteGlobal(ctx, agentID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrAgentNotFound
		}

		s.logger.ErrorWithStack(ctx, "failed to delete agent", err,
			slog.String("agent_id", agentID),
		)
		return ErrDeleteAgent
	}

	return nil
}

func (s service) ensureAgentNameAvailable(ctx context.Context, name string) error {
	_, err := s.repository.FindByName(ctx, name)
	if err == nil {
		return ErrAgentAlreadyExists
	}

	if !errors.Is(err, gorm.ErrRecordNotFound) {
		s.logger.ErrorWithStack(ctx, "failed to check agent existence before create", err,
			slog.String("agent_name", name),
		)
		return ErrCreateAgent
	}

	return nil
}

func NewService(repository Repository, serverLookupService server2.LookupService, db *gorm.DB, appLogger *logger.Logger) Service {
	if appLogger == nil {
		appLogger = logger.Default()
	}

	return &service{
		repository:          repository,
		serverLookupService: serverLookupService,
		db:                  db,
		logger:              appLogger,
	}
}

func mapEntityToResponse(ctx context.Context, agent Agent, serverCount int, boundaryErr error, appLogger *logger.Logger) (api.Agent, error) {
	metadata, err := metadataFromEntity(agent.Metadata)
	if err != nil {
		appLogger.ErrorWithStack(ctx, "failed to decode stored agent metadata", err,
			slog.String("entity_type", "agent"),
			slog.String("agent_id", agent.ID.String()),
		)
		return api.Agent{}, boundaryErr
	}

	return api.Agent{
		Id:          agent.ID,
		Name:        agent.Name,
		Type:        agent.Type,
		Version:     agent.Version,
		Metadata:    metadata,
		ServerCount: serverCount,
	}, nil
}

func mapAgentWithCountToResponse(ctx context.Context, agent AgentWithCount, boundaryErr error, appLogger *logger.Logger) (api.Agent, error) {
	return mapEntityToResponse(ctx, agent.Agent, int(agent.ServerCount), boundaryErr, appLogger)
}

func mapLinkedServersToResponses(servers []LinkedServer) []api.AgentServer {
	responses := make([]api.AgentServer, 0, len(servers))
	for _, serverEntity := range servers {
		responses = append(responses, api.AgentServer{
			Id:       serverEntity.ID,
			Hostname: serverEntity.Hostname,
		})
	}

	return responses
}

func metadataFromEntity(raw datatypes.JSON) (api.ServerMetadata, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return api.ServerMetadata{}, nil
	}

	var metadata api.ServerMetadata
	if err := json.Unmarshal(raw, &metadata); err != nil {
		return nil, ErrCorruptMetadata
	}

	return metadata, nil
}

func metadataToEntity(metadata *api.ServerMetadata) (datatypes.JSON, error) {
	if metadata == nil || len(*metadata) == 0 {
		return datatypes.JSON([]byte(`{}`)), nil
	}

	raw, err := json.Marshal(metadata)
	if err != nil {
		return nil, err
	}

	return datatypes.JSON(raw), nil
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}

	return *value
}

func convertServerMetadata(metadata *serverapi.ServerMetadata) *api.ServerMetadata {
	if metadata == nil {
		return nil
	}

	converted := api.ServerMetadata(*metadata)
	return &converted
}
