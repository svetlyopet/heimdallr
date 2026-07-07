package agent

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"

	"github.com/google/uuid"
	"github.com/svetlyopet/heimdallr/internal/logger"
	"github.com/svetlyopet/heimdallr/internal/server"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type Service interface {
	GetAll(ctx context.Context, serverID string, page int, limit int) ([]GetResponse, int64, error)
	GetById(ctx context.Context, agentID string, serverID string) (GetResponse, error)
	Create(ctx context.Context, serverID string, req CreateRequest) (GetResponse, error)
	Delete(ctx context.Context, serverID string, agentID string) error

	ListGlobal(ctx context.Context, unassignedOnly bool, page int, limit int) ([]GetResponse, int64, error)
	GetByIdGlobal(ctx context.Context, agentID string) (GetResponse, error)
	CreateUnassigned(ctx context.Context, req CreateRequest) (GetResponse, error)
}

type service struct {
	repository          Repository
	serverLookupService server.LookupService
	logger              *logger.Logger
}

func (s service) GetAll(ctx context.Context, serverID string, page int, limit int) ([]GetResponse, int64, error) {
	if _, err := uuid.Parse(serverID); err != nil {
		return nil, 0, ErrInvalidServerID
	}

	if _, err := s.serverLookupService.GetById(ctx, serverID); err != nil {
		if errors.Is(err, server.ErrServerNotFound) {
			return nil, 0, server.ErrServerNotFound
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

	return mapEntitiesToResponses(agents), total, nil
}

func (s service) GetById(ctx context.Context, agentID string, serverID string) (GetResponse, error) {
	agent, err := s.repository.FindById(ctx, agentID, serverID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return GetResponse{}, ErrAgentNotFound
		}

		s.logger.ErrorWithStack(ctx, "failed to find agent by id", err,
			slog.String("agent_id", agentID),
			slog.String("server_id", serverID),
		)
		return GetResponse{}, ErrGetAgent
	}

	return mapEntityToResponse(agent), nil
}

func (s service) Create(ctx context.Context, serverID string, req CreateRequest) (GetResponse, error) {
	parsedServerID, err := uuid.Parse(serverID)
	if err != nil {
		return GetResponse{}, ErrInvalidServerID
	}

	if _, err := s.serverLookupService.GetById(ctx, serverID); err != nil {
		if errors.Is(err, server.ErrServerNotFound) {
			return GetResponse{}, server.ErrServerNotFound
		}

		s.logger.ErrorWithStack(ctx, "failed to find server before creating agent", err,
			slog.String("server_id", serverID),
		)
		return GetResponse{}, ErrCreateAgent
	}

	serverIDCopy := parsedServerID
	agent := Agent{
		ID:       uuid.New(),
		ServerID: &serverIDCopy,
		Name:     req.Name,
		Type:     req.Type,
		Version:  req.Version,
		Metadata: normalizeMetadata(req.Metadata),
	}

	created, err := s.repository.CreateOnServer(ctx, agent)
	if err != nil {
		s.logger.ErrorWithStack(ctx, "failed to create agent", err,
			slog.String("server_id", serverID),
			slog.String("agent_name", req.Name),
		)
		return GetResponse{}, ErrCreateAgent
	}

	return mapEntityToResponse(created), nil
}

func (s service) Delete(ctx context.Context, serverID string, agentID string) error {
	if _, err := s.serverLookupService.GetById(ctx, serverID); err != nil {
		if errors.Is(err, server.ErrServerNotFound) {
			return server.ErrServerNotFound
		}

		return ErrDeleteAgent
	}

	if err := s.repository.Delete(ctx, serverID, agentID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrAgentNotFound
		}

		s.logger.ErrorWithStack(ctx, "failed to delete agent", err,
			slog.String("server_id", serverID),
			slog.String("agent_id", agentID),
		)
		return ErrDeleteAgent
	}

	return nil
}

func (s service) ListGlobal(ctx context.Context, unassignedOnly bool, page int, limit int) ([]GetResponse, int64, error) {
	offset := (page - 1) * limit

	agents, total, err := s.repository.FindAllGlobal(ctx, unassignedOnly, limit, offset)
	if err != nil {
		s.logger.ErrorWithStack(ctx, "failed to list global agents", err,
			slog.Bool("unassigned_only", unassignedOnly),
			slog.Int("page", page),
			slog.Int("limit", limit),
		)
		return nil, 0, ErrListAgents
	}

	return mapEntitiesToResponses(agents), total, nil
}

func (s service) GetByIdGlobal(ctx context.Context, agentID string) (GetResponse, error) {
	if _, err := uuid.Parse(agentID); err != nil {
		return GetResponse{}, ErrInvalidAgentID
	}

	agent, err := s.repository.FindByIdGlobal(ctx, agentID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return GetResponse{}, ErrAgentNotFound
		}

		s.logger.ErrorWithStack(ctx, "failed to find agent by id", err,
			slog.String("agent_id", agentID),
		)
		return GetResponse{}, ErrGetAgent
	}

	return mapEntityToResponse(agent), nil
}

func (s service) CreateUnassigned(ctx context.Context, req CreateRequest) (GetResponse, error) {
	agent := Agent{
		ID:       uuid.New(),
		Name:     req.Name,
		Type:     req.Type,
		Version:  req.Version,
		Metadata: normalizeMetadata(req.Metadata),
	}

	created, err := s.repository.CreateUnassigned(ctx, agent)
	if err != nil {
		s.logger.ErrorWithStack(ctx, "failed to create unassigned agent", err,
			slog.String("agent_name", req.Name),
		)
		return GetResponse{}, ErrCreateAgent
	}

	return mapEntityToResponse(created), nil
}

func NewService(repository Repository, serverLookupService server.LookupService, appLogger *logger.Logger) Service {
	if appLogger == nil {
		appLogger = logger.Default()
	}

	return &service{
		repository:          repository,
		serverLookupService: serverLookupService,
		logger:              appLogger,
	}
}

func mapEntitiesToResponses(agents []Agent) []GetResponse {
	responses := make([]GetResponse, 0, len(agents))
	for _, agent := range agents {
		responses = append(responses, mapEntityToResponse(agent))
	}

	return responses
}

func mapEntityToResponse(agent Agent) GetResponse {
	metadata := json.RawMessage(agent.Metadata)
	if len(metadata) == 0 {
		metadata = json.RawMessage(`{}`)
	}

	return GetResponse{
		ID:       agent.ID,
		Server:   agent.Server,
		ServerID: agent.ServerID,
		Name:     agent.Name,
		Type:     agent.Type,
		Version:  agent.Version,
		Metadata: metadata,
	}
}

func normalizeMetadata(raw json.RawMessage) datatypes.JSON {
	if len(raw) == 0 {
		return datatypes.JSON([]byte(`{}`))
	}

	return datatypes.JSON(raw)
}
