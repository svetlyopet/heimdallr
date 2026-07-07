package agent

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/svetlyopet/heimdallr/internal/server"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type stubServerLookup struct {
	resp server.GetResponse
	err  error
}

func (s stubServerLookup) GetById(_ context.Context, _ string) (server.GetResponse, error) {
	return s.resp, s.err
}

func newAgentService(t *testing.T) (Service, *gorm.DB) {
	t.Helper()

	db := newAgentTestDB(t)
	repo := NewRepository(db)
	lookup := stubServerLookup{}
	return NewService(repo, lookup, nil), db
}

func TestServiceCreateReturnsAgent(t *testing.T) {
	db := newAgentTestDB(t)
	repo := NewRepository(db)

	serverID := uuid.New()
	require.NoError(t, db.Create(&server.Server{
		ID:       serverID,
		Hostname: "agent-host.example.com",
		Metadata: datatypes.JSON([]byte(`{}`)),
	}).Error)

	svc := NewService(repo, stubServerLookup{
		resp: server.GetResponse{ID: serverID, Hostname: "agent-host.example.com"},
	}, nil)

	created, err := svc.Create(context.Background(), serverID.String(), CreateRequest{
		Name:    "datadog",
		Type:    "monitoring",
		Version: "7.0.0",
	})
	require.NoError(t, err)
	require.Equal(t, "datadog", created.Name)
	require.Equal(t, "agent-host.example.com", created.Server)
}

func TestServiceCreateReturnsServerNotFound(t *testing.T) {
	svc := NewService(stubAgentRepository{}, stubServerLookup{err: server.ErrServerNotFound}, nil)

	_, err := svc.Create(context.Background(), uuid.New().String(), CreateRequest{Name: "datadog"})
	require.ErrorIs(t, err, server.ErrServerNotFound)
}

func TestServiceGetByIdReturnsNotFound(t *testing.T) {
	svc, _ := newAgentService(t)

	_, err := svc.GetById(context.Background(), uuid.New().String(), uuid.New().String())
	require.ErrorIs(t, err, ErrAgentNotFound)
}

func TestServiceDeleteReturnsNotFound(t *testing.T) {
	serverID := uuid.New()
	svc := NewService(stubAgentRepository{}, stubServerLookup{
		resp: server.GetResponse{ID: serverID},
	}, nil)

	err := svc.Delete(context.Background(), serverID.String(), uuid.New().String())
	require.ErrorIs(t, err, ErrAgentNotFound)
}

func TestServiceGetAllReturnsServerNotFound(t *testing.T) {
	svc := NewService(stubAgentRepository{}, stubServerLookup{err: server.ErrServerNotFound}, nil)

	_, _, err := svc.GetAll(context.Background(), uuid.New().String(), 1, 10)
	require.ErrorIs(t, err, server.ErrServerNotFound)
}

func TestServiceCreateUsesDefaultMetadata(t *testing.T) {
	db := newAgentTestDB(t)
	repo := NewRepository(db)

	serverID := uuid.New()
	require.NoError(t, db.Create(&server.Server{
		ID:       serverID,
		Hostname: "meta-host.example.com",
		Metadata: datatypes.JSON([]byte(`{}`)),
	}).Error)

	svc := NewService(repo, stubServerLookup{
		resp: server.GetResponse{ID: serverID, Hostname: "meta-host.example.com"},
	}, nil)

	created, err := svc.Create(context.Background(), serverID.String(), CreateRequest{Name: "agent"})
	require.NoError(t, err)
	require.JSONEq(t, `{}`, string(created.Metadata))
}

func TestServiceCreateStoresMetadata(t *testing.T) {
	db := newAgentTestDB(t)
	repo := NewRepository(db)

	serverID := uuid.New()
	require.NoError(t, db.Create(&server.Server{
		ID:       serverID,
		Hostname: "meta-host-2.example.com",
		Metadata: datatypes.JSON([]byte(`{}`)),
	}).Error)

	svc := NewService(repo, stubServerLookup{
		resp: server.GetResponse{ID: serverID, Hostname: "meta-host-2.example.com"},
	}, nil)

	created, err := svc.Create(context.Background(), serverID.String(), CreateRequest{
		Name:     "agent",
		Metadata: json.RawMessage(`{"env":"prod"}`),
	})
	require.NoError(t, err)
	require.JSONEq(t, `{"env":"prod"}`, string(created.Metadata))
}

type stubAgentRepository struct {
	findAllErr error
}

func (s stubAgentRepository) FindAll(context.Context, string, int, int) ([]Agent, int64, error) {
	return nil, 0, s.findAllErr
}

func (s stubAgentRepository) FindAllGlobal(context.Context, bool, int, int) ([]Agent, int64, error) {
	return nil, 0, s.findAllErr
}

func (s stubAgentRepository) FindById(context.Context, string, string) (Agent, error) {
	return Agent{}, gorm.ErrRecordNotFound
}

func (s stubAgentRepository) FindByIdGlobal(context.Context, string) (Agent, error) {
	return Agent{}, gorm.ErrRecordNotFound
}

func (s stubAgentRepository) CreateUnassigned(context.Context, Agent) (Agent, error) {
	return Agent{}, nil
}

func (s stubAgentRepository) CreateOnServer(context.Context, Agent) (Agent, error) {
	return Agent{}, nil
}

func (s stubAgentRepository) AttachToServer(context.Context, uuid.UUID, string, []uuid.UUID) error {
	return nil
}

func (s stubAgentRepository) Delete(context.Context, string, string) error {
	return gorm.ErrRecordNotFound
}

func TestServiceGetAllReturnsRepositoryError(t *testing.T) {
	svc := NewService(
		stubAgentRepository{findAllErr: errors.New("db down")},
		stubServerLookup{resp: server.GetResponse{ID: uuid.New()}},
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
		resp: server.GetResponse{ID: serverID},
	}, nil)

	created, err := svc.Create(context.Background(), serverID.String(), CreateRequest{Name: "falcon"})
	require.NoError(t, err)

	got, err := svc.GetById(context.Background(), created.ID.String(), serverID.String())
	require.NoError(t, err)
	require.Equal(t, "falcon", got.Name)
}

func TestServiceCreateReturnsInvalidServerID(t *testing.T) {
	svc := NewService(stubAgentRepository{}, stubServerLookup{}, nil)

	_, err := svc.Create(context.Background(), "not-a-uuid", CreateRequest{Name: "agent"})
	require.ErrorIs(t, err, ErrInvalidServerID)
}

func TestServiceGetAllReturnsInvalidServerID(t *testing.T) {
	svc := NewService(stubAgentRepository{}, stubServerLookup{}, nil)

	_, _, err := svc.GetAll(context.Background(), "bad-id", 1, 10)
	require.ErrorIs(t, err, ErrInvalidServerID)
}

// Ensure stubServerLookup satisfies server.LookupService at compile time.
var _ server.LookupService = stubServerLookup{}

func TestServiceCreateUnassigned(t *testing.T) {
	svc, _ := newAgentService(t)

	created, err := svc.CreateUnassigned(context.Background(), CreateRequest{Name: "orphan"})
	require.NoError(t, err)
	require.Nil(t, created.ServerID)
	require.Equal(t, "orphan", created.Name)
}

func TestServiceListGlobalUnassignedOnly(t *testing.T) {
	db := newAgentTestDB(t)
	repo := NewRepository(db)
	svc := NewService(repo, stubServerLookup{}, nil)

	_, err := svc.CreateUnassigned(context.Background(), CreateRequest{Name: "orphan"})
	require.NoError(t, err)

	agents, total, err := svc.ListGlobal(context.Background(), true, 1, 10)
	require.NoError(t, err)
	require.Equal(t, int64(1), total)
	require.Len(t, agents, 1)
}
