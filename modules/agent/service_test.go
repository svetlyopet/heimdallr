package agent

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/svetlyopet/heimdallr/internal/server"
	serverapi "github.com/svetlyopet/heimdallr/internal/server/api"
	"github.com/svetlyopet/heimdallr/internal/testutil"
	"github.com/svetlyopet/heimdallr/modules/agent/api"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type stubServerLookup struct {
	resp serverapi.Server
	err  error
}

func (s stubServerLookup) GetById(_ context.Context, _ string) (serverapi.Server, error) {
	return s.resp, s.err
}

func newAgentService(t *testing.T) (Service, *gorm.DB) {
	t.Helper()

	db := newAgentTestDB(t)
	repo := NewRepository(db)
	lookup := stubServerLookup{}
	return NewService(repo, lookup, db, nil), db
}

func TestServiceCreateOnServerReturnsAgent(t *testing.T) {
	db := newAgentTestDB(t)
	repo := NewRepository(db)

	serverID := uuid.New()
	require.NoError(t, db.Create(&server.Server{
		ID:       serverID,
		Hostname: "agent-host.example.com",
		Metadata: datatypes.JSON([]byte(`{}`)),
	}).Error)

	svc := NewService(repo, stubServerLookup{
		resp: serverapi.Server{Id: serverID, Hostname: "agent-host.example.com"},
	}, db, nil)

	name := "datadog"
	agentType := "monitoring"
	agentVersion := "7.0.0"
	created, err := svc.CreateOnServer(context.Background(), serverID.String(), api.ServerAgentRequest{
		Name:    &name,
		Type:    &agentType,
		Version: &agentVersion,
	})
	require.NoError(t, err)
	require.Equal(t, "datadog", created.Name)
	require.Equal(t, 1, created.ServerCount)
}

func TestServiceCreateOnServerReturnsServerNotFound(t *testing.T) {
	svc := NewService(stubAgentRepository{}, stubServerLookup{err: server.ErrServerNotFound}, testutil.NewPostgresDB(t), nil)

	name := "datadog"
	_, err := svc.CreateOnServer(context.Background(), uuid.New().String(), api.ServerAgentRequest{Name: &name})
	require.ErrorIs(t, err, server.ErrServerNotFound)
}

func TestServiceGetByIdReturnsNotFound(t *testing.T) {
	svc, _ := newAgentService(t)

	_, err := svc.GetById(context.Background(), uuid.New().String(), uuid.New().String())
	require.ErrorIs(t, err, ErrAgentNotFound)
}

func TestServiceDetachReturnsNotFound(t *testing.T) {
	serverID := uuid.New()
	db := testutil.NewPostgresDB(t)
	svc := NewService(stubAgentRepository{}, stubServerLookup{
		resp: serverapi.Server{Id: serverID},
	}, db, nil)

	err := svc.Detach(context.Background(), serverID.String(), uuid.New().String())
	require.ErrorIs(t, err, ErrAgentNotFound)
}

func TestServiceGetAllReturnsServerNotFound(t *testing.T) {
	svc := NewService(stubAgentRepository{}, stubServerLookup{err: server.ErrServerNotFound}, testutil.NewPostgresDB(t), nil)

	_, _, err := svc.GetAll(context.Background(), uuid.New().String(), 1, 10)
	require.ErrorIs(t, err, server.ErrServerNotFound)
}

func TestServiceCreateOnServerUsesDefaultMetadata(t *testing.T) {
	db := newAgentTestDB(t)
	repo := NewRepository(db)

	serverID := uuid.New()
	require.NoError(t, db.Create(&server.Server{
		ID:       serverID,
		Hostname: "meta-host.example.com",
		Metadata: datatypes.JSON([]byte(`{}`)),
	}).Error)

	svc := NewService(repo, stubServerLookup{
		resp: serverapi.Server{Id: serverID, Hostname: "meta-host.example.com"},
	}, db, nil)

	name := "agent"
	created, err := svc.CreateOnServer(context.Background(), serverID.String(), api.ServerAgentRequest{Name: &name})
	require.NoError(t, err)
	require.Empty(t, created.Metadata)
}

func TestServiceCreateOnServerStoresMetadata(t *testing.T) {
	db := newAgentTestDB(t)
	repo := NewRepository(db)

	serverID := uuid.New()
	require.NoError(t, db.Create(&server.Server{
		ID:       serverID,
		Hostname: "meta-host-2.example.com",
		Metadata: datatypes.JSON([]byte(`{}`)),
	}).Error)

	svc := NewService(repo, stubServerLookup{
		resp: serverapi.Server{Id: serverID, Hostname: "meta-host-2.example.com"},
	}, db, nil)

	name := "agent"
	metadata := api.ServerMetadata{"env": "prod"}
	created, err := svc.CreateOnServer(context.Background(), serverID.String(), api.ServerAgentRequest{
		Name:     &name,
		Metadata: &metadata,
	})
	require.NoError(t, err)
	require.Equal(t, "prod", created.Metadata["env"])
}

type stubAgentRepository struct {
	findAllErr error
	findById   Agent
}

func (s stubAgentRepository) FindAll(context.Context, string, int, int) ([]Agent, int64, error) {
	return nil, 0, s.findAllErr
}

func (s stubAgentRepository) FindAllGlobal(context.Context, ListFilters, int, int) ([]AgentWithCount, int64, error) {
	return nil, 0, s.findAllErr
}

func (s stubAgentRepository) FindById(_ context.Context, _ string, _ string) (Agent, error) {
	if s.findById.ID != uuid.Nil {
		return s.findById, nil
	}

	return Agent{}, gorm.ErrRecordNotFound
}

func (s stubAgentRepository) FindByIdGlobal(context.Context, string) (AgentWithCount, error) {
	return AgentWithCount{}, gorm.ErrRecordNotFound
}

func (s stubAgentRepository) FindByName(context.Context, string) (Agent, error) {
	return Agent{}, gorm.ErrRecordNotFound
}

func (s stubAgentRepository) FindServersByAgent(context.Context, string, int, int) ([]LinkedServer, int64, error) {
	return nil, 0, gorm.ErrRecordNotFound
}

func (s stubAgentRepository) CreateUnassigned(context.Context, Agent) (Agent, error) {
	return Agent{}, nil
}

func (s stubAgentRepository) CreateOnServer(context.Context, uuid.UUID, Agent) (Agent, error) {
	return Agent{}, nil
}

func (s stubAgentRepository) AttachToServer(context.Context, uuid.UUID, []uuid.UUID) error {
	return nil
}

func (s stubAgentRepository) DetachFromServer(context.Context, string, string) error {
	return gorm.ErrRecordNotFound
}

func (s stubAgentRepository) DeleteGlobal(context.Context, string) error {
	return gorm.ErrRecordNotFound
}

func (s stubAgentRepository) WithTx(*gorm.DB) Repository {
	return s
}

func TestServiceGetAllReturnsRepositoryError(t *testing.T) {
	svc := NewService(
		stubAgentRepository{findAllErr: errors.New("db down")},
		stubServerLookup{resp: serverapi.Server{Id: uuid.New()}},
		testutil.NewPostgresDB(t),
		nil,
	)

	_, _, err := svc.GetAll(context.Background(), uuid.New().String(), 1, 10)
	require.ErrorIs(t, err, ErrListAgents)
}

func TestServiceGetByIdReturnsAgent(t *testing.T) {
	db := newAgentTestDB(t)
	repo := NewRepository(db)

	serverID := uuid.New()
	require.NoError(t, db.Create(&server.Server{
		ID:       serverID,
		Hostname: "get-host.example.com",
		Metadata: datatypes.JSON([]byte(`{}`)),
	}).Error)

	svc := NewService(repo, stubServerLookup{
		resp: serverapi.Server{Id: serverID},
	}, db, nil)

	name := "falcon"
	created, err := svc.CreateOnServer(context.Background(), serverID.String(), api.ServerAgentRequest{Name: &name})
	require.NoError(t, err)

	got, err := svc.GetById(context.Background(), created.Id.String(), serverID.String())
	require.NoError(t, err)
	require.Equal(t, "falcon", got.Name)
}

func TestServiceCreateOnServerReturnsInvalidServerID(t *testing.T) {
	svc := NewService(stubAgentRepository{}, stubServerLookup{}, testutil.NewPostgresDB(t), nil)

	name := "agent"
	_, err := svc.CreateOnServer(context.Background(), "not-a-uuid", api.ServerAgentRequest{Name: &name})
	require.ErrorIs(t, err, ErrInvalidServerID)
}

func TestServiceGetAllReturnsInvalidServerID(t *testing.T) {
	svc := NewService(stubAgentRepository{}, stubServerLookup{}, testutil.NewPostgresDB(t), nil)

	_, _, err := svc.GetAll(context.Background(), "bad-id", 1, 10)
	require.ErrorIs(t, err, ErrInvalidServerID)
}

var _ server.LookupService = stubServerLookup{}

func TestServiceCreateUnassigned(t *testing.T) {
	svc, _ := newAgentService(t)

	created, err := svc.CreateUnassigned(context.Background(), api.AgentCreateRequest{Name: "orphan"})
	require.NoError(t, err)
	require.Equal(t, 0, created.ServerCount)
	require.Equal(t, "orphan", created.Name)
}

func TestServiceCreateUnassignedReturnsAlreadyExists(t *testing.T) {
	svc, _ := newAgentService(t)

	_, err := svc.CreateUnassigned(context.Background(), api.AgentCreateRequest{Name: "duplicate"})
	require.NoError(t, err)

	_, err = svc.CreateUnassigned(context.Background(), api.AgentCreateRequest{Name: "duplicate"})
	require.ErrorIs(t, err, ErrAgentAlreadyExists)
}

func TestServiceCreateOnServerReturnsAlreadyExists(t *testing.T) {
	db := newAgentTestDB(t)
	repo := NewRepository(db)

	serverID := uuid.New()
	require.NoError(t, db.Create(&server.Server{
		ID:       serverID,
		Hostname: "dup-host.example.com",
		Metadata: datatypes.JSON([]byte(`{}`)),
	}).Error)

	svc := NewService(repo, stubServerLookup{
		resp: serverapi.Server{Id: serverID},
	}, db, nil)

	falcon := "falcon"
	_, err := svc.CreateOnServer(context.Background(), serverID.String(), api.ServerAgentRequest{Name: &falcon})
	require.NoError(t, err)

	_, err = svc.CreateOnServer(context.Background(), serverID.String(), api.ServerAgentRequest{Name: &falcon})
	require.ErrorIs(t, err, ErrAgentAlreadyExists)
}

func TestServiceListGlobalUnassignedOnly(t *testing.T) {
	db := newAgentTestDB(t)
	repo := NewRepository(db)
	svc := NewService(repo, stubServerLookup{}, db, nil)

	_, err := svc.CreateUnassigned(context.Background(), api.AgentCreateRequest{Name: "orphan"})
	require.NoError(t, err)

	agents, total, err := svc.ListGlobal(context.Background(), ListFilters{UnassignedOnly: true}, 1, 10)
	require.NoError(t, err)
	require.Equal(t, int64(1), total)
	require.Len(t, agents, 1)
}
