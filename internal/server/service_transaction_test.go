package server_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/svetlyopet/heimdallr/internal/agent"
	"github.com/svetlyopet/heimdallr/internal/server"
	"github.com/svetlyopet/heimdallr/internal/server/api"
	"github.com/svetlyopet/heimdallr/internal/testutil"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type dbServerLookup struct {
	repo server.Repository
}

func (s dbServerLookup) GetById(ctx context.Context, serverID string) (api.Server, error) {
	serverEntity, err := s.repo.FindById(ctx, serverID)
	if err != nil {
		return api.Server{}, server.ErrServerNotFound
	}

	return api.Server{
		Id:              serverEntity.ID,
		Hostname:        serverEntity.Hostname,
		OperatingSystem: serverEntity.OperatingSystem,
		Hypervisor:      serverEntity.Hypervisor,
		Location:        serverEntity.Location,
	}, nil
}

func newTransactionalServerService(t *testing.T) (server.Service, server.Repository, *gorm.DB, agent.Repository) {
	t.Helper()

	db := testutil.NewSQLiteDB(t, &server.Server{}, &agent.Agent{}, &agent.ServerAgent{})
	serverRepo := server.NewRepository(db)
	agentRepo := agent.NewRepository(db)
	lookup := dbServerLookup{repo: serverRepo}
	attachment := agent.NewAttachmentService(agentRepo, lookup, db, nil)

	return server.NewService(serverRepo, attachment, db, nil), serverRepo, db, agentRepo
}

func TestServiceCreateRollsBackOnSecondAgentNameConflict(t *testing.T) {
	svc, serverRepo, db, agentRepo := newTransactionalServerService(t)

	_, err := agentRepo.CreateUnassigned(context.Background(), agent.Agent{
		ID:       uuid.New(),
		Name:     "taken",
		Metadata: datatypes.JSON([]byte(`{}`)),
	})
	require.NoError(t, err)

	agentType := "security"
	_, err = svc.Create(context.Background(), api.ServerCreateRequest{
		Hostname: "rollback-create.example.com",
		Agents: &[]api.AgentCreateRequest{
			{Name: "first-agent", Type: &agentType},
			{Name: "taken", Type: &agentType},
		},
	})
	require.ErrorIs(t, err, server.ErrAgentAlreadyExists)

	_, err = serverRepo.FindByHostname(context.Background(), "rollback-create.example.com")
	require.ErrorIs(t, err, gorm.ErrRecordNotFound)

	var agentCount int64
	require.NoError(t, db.Model(&agent.Agent{}).Where("deleted_at IS NULL").Count(&agentCount).Error)
	require.Equal(t, int64(1), agentCount)
}

func TestServiceUpdateRollsBackOnSecondMissingAgentAttachment(t *testing.T) {
	svc, _, db, agentRepo := newTransactionalServerService(t)

	created, err := svc.Create(context.Background(), api.ServerCreateRequest{
		Hostname: "rollback-update.example.com",
	})
	require.NoError(t, err)

	orphan, err := agentRepo.CreateUnassigned(context.Background(), agent.Agent{
		ID:       uuid.New(),
		Name:     "orphan",
		Metadata: datatypes.JSON([]byte(`{}`)),
	})
	require.NoError(t, err)

	_, err = svc.Update(context.Background(), created.Id.String(), api.ServerUpdateRequest{
		AgentIds: &[]uuid.UUID{orphan.ID, uuid.New()},
	})
	require.Error(t, err)

	var linkCount int64
	require.NoError(t, db.Model(&agent.ServerAgent{}).Where("server_id = ?", created.Id).Count(&linkCount).Error)
	require.Equal(t, int64(0), linkCount)
}

func TestServiceCreateRejectsDuplicateAgentIDs(t *testing.T) {
	svc, serverRepo, _, _ := newTransactionalServerService(t)

	duplicateID := uuid.New()
	_, err := svc.Create(context.Background(), api.ServerCreateRequest{
		Hostname: "dup-ids.example.com",
		AgentIds: &[]uuid.UUID{duplicateID, duplicateID},
	})
	require.ErrorIs(t, err, server.ErrDuplicateAgentIDs)

	_, err = serverRepo.FindByHostname(context.Background(), "dup-ids.example.com")
	require.ErrorIs(t, err, gorm.ErrRecordNotFound)
}

func TestServiceCreateRejectsDuplicateAgentNames(t *testing.T) {
	svc, serverRepo, _, _ := newTransactionalServerService(t)

	agentType := "security"
	_, err := svc.Create(context.Background(), api.ServerCreateRequest{
		Hostname: "dup-names.example.com",
		Agents: &[]api.AgentCreateRequest{
			{Name: "same-name", Type: &agentType},
			{Name: "same-name", Type: &agentType},
		},
	})
	require.ErrorIs(t, err, server.ErrDuplicateAgentNames)

	_, err = serverRepo.FindByHostname(context.Background(), "dup-names.example.com")
	require.ErrorIs(t, err, gorm.ErrRecordNotFound)
}
