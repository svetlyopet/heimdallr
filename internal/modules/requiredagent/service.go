package requiredagent

import (
	"context"
	"errors"
	"log/slog"
	"strings"

	"github.com/google/uuid"
	"github.com/svetlyopet/heimdallr/internal/database"
	"github.com/svetlyopet/heimdallr/internal/logger"
	"github.com/svetlyopet/heimdallr/internal/modules/requiredagent/api"
	"gorm.io/gorm"
)

type Service interface {
	GetAll(ctx context.Context, page int, limit int) ([]api.RequiredAgent, int64, error)
	GetById(ctx context.Context, requiredAgentID string) (api.RequiredAgent, error)
	Create(ctx context.Context, req api.RequiredAgentCreateRequest) (api.RequiredAgent, error)
	Update(ctx context.Context, req api.RequiredAgentUpdateRequest, requiredAgentID string) (api.RequiredAgent, error)
	Delete(ctx context.Context, requiredAgentID string) error
}

type service struct {
	repository Repository
	logger     *logger.Logger
}

func (s service) GetAll(ctx context.Context, page int, limit int) ([]api.RequiredAgent, int64, error) {
	offset := (page - 1) * limit

	requiredAgents, total, err := s.repository.FindAll(ctx, limit, offset)
	if err != nil {
		s.logger.ErrorWithStack(ctx, "failed to list required agents", err,
			slog.Int("page", page),
			slog.Int("limit", limit),
		)
		return nil, 0, ErrListRequiredAgents
	}

	responses := make([]api.RequiredAgent, 0, len(requiredAgents))
	for _, item := range requiredAgents {
		responses = append(responses, mapEntityToResponse(item))
	}

	return responses, total, nil
}

func (s service) GetById(ctx context.Context, requiredAgentID string) (api.RequiredAgent, error) {
	if _, err := uuid.Parse(requiredAgentID); err != nil {
		return api.RequiredAgent{}, ErrInvalidRequiredAgentID
	}

	requiredAgent, err := s.repository.FindById(ctx, requiredAgentID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return api.RequiredAgent{}, ErrRequiredAgentNotFound
		}

		s.logger.ErrorWithStack(ctx, "failed to get required agent", err,
			slog.String("required_agent_id", requiredAgentID),
		)
		return api.RequiredAgent{}, ErrGetRequiredAgent
	}

	return mapEntityToResponse(requiredAgent), nil
}

func (s service) Create(ctx context.Context, req api.RequiredAgentCreateRequest) (api.RequiredAgent, error) {
	agentName := normalizeAgentName(req.AgentName)
	if agentName == "" {
		return api.RequiredAgent{}, ErrCreateRequiredAgent
	}

	if _, err := s.repository.FindByName(ctx, agentName); err == nil {
		return api.RequiredAgent{}, ErrRequiredAgentAlreadyExists
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		s.logger.ErrorWithStack(ctx, "failed to check required agent existence before create", err,
			slog.String("agent_name", agentName),
		)
		return api.RequiredAgent{}, ErrCreateRequiredAgent
	}

	requiredAgent := RequiredAgent{
		ID:        uuid.New(),
		AgentName: agentName,
		AgentType: stringValue(req.AgentType),
	}

	created, err := s.repository.Create(ctx, requiredAgent)
	if err != nil {
		if database.IsUniqueViolation(err) {
			return api.RequiredAgent{}, ErrRequiredAgentAlreadyExists
		}

		s.logger.ErrorWithStack(ctx, "failed to create required agent", err,
			slog.String("agent_name", agentName),
		)
		return api.RequiredAgent{}, ErrCreateRequiredAgent
	}

	return mapEntityToResponse(created), nil
}

func (s service) Update(ctx context.Context, req api.RequiredAgentUpdateRequest, requiredAgentID string) (api.RequiredAgent, error) {
	if _, err := uuid.Parse(requiredAgentID); err != nil {
		return api.RequiredAgent{}, ErrInvalidRequiredAgentID
	}

	requiredAgent, err := s.repository.FindById(ctx, requiredAgentID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return api.RequiredAgent{}, ErrRequiredAgentNotFound
		}

		s.logger.ErrorWithStack(ctx, "failed to find required agent before update", err,
			slog.String("required_agent_id", requiredAgentID),
		)
		return api.RequiredAgent{}, ErrUpdateRequiredAgent
	}

	if req.AgentName != nil {
		agentName := normalizeAgentName(*req.AgentName)
		if agentName == "" {
			return api.RequiredAgent{}, ErrUpdateRequiredAgent
		}

		if !strings.EqualFold(agentName, requiredAgent.AgentName) {
			if _, findErr := s.repository.FindByName(ctx, agentName); findErr == nil {
				return api.RequiredAgent{}, ErrRequiredAgentAlreadyExists
			} else if !errors.Is(findErr, gorm.ErrRecordNotFound) {
				return api.RequiredAgent{}, ErrUpdateRequiredAgent
			}
		}

		requiredAgent.AgentName = agentName
	}

	if req.AgentType != nil {
		requiredAgent.AgentType = *req.AgentType
	}

	updated, err := s.repository.Update(ctx, requiredAgent)
	if err != nil {
		if database.IsUniqueViolation(err) {
			return api.RequiredAgent{}, ErrRequiredAgentAlreadyExists
		}

		s.logger.ErrorWithStack(ctx, "failed to update required agent", err,
			slog.String("required_agent_id", requiredAgentID),
		)
		return api.RequiredAgent{}, ErrUpdateRequiredAgent
	}

	return mapEntityToResponse(updated), nil
}

func (s service) Delete(ctx context.Context, requiredAgentID string) error {
	if _, err := uuid.Parse(requiredAgentID); err != nil {
		return ErrInvalidRequiredAgentID
	}

	if err := s.repository.Delete(ctx, requiredAgentID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrRequiredAgentNotFound
		}

		s.logger.ErrorWithStack(ctx, "failed to delete required agent", err,
			slog.String("required_agent_id", requiredAgentID),
		)
		return ErrDeleteRequiredAgent
	}

	return nil
}

func NewService(repository Repository, appLogger *logger.Logger) Service {
	if appLogger == nil {
		appLogger = logger.Default()
	}

	return &service{
		repository: repository,
		logger:     appLogger,
	}
}

func mapEntityToResponse(requiredAgent RequiredAgent) api.RequiredAgent {
	return api.RequiredAgent{
		Id:        requiredAgent.ID,
		AgentName: requiredAgent.AgentName,
		AgentType: requiredAgent.AgentType,
	}
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}

	return strings.TrimSpace(*value)
}
