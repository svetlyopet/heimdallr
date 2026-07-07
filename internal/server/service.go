package server

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"

	"github.com/google/uuid"
	"github.com/svetlyopet/heimdallr/internal/logger"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type LookupService interface {
	GetById(ctx context.Context, serverID string) (GetResponse, error)
}

type Service interface {
	GetAll(ctx context.Context, page int, limit int) ([]ListItemResponse, int64, error)
	GetById(ctx context.Context, serverID string) (GetWithRelationsResponse, error)
	Create(ctx context.Context, req CreateRequest) (GetResponse, error)
	Update(ctx context.Context, serverID string, req UpdateRequest) (GetWithRelationsResponse, error)

	ListJobs(ctx context.Context, serverID string, page int, limit int) ([]JobAssociationResponse, int64, error)
	AssociateJob(ctx context.Context, serverID string, req JobAssociateRequest) error
	DissociateJob(ctx context.Context, serverID string, jobID string, automationID uuid.UUID) error

	ListReleases(ctx context.Context, serverID string, page int, limit int) ([]ReleaseAssociationResponse, int64, error)
	AssociateRelease(ctx context.Context, serverID string, req ReleaseAssociateRequest) error
	DissociateRelease(ctx context.Context, serverID string, releaseID uuid.UUID) error
}

type service struct {
	repository      Repository
	agentAttachment AgentAttachmentService
	logger          *logger.Logger
}

func (s service) GetAll(ctx context.Context, page int, limit int) ([]ListItemResponse, int64, error) {
	offset := (page - 1) * limit

	servers, total, err := s.repository.FindAll(ctx, limit, offset)
	if err != nil {
		s.logger.ErrorWithStack(ctx, "failed to find servers", err,
			slog.Int("page", page),
			slog.Int("limit", limit),
		)
		return nil, 0, ErrListServers
	}

	responses := make([]ListItemResponse, 0, len(servers))
	for _, server := range servers {
		responses = append(responses, ListItemResponse{
			GetResponse: mapEntityToResponse(server.Server),
			Relations: RelationSummary{
				AgentCount:   server.AgentCount,
				JobCount:     server.JobCount,
				ReleaseCount: server.ReleaseCount,
			},
		})
	}

	return responses, total, nil
}

func (s service) GetById(ctx context.Context, serverID string) (GetWithRelationsResponse, error) {
	server, err := s.repository.FindById(ctx, serverID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return GetWithRelationsResponse{}, ErrServerNotFound
		}

		s.logger.ErrorWithStack(ctx, "failed to find server by id", err,
			slog.String("server_id", serverID),
		)
		return GetWithRelationsResponse{}, ErrGetServer
	}

	counts, err := s.repository.GetRelationCounts(ctx, server.ID)
	if err != nil {
		s.logger.ErrorWithStack(ctx, "failed to get server relation counts", err,
			slog.String("server_id", serverID),
		)
		return GetWithRelationsResponse{}, ErrGetServer
	}

	return GetWithRelationsResponse{
		GetResponse: mapEntityToResponse(server),
		Relations:   counts,
	}, nil
}

func (s service) Create(ctx context.Context, req CreateRequest) (GetResponse, error) {
	_, err := s.repository.FindByHostname(ctx, req.Hostname)
	if err == nil {
		return GetResponse{}, ErrServerAlreadyExists
	}

	if !errors.Is(err, gorm.ErrRecordNotFound) {
		s.logger.ErrorWithStack(ctx, "failed to check server existence before create", err,
			slog.String("hostname", req.Hostname),
		)
		return GetResponse{}, ErrCreateServer
	}

	metadata := normalizeMetadata(req.Metadata)

	server := Server{
		ID:              uuid.New(),
		Hostname:        req.Hostname,
		Metadata:        metadata,
		OperatingSystem: req.OperatingSystem,
		Hypervisor:      req.Hypervisor,
		Location:        req.Location,
	}

	created, err := s.repository.Create(ctx, server)
	if err != nil {
		if isUniqueViolation(err) {
			return GetResponse{}, ErrServerAlreadyExists
		}

		s.logger.ErrorWithStack(ctx, "failed to create server", err,
			slog.String("hostname", req.Hostname),
		)
		return GetResponse{}, ErrCreateServer
	}

	if err := s.attachAgents(ctx, created.ID, req.AgentIDs, req.Agents); err != nil {
		return GetResponse{}, err
	}

	return mapEntityToResponse(created), nil
}

func (s service) Update(ctx context.Context, serverID string, req UpdateRequest) (GetWithRelationsResponse, error) {
	if err := s.ensureServerExists(ctx, serverID); err != nil {
		return GetWithRelationsResponse{}, err
	}

	serverEntity, err := s.repository.FindById(ctx, serverID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return GetWithRelationsResponse{}, ErrServerNotFound
		}

		return GetWithRelationsResponse{}, ErrUpdateServer
	}

	if err := s.attachAgents(ctx, serverEntity.ID, req.AgentIDs, req.Agents); err != nil {
		return GetWithRelationsResponse{}, err
	}

	return s.GetById(ctx, serverID)
}

func (s service) attachAgents(ctx context.Context, serverID uuid.UUID, agentIDs []uuid.UUID, agents []AgentRegistrationInput) error {
	if len(agentIDs) > 0 {
		if err := s.agentAttachment.AttachAgentIDs(ctx, serverID, agentIDs); err != nil {
			return err
		}
	}

	if len(agents) > 0 {
		if err := s.agentAttachment.CreateAgentsOnServer(ctx, serverID, agents); err != nil {
			return err
		}
	}

	return nil
}

func (s service) ListJobs(ctx context.Context, serverID string, page int, limit int) ([]JobAssociationResponse, int64, error) {
	if err := s.ensureServerExists(ctx, serverID); err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * limit

	rows, total, err := s.repository.FindAssociatedJobs(ctx, serverID, limit, offset)
	if err != nil {
		s.logger.ErrorWithStack(ctx, "failed to list server jobs", err,
			slog.String("server_id", serverID),
		)
		return nil, 0, ErrListJobs
	}

	responses := make([]JobAssociationResponse, 0, len(rows))
	for _, row := range rows {
		responses = append(responses, mapJobAssociationRow(row))
	}

	return responses, total, nil
}

func (s service) AssociateJob(ctx context.Context, serverID string, req JobAssociateRequest) error {
	server, err := s.repository.FindById(ctx, serverID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrServerNotFound
		}

		s.logger.ErrorWithStack(ctx, "failed to find server before associating job", err,
			slog.String("server_id", serverID),
		)
		return ErrAssociateJob
	}

	exists, err := s.repository.JobExists(ctx, req.JobID, req.AutomationID)
	if err != nil {
		s.logger.ErrorWithStack(ctx, "failed to check job existence", err,
			slog.String("job_id", req.JobID),
			slog.String("automation_id", req.AutomationID.String()),
		)
		return ErrAssociateJob
	}

	if !exists {
		return ErrJobNotFound
	}

	associated, err := s.repository.JobAssociationExists(ctx, server.ID, req.JobID, req.AutomationID)
	if err != nil {
		s.logger.ErrorWithStack(ctx, "failed to check job association", err,
			slog.String("server_id", serverID),
		)
		return ErrAssociateJob
	}

	if associated {
		return ErrJobAlreadyAssociated
	}

	association := ServerJob{
		ServerID:     server.ID,
		JobID:        req.JobID,
		AutomationID: req.AutomationID,
	}

	if err := s.repository.CreateJobAssociation(ctx, association); err != nil {
		s.logger.ErrorWithStack(ctx, "failed to create job association", err,
			slog.String("server_id", serverID),
			slog.String("job_id", req.JobID),
		)
		return ErrAssociateJob
	}

	return nil
}

func (s service) DissociateJob(ctx context.Context, serverID string, jobID string, automationID uuid.UUID) error {
	server, err := s.repository.FindById(ctx, serverID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrServerNotFound
		}

		s.logger.ErrorWithStack(ctx, "failed to find server before dissociating job", err,
			slog.String("server_id", serverID),
		)
		return ErrDissociateJob
	}

	if err := s.repository.DeleteJobAssociation(ctx, server.ID, jobID, automationID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrJobNotFound
		}

		s.logger.ErrorWithStack(ctx, "failed to delete job association", err,
			slog.String("server_id", serverID),
			slog.String("job_id", jobID),
		)
		return ErrDissociateJob
	}

	return nil
}

func (s service) ListReleases(ctx context.Context, serverID string, page int, limit int) ([]ReleaseAssociationResponse, int64, error) {
	if err := s.ensureServerExists(ctx, serverID); err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * limit

	rows, total, err := s.repository.FindAssociatedReleases(ctx, serverID, limit, offset)
	if err != nil {
		s.logger.ErrorWithStack(ctx, "failed to list server releases", err,
			slog.String("server_id", serverID),
		)
		return nil, 0, ErrListReleases
	}

	responses := make([]ReleaseAssociationResponse, 0, len(rows))
	for _, row := range rows {
		responses = append(responses, mapReleaseAssociationRow(row))
	}

	return responses, total, nil
}

func (s service) AssociateRelease(ctx context.Context, serverID string, req ReleaseAssociateRequest) error {
	server, err := s.repository.FindById(ctx, serverID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrServerNotFound
		}

		s.logger.ErrorWithStack(ctx, "failed to find server before associating release", err,
			slog.String("server_id", serverID),
		)
		return ErrAssociateRelease
	}

	exists, err := s.repository.ReleaseExists(ctx, req.ReleaseID, req.ApplicationID)
	if err != nil {
		s.logger.ErrorWithStack(ctx, "failed to check release existence", err,
			slog.String("release_id", req.ReleaseID.String()),
		)
		return ErrAssociateRelease
	}

	if !exists {
		return ErrReleaseNotFound
	}

	associated, err := s.repository.ReleaseAssociationExists(ctx, server.ID, req.ReleaseID)
	if err != nil {
		s.logger.ErrorWithStack(ctx, "failed to check release association", err,
			slog.String("server_id", serverID),
		)
		return ErrAssociateRelease
	}

	if associated {
		return ErrReleaseAlreadyAssociated
	}

	association := ServerRelease{
		ServerID:      server.ID,
		ReleaseID:     req.ReleaseID,
		ApplicationID: req.ApplicationID,
	}

	if err := s.repository.CreateReleaseAssociation(ctx, association); err != nil {
		s.logger.ErrorWithStack(ctx, "failed to create release association", err,
			slog.String("server_id", serverID),
			slog.String("release_id", req.ReleaseID.String()),
		)
		return ErrAssociateRelease
	}

	return nil
}

func (s service) DissociateRelease(ctx context.Context, serverID string, releaseID uuid.UUID) error {
	server, err := s.repository.FindById(ctx, serverID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrServerNotFound
		}

		s.logger.ErrorWithStack(ctx, "failed to find server before dissociating release", err,
			slog.String("server_id", serverID),
		)
		return ErrDissociateRelease
	}

	if err := s.repository.DeleteReleaseAssociation(ctx, server.ID, releaseID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrReleaseNotFound
		}

		s.logger.ErrorWithStack(ctx, "failed to delete release association", err,
			slog.String("server_id", serverID),
			slog.String("release_id", releaseID.String()),
		)
		return ErrDissociateRelease
	}

	return nil
}

func (s service) ensureServerExists(ctx context.Context, serverID string) error {
	_, err := s.repository.FindById(ctx, serverID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrServerNotFound
		}

		s.logger.ErrorWithStack(ctx, "failed to find server", err,
			slog.String("server_id", serverID),
		)
		return ErrGetServer
	}

	return nil
}

func NewService(repository Repository, agentAttachment AgentAttachmentService, appLogger *logger.Logger) Service {
	if appLogger == nil {
		appLogger = logger.Default()
	}

	return &service{
		repository:      repository,
		agentAttachment: agentAttachment,
		logger:          appLogger,
	}
}

func mapEntityToResponse(server Server) GetResponse {
	metadata := json.RawMessage(server.Metadata)
	if len(metadata) == 0 {
		metadata = json.RawMessage(`{}`)
	}

	return GetResponse{
		ID:              server.ID,
		Hostname:        server.Hostname,
		Metadata:        metadata,
		OperatingSystem: server.OperatingSystem,
		Hypervisor:      server.Hypervisor,
		Location:        server.Location,
	}
}

func mapJobAssociationRow(row JobAssociationRow) JobAssociationResponse {
	return JobAssociationResponse(row)
}

func mapReleaseAssociationRow(row ReleaseAssociationRow) ReleaseAssociationResponse {
	return ReleaseAssociationResponse(row)
}

func normalizeMetadata(raw json.RawMessage) datatypes.JSON {
	if len(raw) == 0 {
		return datatypes.JSON([]byte(`{}`))
	}

	return datatypes.JSON(raw)
}
