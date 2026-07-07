package server

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/svetlyopet/heimdallr/internal/testutil"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type stubAgentAttachment struct {
	attachErr error
	createErr error
}

func (s stubAgentAttachment) AttachAgentIDs(context.Context, uuid.UUID, []uuid.UUID) error {
	return s.attachErr
}

func (s stubAgentAttachment) CreateAgentsOnServer(context.Context, uuid.UUID, []AgentRegistrationInput) error {
	return s.createErr
}

func uuidPtr(id uuid.UUID) *uuid.UUID {
	return &id
}

func newServerService(t *testing.T) (Service, *gorm.DB) {
	t.Helper()

	db := newServerTestDB(t)
	repo := NewRepository(db)
	return NewService(repo, stubAgentAttachment{}, nil), db
}

func TestServiceCreateReturnsServer(t *testing.T) {
	svc, _ := newServerService(t)

	created, err := svc.Create(context.Background(), CreateRequest{
		Hostname:        "new-host.example.com",
		OperatingSystem: "linux",
		Hypervisor:      "vmware",
		Location:        "dc1",
	})
	require.NoError(t, err)
	require.Equal(t, "new-host.example.com", created.Hostname)
}

func TestServiceCreateReturnsConflictForDuplicateHostname(t *testing.T) {
	svc, _ := newServerService(t)

	req := CreateRequest{Hostname: "dup-host.example.com"}
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

	created, err := svc.Create(context.Background(), CreateRequest{Hostname: "job-host.example.com"})
	require.NoError(t, err)

	err = svc.AssociateJob(context.Background(), created.ID.String(), JobAssociateRequest{
		JobID:        "missing",
		AutomationID: uuid.New(),
	})
	require.ErrorIs(t, err, ErrJobNotFound)
}

func TestServiceAssociateJobReturnsConflictForDuplicateAssociation(t *testing.T) {
	svc, db := newServerService(t)

	created, err := svc.Create(context.Background(), CreateRequest{Hostname: "assoc-host.example.com"})
	require.NoError(t, err)

	automationID := uuid.New()
	providerID := seedProvider(t, db)
	seedAutomation(t, db, automationID, providerID)
	seedJob(t, db, "job-42", automationID, providerID)

	req := JobAssociateRequest{JobID: "job-42", AutomationID: automationID}
	require.NoError(t, svc.AssociateJob(context.Background(), created.ID.String(), req))

	err = svc.AssociateJob(context.Background(), created.ID.String(), req)
	require.ErrorIs(t, err, ErrJobAlreadyAssociated)
}

func TestServiceAssociateReleaseReturnsNotFoundForMissingRelease(t *testing.T) {
	svc, _ := newServerService(t)

	created, err := svc.Create(context.Background(), CreateRequest{Hostname: "rel-host.example.com"})
	require.NoError(t, err)

	err = svc.AssociateRelease(context.Background(), created.ID.String(), ReleaseAssociateRequest{
		ReleaseID:     uuid.New(),
		ApplicationID: uuid.New(),
	})
	require.ErrorIs(t, err, ErrReleaseNotFound)
}

func TestServiceGetByIdReturnsRelationCounts(t *testing.T) {
	svc, db := newServerService(t)

	created, err := svc.Create(context.Background(), CreateRequest{Hostname: "counts-host.example.com"})
	require.NoError(t, err)

	require.NoError(t, db.Create(&testAgent{
		ID:       uuid.New(),
		ServerID: uuidPtr(created.ID),
		Server:   "counts-host.example.com",
		Name:     "agent-a",
		Metadata: datatypes.JSON([]byte(`{}`)),
	}).Error)

	got, err := svc.GetById(context.Background(), created.ID.String())
	require.NoError(t, err)
	require.Equal(t, int64(1), got.Relations.AgentCount)
}

type stubServerRepository struct {
	findAllResult []ServerWithCounts
	findAllTotal  int64
	findAllErr    error
	findById      Server
	findByIdErr   error
}

func (s stubServerRepository) FindAll(context.Context, int, int) ([]ServerWithCounts, int64, error) {
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

func TestServiceGetAllReturnsRepositoryError(t *testing.T) {
	svc := NewService(stubServerRepository{findAllErr: ErrListServers}, stubAgentAttachment{}, nil)

	_, _, err := svc.GetAll(context.Background(), 1, 10)
	require.ErrorIs(t, err, ErrListServers)
}

func TestServiceCreateUsesDefaultMetadata(t *testing.T) {
	db := testutil.NewSQLiteDB(t, &Server{})
	repo := NewRepository(db)
	svc := NewService(repo, stubAgentAttachment{}, nil)

	created, err := svc.Create(context.Background(), CreateRequest{Hostname: "meta-host.example.com"})
	require.NoError(t, err)
	require.JSONEq(t, `{}`, string(created.Metadata))
}

func TestServiceCreateStoresMetadata(t *testing.T) {
	db := testutil.NewSQLiteDB(t, &Server{})
	repo := NewRepository(db)
	svc := NewService(repo, stubAgentAttachment{}, nil)

	created, err := svc.Create(context.Background(), CreateRequest{
		Hostname: "meta-host-2.example.com",
		Metadata: json.RawMessage(`{"rack":"A1"}`),
	})
	require.NoError(t, err)
	require.JSONEq(t, `{"rack":"A1"}`, string(created.Metadata))
}

type recordingAgentAttachment struct {
	attachIDs []uuid.UUID
	agents    []AgentRegistrationInput
}

func (r *recordingAgentAttachment) AttachAgentIDs(_ context.Context, _ uuid.UUID, agentIDs []uuid.UUID) error {
	r.attachIDs = append(r.attachIDs, agentIDs...)
	return nil
}

func (r *recordingAgentAttachment) CreateAgentsOnServer(_ context.Context, _ uuid.UUID, agents []AgentRegistrationInput) error {
	r.agents = append(r.agents, agents...)
	return nil
}

func TestServiceCreateInvokesAgentAttachment(t *testing.T) {
	db := newServerTestDB(t)
	repo := NewRepository(db)
	attachment := &recordingAgentAttachment{}
	svc := NewService(repo, attachment, nil)

	orphanID := uuid.New()
	_, err := svc.Create(context.Background(), CreateRequest{
		Hostname: "attach-create.example.com",
		AgentIDs: []uuid.UUID{orphanID},
		Agents: []AgentRegistrationInput{{
			Name:    "crowdstrike",
			Type:    "security",
			Version: "1.0.0",
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
	svc := NewService(repo, attachment, nil)

	created, err := svc.Create(context.Background(), CreateRequest{Hostname: "update-host.example.com"})
	require.NoError(t, err)

	orphanID := uuid.New()
	_, err = svc.Update(context.Background(), created.ID.String(), UpdateRequest{
		AgentIDs: []uuid.UUID{orphanID},
	})
	require.NoError(t, err)
	require.Equal(t, []uuid.UUID{orphanID}, attachment.attachIDs)
}
