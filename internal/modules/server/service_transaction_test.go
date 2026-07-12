package server_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	agent2 "github.com/svetlyopet/heimdallr/internal/modules/agent"
	server2 "github.com/svetlyopet/heimdallr/internal/modules/server"
	"github.com/svetlyopet/heimdallr/internal/modules/server/api"
	"github.com/svetlyopet/heimdallr/internal/testutil"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type dbServerLookup struct {
	repo server2.Repository
}

func (s dbServerLookup) GetById(ctx context.Context, serverID string) (api.Server, error) {
	serverEntity, err := s.repo.FindById(ctx, serverID)
	if err != nil {
		return api.Server{}, server2.ErrServerNotFound
	}

	return api.Server{
		Id:              serverEntity.ID,
		Hostname:        serverEntity.Hostname,
		OperatingSystem: serverEntity.OperatingSystem,
		Hypervisor:      serverEntity.Hypervisor,
		Location:        serverEntity.Location,
	}, nil
}

func newTransactionalServerService(t *testing.T) (server2.Service, server2.Repository, *gorm.DB, agent2.Repository) {
	t.Helper()

	db := testutil.NewPostgresDB(t)
	serverRepo := server2.NewRepository(db)
	agentRepo := agent2.NewRepository(db)
	lookup := dbServerLookup{repo: serverRepo}
	attachment := agent2.NewAttachmentService(agentRepo, lookup, db, nil)

	return server2.NewService(serverRepo, attachment, db, nil), serverRepo, db, agentRepo
}

func TestServiceCreateRollsBackOnSecondAgentNameConflict(t *testing.T) {
	svc, serverRepo, db, agentRepo := newTransactionalServerService(t)

	_, err := agentRepo.CreateUnassigned(context.Background(), agent2.Agent{
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
	require.ErrorIs(t, err, server2.ErrAgentAlreadyExists)

	_, err = serverRepo.FindByHostname(context.Background(), "rollback-create.example.com")
	require.ErrorIs(t, err, gorm.ErrRecordNotFound)

	var agentCount int64
	require.NoError(t, db.Model(&agent2.Agent{}).Where("deleted_at IS NULL").Count(&agentCount).Error)
	require.Equal(t, int64(1), agentCount)
}

func TestServiceUpdateRollsBackOnSecondMissingAgentAttachment(t *testing.T) {
	svc, _, db, agentRepo := newTransactionalServerService(t)

	created, err := svc.Create(context.Background(), api.ServerCreateRequest{
		Hostname: "rollback-update.example.com",
	})
	require.NoError(t, err)

	orphan, err := agentRepo.CreateUnassigned(context.Background(), agent2.Agent{
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
	require.NoError(t, db.Model(&agent2.ServerAgent{}).Where("server_id = ?", created.Id).Count(&linkCount).Error)
	require.Equal(t, int64(0), linkCount)
}

func TestServiceCreateRejectsDuplicateAgentIDs(t *testing.T) {
	svc, serverRepo, _, _ := newTransactionalServerService(t)

	duplicateID := uuid.New()
	_, err := svc.Create(context.Background(), api.ServerCreateRequest{
		Hostname: "dup-ids.example.com",
		AgentIds: &[]uuid.UUID{duplicateID, duplicateID},
	})
	require.ErrorIs(t, err, server2.ErrDuplicateAgentIDs)

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
	require.ErrorIs(t, err, server2.ErrDuplicateAgentNames)

	_, err = serverRepo.FindByHostname(context.Background(), "dup-names.example.com")
	require.ErrorIs(t, err, gorm.ErrRecordNotFound)
}
