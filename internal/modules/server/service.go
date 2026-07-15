package server

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"

	"github.com/google/uuid"
	"github.com/svetlyopet/heimdallr/internal/database"
	"github.com/svetlyopet/heimdallr/internal/logger"
	"github.com/svetlyopet/heimdallr/internal/modules/server/api"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type LookupService interface {
	GetById(ctx context.Context, serverID string) (api.Server, error)
}

type Service interface {
	GetAll(ctx context.Context, agentID string, page int, limit int) ([]api.ServerListItem, int64, error)
	GetById(ctx context.Context, serverID string) (api.ServerWithRelations, error)
	Create(ctx context.Context, req api.ServerCreateRequest) (api.Server, error)
	Update(ctx context.Context, serverID string, req api.ServerUpdateRequest) (api.ServerWithRelations, error)

	ListJobs(ctx context.Context, serverID string, page int, limit int) ([]api.ServerJobAssociation, int64, error)
	AssociateJob(ctx context.Context, serverID string, req api.ServerJobAssociateRequest) error
	DissociateJob(ctx context.Context, serverID string, jobID string, automationID uuid.UUID) error

	ListReleases(ctx context.Context, serverID string, page int, limit int) ([]api.ServerReleaseAssociation, int64, error)
	AssociateRelease(ctx context.Context, serverID string, req api.ServerReleaseAssociateRequest) error
	DissociateRelease(ctx context.Context, serverID string, releaseID uuid.UUID) error
}

type service struct {
	repository      Repository
	agentAttachment AgentAttachmentService
	db              *gorm.DB
	logger          *logger.Logger
}

func (s service) GetAll(ctx context.Context, agentID string, page int, limit int) ([]api.ServerListItem, int64, error) {
	offset := (page - 1) * limit

	servers, total, err := s.repository.FindAll(ctx, agentID, limit, offset)
	if err != nil {
		s.logger.ErrorWithStack(ctx, "failed to find servers", err,
			slog.Int("page", page),
			slog.Int("limit", limit),
		)
		return nil, 0, ErrListServers
	}

	responses := make([]api.ServerListItem, 0, len(servers))
	for _, server := range servers {
		item, err := s.mapServerListItem(ctx, server)
		if err != nil {
			return nil, 0, err
		}
		responses = append(responses, item)
	}

	return responses, total, nil
}

func (s service) GetById(ctx context.Context, serverID string) (api.ServerWithRelations, error) {
	server, err := s.repository.FindById(ctx, serverID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return api.ServerWithRelations{}, ErrServerNotFound
		}

		s.logger.ErrorWithStack(ctx, "failed to find server by id", err,
			slog.String("server_id", serverID),
		)
		return api.ServerWithRelations{}, ErrGetServer
	}

	counts, err := s.repository.GetRelationCounts(ctx, server.ID)
	if err != nil {
		s.logger.ErrorWithStack(ctx, "failed to get server relation counts", err,
			slog.String("server_id", serverID),
		)
		return api.ServerWithRelations{}, ErrGetServer
	}

	return s.mapEntityToWithRelations(ctx, server, counts)
}

func (s service) Create(ctx context.Context, req api.ServerCreateRequest) (api.Server, error) {
	_, err := s.repository.FindByHostname(ctx, req.Hostname)
	if err == nil {
		return api.Server{}, ErrServerAlreadyExists
	}

	if !errors.Is(err, gorm.ErrRecordNotFound) {
		s.logger.ErrorWithStack(ctx, "failed to check server existence before create", err,
			slog.String("hostname", req.Hostname),
		)
		return api.Server{}, ErrCreateServer
	}

	agentIDs := uuidSliceValue(req.AgentIds)
	agents := agentCreateSliceValue(req.Agents)
	if err := validateAgentAttachments(agentIDs, agents); err != nil {
		return api.Server{}, err
	}

	metadata, err := metadataToEntity(req.Metadata)
	if err != nil {
		return api.Server{}, ErrCreateServer
	}

	serverEntity := Server{
		ID:              uuid.New(),
		Hostname:        req.Hostname,
		Metadata:        metadata,
		OperatingSystem: stringValue(req.OperatingSystem),
		Hypervisor:      stringValue(req.Hypervisor),
		Location:        stringValue(req.Location),
	}

	var created Server

	if err := database.WithTransaction(ctx, s.db, func(tx *gorm.DB) error {
		txRepo := s.repository.WithTx(tx)

		var createErr error
		created, createErr = txRepo.Create(ctx, serverEntity)
		if createErr != nil {
			if database.IsUniqueViolation(createErr) {
				return ErrServerAlreadyExists
			}

			return createErr
		}

		return s.attachAgents(ctx, created.ID, agentIDs, agents, tx)
	}); err != nil {
		if errors.Is(err, ErrServerAlreadyExists) ||
			errors.Is(err, ErrAgentAlreadyLinked) ||
			errors.Is(err, ErrAgentAlreadyExists) ||
			errors.Is(err, ErrDuplicateAgentIDs) ||
			errors.Is(err, ErrDuplicateAgentNames) {
			return api.Server{}, err
		}

		s.logger.ErrorWithStack(ctx, "failed to create server", err,
			slog.String("hostname", req.Hostname),
		)
		return api.Server{}, ErrCreateServer
	}

	return s.mapEntityToResponse(ctx, created)
}

func (s service) Update(ctx context.Context, serverID string, req api.ServerUpdateRequest) (api.ServerWithRelations, error) {
	if err := s.ensureServerExists(ctx, serverID); err != nil {
		return api.ServerWithRelations{}, err
	}

	serverEntity, err := s.repository.FindById(ctx, serverID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return api.ServerWithRelations{}, ErrServerNotFound
		}

		return api.ServerWithRelations{}, ErrUpdateServer
	}

	agentIDs := uuidSliceValue(req.AgentIds)
	agents := agentCreateSliceValue(req.Agents)
	if err := validateAgentAttachments(agentIDs, agents); err != nil {
		return api.ServerWithRelations{}, err
	}

	if len(agentIDs) > 0 || len(agents) > 0 {
		if err := database.WithTransaction(ctx, s.db, func(tx *gorm.DB) error {
			return s.attachAgents(ctx, serverEntity.ID, agentIDs, agents, tx)
		}); err != nil {
			if errors.Is(err, ErrAgentAlreadyLinked) ||
				errors.Is(err, ErrAgentAlreadyExists) ||
				errors.Is(err, ErrDuplicateAgentIDs) ||
				errors.Is(err, ErrDuplicateAgentNames) {
				return api.ServerWithRelations{}, err
			}

			s.logger.ErrorWithStack(ctx, "failed to update server agents", err,
				slog.String("server_id", serverID),
			)
			return api.ServerWithRelations{}, ErrUpdateServer
		}
	}

	return s.GetById(ctx, serverID)
}

func (s service) attachAgents(ctx context.Context, serverID uuid.UUID, agentIDs []uuid.UUID, agents []api.AgentCreateRequest, tx *gorm.DB) error {
	if len(agentIDs) > 0 {
		if err := s.agentAttachment.AttachAgentIDs(ctx, serverID, agentIDs, tx); err != nil {
			return err
		}
	}

	if len(agents) > 0 {
		if err := s.agentAttachment.CreateAgentsOnServer(ctx, serverID, agents, tx); err != nil {
			return err
		}
	}

	return nil
}

func validateAgentAttachments(agentIDs []uuid.UUID, agents []api.AgentCreateRequest) error {
	seenIDs := make(map[uuid.UUID]struct{}, len(agentIDs))
	for _, agentID := range agentIDs {
		if _, exists := seenIDs[agentID]; exists {
			return ErrDuplicateAgentIDs
		}

		seenIDs[agentID] = struct{}{}
	}

	seenNames := make(map[string]struct{}, len(agents))
	for _, agent := range agents {
		if _, exists := seenNames[agent.Name]; exists {
			return ErrDuplicateAgentNames
		}

		seenNames[agent.Name] = struct{}{}
	}

	return nil
}

func (s service) ListJobs(ctx context.Context, serverID string, page int, limit int) ([]api.ServerJobAssociation, int64, error) {
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

	responses := make([]api.ServerJobAssociation, 0, len(rows))
	for _, row := range rows {
		responses = append(responses, mapJobAssociationRow(row))
	}

	return responses, total, nil
}

func (s service) AssociateJob(ctx context.Context, serverID string, req api.ServerJobAssociateRequest) error {
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

	exists, err := s.repository.JobExists(ctx, req.JobId, req.AutomationId)
	if err != nil {
		s.logger.ErrorWithStack(ctx, "failed to check job existence", err,
			slog.String("job_id", req.JobId),
			slog.String("automation_id", req.AutomationId.String()),
		)
		return ErrAssociateJob
	}

	if !exists {
		return ErrJobNotFound
	}

	associated, err := s.repository.JobAssociationExists(ctx, server.ID, req.JobId, req.AutomationId)
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
		JobID:        req.JobId,
		AutomationID: req.AutomationId,
	}

	if err := s.repository.CreateJobAssociation(ctx, association); err != nil {
		s.logger.ErrorWithStack(ctx, "failed to create job association", err,
			slog.String("server_id", serverID),
			slog.String("job_id", req.JobId),
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

func (s service) ListReleases(ctx context.Context, serverID string, page int, limit int) ([]api.ServerReleaseAssociation, int64, error) {
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

	responses := make([]api.ServerReleaseAssociation, 0, len(rows))
	for _, row := range rows {
		responses = append(responses, mapReleaseAssociationRow(row))
	}

	return responses, total, nil
}

func (s service) AssociateRelease(ctx context.Context, serverID string, req api.ServerReleaseAssociateRequest) error {
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

	exists, err := s.repository.ReleaseExists(ctx, req.ReleaseId, req.ApplicationId)
	if err != nil {
		s.logger.ErrorWithStack(ctx, "failed to check release existence", err,
			slog.String("release_id", req.ReleaseId.String()),
		)
		return ErrAssociateRelease
	}

	if !exists {
		return ErrReleaseNotFound
	}

	associated, err := s.repository.ReleaseAssociationExists(ctx, server.ID, req.ReleaseId)
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
		ReleaseID:     req.ReleaseId,
		ApplicationID: req.ApplicationId,
	}

	if err := s.repository.CreateReleaseAssociation(ctx, association); err != nil {
		s.logger.ErrorWithStack(ctx, "failed to create release association", err,
			slog.String("server_id", serverID),
			slog.String("release_id", req.ReleaseId.String()),
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

func NewService(repository Repository, agentAttachment AgentAttachmentService, db *gorm.DB, appLogger *logger.Logger) Service {
	if appLogger == nil {
		appLogger = logger.Default()
	}

	return &service{
		repository:      repository,
		agentAttachment: agentAttachment,
		db:              db,
		logger:          appLogger,
	}
}

func mapEntityToResponse(ctx context.Context, server Server, logger *logger.Logger) (api.Server, error) {
	metadata, err := metadataFromEntity(server.Metadata)
	if err != nil {
		logger.ErrorWithStack(ctx, "failed to decode stored server metadata", err,
			slog.String("entity_type", "server"),
			slog.String("server_id", server.ID.String()),
		)
		return api.Server{}, ErrGetServer
	}

	return api.Server{
		Id:              server.ID,
		Hostname:        server.Hostname,
		Metadata:        metadata,
		OperatingSystem: server.OperatingSystem,
		Hypervisor:      server.Hypervisor,
		Location:        server.Location,
	}, nil
}

func (s service) mapEntityToResponse(ctx context.Context, server Server) (api.Server, error) {
	return mapEntityToResponse(ctx, server, s.logger)
}

func mapEntityToWithRelations(ctx context.Context, server Server, counts RelationSummary, logger *logger.Logger) (api.ServerWithRelations, error) {
	metadata, err := metadataFromEntity(server.Metadata)
	if err != nil {
		logger.ErrorWithStack(ctx, "failed to decode stored server metadata", err,
			slog.String("entity_type", "server"),
			slog.String("server_id", server.ID.String()),
		)
		return api.ServerWithRelations{}, ErrGetServer
	}

	return api.ServerWithRelations{
		Id:              server.ID,
		Hostname:        server.Hostname,
		Metadata:        metadata,
		OperatingSystem: server.OperatingSystem,
		Hypervisor:      server.Hypervisor,
		Location:        server.Location,
		Relations: api.ServerRelationSummary{
			AgentCount:   int(counts.AgentCount),
			JobCount:     int(counts.JobCount),
			ReleaseCount: int(counts.ReleaseCount),
		},
	}, nil
}

func (s service) mapEntityToWithRelations(ctx context.Context, server Server, counts RelationSummary) (api.ServerWithRelations, error) {
	return mapEntityToWithRelations(ctx, server, counts, s.logger)
}

func (s service) mapServerListItem(ctx context.Context, server ServerWithCounts) (api.ServerListItem, error) {
	metadata, err := metadataFromEntity(server.Metadata)
	if err != nil {
		s.logger.ErrorWithStack(ctx, "failed to decode stored server metadata", err,
			slog.String("entity_type", "server"),
			slog.String("server_id", server.ID.String()),
		)
		return api.ServerListItem{}, ErrListServers
	}

	return api.ServerListItem{
		Id:              server.ID,
		Hostname:        server.Hostname,
		Metadata:        metadata,
		OperatingSystem: server.OperatingSystem,
		Hypervisor:      server.Hypervisor,
		Location:        server.Location,
		Compliant:       server.Compliant,
		Relations: api.ServerRelationSummary{
			AgentCount:   int(server.AgentCount),
			JobCount:     int(server.JobCount),
			ReleaseCount: int(server.ReleaseCount),
		},
	}, nil
}

func mapJobAssociationRow(row JobAssociationRow) api.ServerJobAssociation {
	return api.ServerJobAssociation{
		JobId:        row.JobID,
		AutomationId: row.AutomationID,
		Automation:   row.Automation,
		Provider:     row.Provider,
		Status:       api.JobStatus(row.Status),
		Location:     row.Location,
		Url:          row.URL,
	}
}

func mapReleaseAssociationRow(row ReleaseAssociationRow) api.ServerReleaseAssociation {
	return api.ServerReleaseAssociation{
		ReleaseId:     row.ReleaseID,
		ApplicationId: row.ApplicationID,
		Application:   row.Application,
		Version:       row.Version,
		CommitSha:     row.CommitSHA,
		Branch:        row.Branch,
	}
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

func uuidSliceValue(values *[]uuid.UUID) []uuid.UUID {
	if values == nil {
		return nil
	}

	return *values
}

func agentCreateSliceValue(values *[]api.AgentCreateRequest) []api.AgentCreateRequest {
	if values == nil {
		return nil
	}

	return *values
}
