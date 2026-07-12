package server

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/svetlyopet/heimdallr/internal/server/api"
	"github.com/svetlyopet/heimdallr/internal/testutil"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type stubAgentAttachment struct {
	attachErr error
	createErr error
}

func (s stubAgentAttachment) AttachAgentIDs(context.Context, uuid.UUID, []uuid.UUID, *gorm.DB) error {
	return s.attachErr
}

func (s stubAgentAttachment) CreateAgentsOnServer(context.Context, uuid.UUID, []api.AgentCreateRequest, *gorm.DB) error {
	return s.createErr
}

func strPtr(value string) *string {
	return &value
}

func newServerService(t *testing.T) (Service, *gorm.DB) {
	t.Helper()

	db := newServerTestDB(t)
	repo := NewRepository(db)
	return NewService(repo, stubAgentAttachment{}, db, nil), db
}

func TestServiceCreateReturnsServer(t *testing.T) {
	svc, _ := newServerService(t)

	created, err := svc.Create(context.Background(), api.ServerCreateRequest{
		Hostname:        "new-host.example.com",
		OperatingSystem: strPtr("linux"),
		Hypervisor:      strPtr("vmware"),
		Location:        strPtr("dc1"),
	})
	require.NoError(t, err)
	require.Equal(t, "new-host.example.com", created.Hostname)
}

func TestServiceCreateReturnsConflictForDuplicateHostname(t *testing.T) {
	svc, _ := newServerService(t)

	req := api.ServerCreateRequest{Hostname: "dup-host.example.com"}
	_, err := svc.Create(context.Background(), req)
	require.NoError(t, err)

	_, err = svc.Create(context.Background(), req)
	require.ErrorIs(t, err, ErrServerAlreadyExists)
}

func TestServiceGetByIdReturnsNotFound(t *testing.T) {
	svc, _ := newServerService(t)

	_, err := svc.GetById(context.Background(), uuid.New().String())
	require.ErrorIs(t, err, ErrServerNotFound)
}

func TestServiceAssociateJobReturnsNotFoundForMissingJob(t *testing.T) {
	svc, _ := newServerService(t)

	created, err := svc.Create(context.Background(), api.ServerCreateRequest{Hostname: "job-host.example.com"})
	require.NoError(t, err)

	err = svc.AssociateJob(context.Background(), created.Id.String(), api.ServerJobAssociateRequest{
		JobId:        "missing",
		AutomationId: uuid.New(),
	})
	require.ErrorIs(t, err, ErrJobNotFound)
}

func TestServiceAssociateJobReturnsConflictForDuplicateAssociation(t *testing.T) {
	svc, db := newServerService(t)

	created, err := svc.Create(context.Background(), api.ServerCreateRequest{Hostname: "assoc-host.example.com"})
	require.NoError(t, err)

	automationID := uuid.New()
	providerID := seedProvider(t, db)
	seedAutomation(t, db, automationID, providerID)
	seedJob(t, db, "job-42", automationID, providerID)

	req := api.ServerJobAssociateRequest{JobId: "job-42", AutomationId: automationID}
	require.NoError(t, svc.AssociateJob(context.Background(), created.Id.String(), req))

	err = svc.AssociateJob(context.Background(), created.Id.String(), req)
	require.ErrorIs(t, err, ErrJobAlreadyAssociated)
}

func TestServiceAssociateReleaseReturnsNotFoundForMissingRelease(t *testing.T) {
	svc, _ := newServerService(t)

	created, err := svc.Create(context.Background(), api.ServerCreateRequest{Hostname: "rel-host.example.com"})
	require.NoError(t, err)

	err = svc.AssociateRelease(context.Background(), created.Id.String(), api.ServerReleaseAssociateRequest{
		ReleaseId:     uuid.New(),
		ApplicationId: uuid.New(),
	})
	require.ErrorIs(t, err, ErrReleaseNotFound)
}

func TestServiceGetByIdReturnsRelationCounts(t *testing.T) {
	svc, db := newServerService(t)

	created, err := svc.Create(context.Background(), api.ServerCreateRequest{Hostname: "counts-host.example.com"})
	require.NoError(t, err)

	agentID := uuid.New()
	require.NoError(t, db.Create(&testAgent{
		ID:       agentID,
		Name:     "agent-a",
		Metadata: datatypes.JSON([]byte(`{}`)),
	}).Error)
	require.NoError(t, db.Create(&testServerAgent{
		ServerID: created.Id,
		AgentID:  agentID,
	}).Error)

	got, err := svc.GetById(context.Background(), created.Id.String())
	require.NoError(t, err)
	require.Equal(t, 1, got.Relations.AgentCount)
}

type stubServerRepository struct {
	findAllResult []ServerWithCounts
	findAllTotal  int64
	findAllErr    error
	findById      Server
	findByIdErr   error
}

func (s stubServerRepository) FindAll(context.Context, string, int, int) ([]ServerWithCounts, int64, error) {
	return s.findAllResult, s.findAllTotal, s.findAllErr
}

func (s stubServerRepository) FindById(context.Context, string) (Server, error) {
	return s.findById, s.findByIdErr
}

func (s stubServerRepository) FindByHostname(context.Context, string) (Server, error) {
	return Server{}, gorm.ErrRecordNotFound
}

func (s stubServerRepository) Create(context.Context, Server) (Server, error) {
	return Server{}, nil
}

func (s stubServerRepository) GetRelationCounts(context.Context, uuid.UUID) (RelationSummary, error) {
	return RelationSummary{}, nil
}

func (s stubServerRepository) FindAssociatedJobs(context.Context, string, int, int) ([]JobAssociationRow, int64, error) {
	return nil, 0, nil
}

func (s stubServerRepository) JobExists(context.Context, string, uuid.UUID) (bool, error) {
	return false, nil
}

func (s stubServerRepository) JobAssociationExists(context.Context, uuid.UUID, string, uuid.UUID) (bool, error) {
	return false, nil
}

func (s stubServerRepository) CreateJobAssociation(context.Context, ServerJob) error {
	return nil
}

func (s stubServerRepository) DeleteJobAssociation(context.Context, uuid.UUID, string, uuid.UUID) error {
	return nil
}

func (s stubServerRepository) FindAssociatedReleases(context.Context, string, int, int) ([]ReleaseAssociationRow, int64, error) {
	return nil, 0, nil
}

func (s stubServerRepository) ReleaseExists(context.Context, uuid.UUID, uuid.UUID) (bool, error) {
	return false, nil
}

func (s stubServerRepository) ReleaseAssociationExists(context.Context, uuid.UUID, uuid.UUID) (bool, error) {
	return false, nil
}

func (s stubServerRepository) CreateReleaseAssociation(context.Context, ServerRelease) error {
	return nil
}

func (s stubServerRepository) DeleteReleaseAssociation(context.Context, uuid.UUID, uuid.UUID) error {
	return nil
}

func (s stubServerRepository) WithTx(*gorm.DB) Repository {
	return s
}

func TestServiceGetAllReturnsRepositoryError(t *testing.T) {
	db := testutil.NewSQLiteDB(t, &Server{})
	svc := NewService(stubServerRepository{findAllErr: ErrListServers}, stubAgentAttachment{}, db, nil)

	_, _, err := svc.GetAll(context.Background(), "", 1, 10)
	require.ErrorIs(t, err, ErrListServers)
}

func TestServiceCreateUsesDefaultMetadata(t *testing.T) {
	db := testutil.NewSQLiteDB(t, &Server{})
	repo := NewRepository(db)
	svc := NewService(repo, stubAgentAttachment{}, db, nil)

	created, err := svc.Create(context.Background(), api.ServerCreateRequest{Hostname: "meta-host.example.com"})
	require.NoError(t, err)
	require.Empty(t, created.Metadata)
}

func TestServiceCreateStoresMetadata(t *testing.T) {
	db := testutil.NewSQLiteDB(t, &Server{})
	repo := NewRepository(db)
	svc := NewService(repo, stubAgentAttachment{}, db, nil)

	metadata := api.ServerMetadata{"rack": "A1"}
	created, err := svc.Create(context.Background(), api.ServerCreateRequest{
		Hostname: "meta-host-2.example.com",
		Metadata: &metadata,
	})
	require.NoError(t, err)
	require.Equal(t, "A1", created.Metadata["rack"])
}

type recordingAgentAttachment struct {
	attachIDs []uuid.UUID
	agents    []api.AgentCreateRequest
}

func (r *recordingAgentAttachment) AttachAgentIDs(_ context.Context, _ uuid.UUID, agentIDs []uuid.UUID, _ *gorm.DB) error {
	r.attachIDs = append(r.attachIDs, agentIDs...)
	return nil
}

func (r *recordingAgentAttachment) CreateAgentsOnServer(_ context.Context, _ uuid.UUID, agents []api.AgentCreateRequest, _ *gorm.DB) error {
	r.agents = append(r.agents, agents...)
	return nil
}

func TestServiceCreateInvokesAgentAttachment(t *testing.T) {
	db := newServerTestDB(t)
	repo := NewRepository(db)
	attachment := &recordingAgentAttachment{}
	svc := NewService(repo, attachment, db, nil)

	orphanID := uuid.New()
	agentType := "security"
	agentVersion := "1.0.0"
	_, err := svc.Create(context.Background(), api.ServerCreateRequest{
		Hostname: "attach-create.example.com",
		AgentIds: &[]uuid.UUID{orphanID},
		Agents: &[]api.AgentCreateRequest{{
			Name:    "crowdstrike",
			Type:    &agentType,
			Version: &agentVersion,
		}},
	})
	require.NoError(t, err)
	require.Equal(t, []uuid.UUID{orphanID}, attachment.attachIDs)
	require.Len(t, attachment.agents, 1)
}

func TestServiceUpdateInvokesAgentAttachment(t *testing.T) {
	db := newServerTestDB(t)
	repo := NewRepository(db)
	attachment := &recordingAgentAttachment{}
	svc := NewService(repo, attachment, db, nil)

	created, err := svc.Create(context.Background(), api.ServerCreateRequest{Hostname: "update-host.example.com"})
	require.NoError(t, err)

	orphanID := uuid.New()
	_, err = svc.Update(context.Background(), created.Id.String(), api.ServerUpdateRequest{
		AgentIds: &[]uuid.UUID{orphanID},
	})
	require.NoError(t, err)
	require.Equal(t, []uuid.UUID{orphanID}, attachment.attachIDs)
}
