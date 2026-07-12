package agent

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/svetlyopet/heimdallr/internal/database"
	"github.com/svetlyopet/heimdallr/internal/modules/server"
	"github.com/svetlyopet/heimdallr/internal/testutil"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func newAgentTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	return testutil.NewPostgresDB(t)
}

func TestRepositoryCreateOnServer(t *testing.T) {
	db := newAgentTestDB(t)
	repo := NewRepository(db)

	serverID := uuid.New()
	require.NoError(t, db.Create(&server.Server{
		ID:       serverID,
		Hostname: "web-01.example.com",
		Metadata: datatypes.JSON([]byte(`{}`)),
	}).Error)

	created, err := repo.CreateOnServer(context.Background(), serverID, Agent{
		ID:       uuid.New(),
		Name:     "datadog",
		Type:     "monitoring",
		Version:  "7.0.0",
		Metadata: datatypes.JSON([]byte(`{}`)),
	})
	require.NoError(t, err)
	require.Equal(t, "datadog", created.Name)
}

func TestRepositoryCreateUnassigned(t *testing.T) {
	db := newAgentTestDB(t)
	repo := NewRepository(db)

	created, err := repo.CreateUnassigned(context.Background(), Agent{
		ID:       uuid.New(),
		Name:     "orphan-agent",
		Type:     "monitoring",
		Version:  "1.0.0",
		Metadata: datatypes.JSON([]byte(`{}`)),
	})
	require.NoError(t, err)
	require.Equal(t, "orphan-agent", created.Name)
}

func TestRepositoryAttachToServer(t *testing.T) {
	db := newAgentTestDB(t)
	repo := NewRepository(db)

	serverID := uuid.New()
	require.NoError(t, db.Create(&server.Server{
		ID:       serverID,
		Hostname: "attach-host.example.com",
		Metadata: datatypes.JSON([]byte(`{}`)),
	}).Error)

	created, err := repo.CreateUnassigned(context.Background(), Agent{
		ID:       uuid.New(),
		Name:     "falcon",
		Metadata: datatypes.JSON([]byte(`{}`)),
	})
	require.NoError(t, err)

	require.NoError(t, repo.AttachToServer(context.Background(), serverID, []uuid.UUID{created.ID}))

	found, err := repo.FindById(context.Background(), created.ID.String(), serverID.String())
	require.NoError(t, err)
	require.Equal(t, "falcon", found.Name)
}

func TestRepositoryAttachToServerAllowsMultipleServers(t *testing.T) {
	db := newAgentTestDB(t)
	repo := NewRepository(db)

	serverA := uuid.New()
	serverB := uuid.New()
	require.NoError(t, db.Create(&server.Server{
		ID:       serverA,
		Hostname: "host-a.example.com",
		Metadata: datatypes.JSON([]byte(`{}`)),
	}).Error)
	require.NoError(t, db.Create(&server.Server{
		ID:       serverB,
		Hostname: "host-b.example.com",
		Metadata: datatypes.JSON([]byte(`{}`)),
	}).Error)

	created, err := repo.CreateUnassigned(context.Background(), Agent{
		ID:       uuid.New(),
		Name:     "shared-agent",
		Metadata: datatypes.JSON([]byte(`{}`)),
	})
	require.NoError(t, err)

	require.NoError(t, repo.AttachToServer(context.Background(), serverA, []uuid.UUID{created.ID}))
	require.NoError(t, repo.AttachToServer(context.Background(), serverB, []uuid.UUID{created.ID}))

	servers, total, err := repo.FindServersByAgent(context.Background(), created.ID.String(), 10, 0)
	require.NoError(t, err)
	require.Equal(t, int64(2), total)
	require.Len(t, servers, 2)
}

func TestRepositoryAttachToServerRejectsDuplicateLink(t *testing.T) {
	db := newAgentTestDB(t)
	repo := NewRepository(db)

	serverID := uuid.New()
	require.NoError(t, db.Create(&server.Server{
		ID:       serverID,
		Hostname: "host-a.example.com",
		Metadata: datatypes.JSON([]byte(`{}`)),
	}).Error)

	created, err := repo.CreateOnServer(context.Background(), serverID, Agent{
		ID:       uuid.New(),
		Name:     "assigned",
		Metadata: datatypes.JSON([]byte(`{}`)),
	})
	require.NoError(t, err)

	err = repo.AttachToServer(context.Background(), serverID, []uuid.UUID{created.ID})
	require.ErrorIs(t, err, ErrAgentAlreadyLinked)
}

func TestRepositoryFindAllReturnsAgents(t *testing.T) {
	db := newAgentTestDB(t)
	repo := NewRepository(db)

	serverID := uuid.New()
	require.NoError(t, db.Create(&server.Server{
		ID:       serverID,
		Hostname: "db-01.example.com",
		Metadata: datatypes.JSON([]byte(`{}`)),
	}).Error)

	_, err := repo.CreateOnServer(context.Background(), serverID, Agent{
		ID:       uuid.New(),
		Name:     "crowdstrike",
		Metadata: datatypes.JSON([]byte(`{}`)),
	})
	require.NoError(t, err)

	agents, total, err := repo.FindAll(context.Background(), serverID.String(), 10, 0)
	require.NoError(t, err)
	require.Equal(t, int64(1), total)
	require.Len(t, agents, 1)
	require.Equal(t, "crowdstrike", agents[0].Name)
}

func TestRepositoryFindAllGlobalUnassignedOnly(t *testing.T) {
	db := newAgentTestDB(t)
	repo := NewRepository(db)

	serverID := uuid.New()
	require.NoError(t, db.Create(&server.Server{
		ID:       serverID,
		Hostname: "db-02.example.com",
		Metadata: datatypes.JSON([]byte(`{}`)),
	}).Error)

	_, err := repo.CreateUnassigned(context.Background(), Agent{
		ID:       uuid.New(),
		Name:     "unassigned",
		Metadata: datatypes.JSON([]byte(`{}`)),
	})
	require.NoError(t, err)

	_, err = repo.CreateOnServer(context.Background(), serverID, Agent{
		ID:       uuid.New(),
		Name:     "assigned",
		Metadata: datatypes.JSON([]byte(`{}`)),
	})
	require.NoError(t, err)

	agents, total, err := repo.FindAllGlobal(context.Background(), ListFilters{UnassignedOnly: true}, 10, 0)
	require.NoError(t, err)
	require.Equal(t, int64(1), total)
	require.Len(t, agents, 1)
	require.Equal(t, "unassigned", agents[0].Name)
}

func TestRepositoryFindByIdReturnsAgent(t *testing.T) {
	db := newAgentTestDB(t)
	repo := NewRepository(db)

	serverID := uuid.New()
	require.NoError(t, db.Create(&server.Server{
		ID:       serverID,
		Hostname: "app-01.example.com",
		Metadata: datatypes.JSON([]byte(`{}`)),
	}).Error)

	agentID := uuid.New()
	_, err := repo.CreateOnServer(context.Background(), serverID, Agent{
		ID:       agentID,
		Name:     "falcon",
		Metadata: datatypes.JSON([]byte(`{}`)),
	})
	require.NoError(t, err)

	found, err := repo.FindById(context.Background(), agentID.String(), serverID.String())
	require.NoError(t, err)
	require.Equal(t, agentID, found.ID)
	require.Equal(t, "falcon", found.Name)
}

func TestRepositoryFindByIdGlobalReturnsUnassignedAgent(t *testing.T) {
	db := newAgentTestDB(t)
	repo := NewRepository(db)

	created, err := repo.CreateUnassigned(context.Background(), Agent{
		ID:       uuid.New(),
		Name:     "global-agent",
		Metadata: datatypes.JSON([]byte(`{}`)),
	})
	require.NoError(t, err)

	found, err := repo.FindByIdGlobal(context.Background(), created.ID.String())
	require.NoError(t, err)
	require.Equal(t, "global-agent", found.Name)
	require.Equal(t, int64(0), found.ServerCount)
}

func TestRepositoryFindByIdReturnsNotFound(t *testing.T) {
	db := newAgentTestDB(t)
	repo := NewRepository(db)

	_, err := repo.FindById(context.Background(), uuid.New().String(), uuid.New().String())
	require.Error(t, err)
}

func TestRepositoryDetachRemovesLinkOnly(t *testing.T) {
	db := newAgentTestDB(t)
	repo := NewRepository(db)

	serverID := uuid.New()
	require.NoError(t, db.Create(&server.Server{
		ID:       serverID,
		Hostname: "del-01.example.com",
		Metadata: datatypes.JSON([]byte(`{}`)),
	}).Error)

	agentID := uuid.New()
	_, err := repo.CreateOnServer(context.Background(), serverID, Agent{
		ID:       agentID,
		Name:     "temp-agent",
		Metadata: datatypes.JSON([]byte(`{}`)),
	})
	require.NoError(t, err)

	require.NoError(t, repo.DetachFromServer(context.Background(), serverID.String(), agentID.String()))

	_, err = repo.FindById(context.Background(), agentID.String(), serverID.String())
	require.Error(t, err)

	found, err := repo.FindByIdGlobal(context.Background(), agentID.String())
	require.NoError(t, err)
	require.Equal(t, "temp-agent", found.Name)
}

func TestRepositoryFindByNameReturnsAgent(t *testing.T) {
	db := newAgentTestDB(t)
	repo := NewRepository(db)

	created, err := repo.CreateUnassigned(context.Background(), Agent{
		ID:       uuid.New(),
		Name:     "named-agent",
		Metadata: datatypes.JSON([]byte(`{}`)),
	})
	require.NoError(t, err)

	found, err := repo.FindByName(context.Background(), "named-agent")
	require.NoError(t, err)
	require.Equal(t, created.ID, found.ID)
}

func TestRepositoryCreateUnassignedRejectsDuplicateName(t *testing.T) {
	db := newAgentTestDB(t)
	repo := NewRepository(db)

	_, err := repo.CreateUnassigned(context.Background(), Agent{
		ID:       uuid.New(),
		Name:     "duplicate-agent",
		Metadata: datatypes.JSON([]byte(`{}`)),
	})
	require.NoError(t, err)

	_, err = repo.CreateUnassigned(context.Background(), Agent{
		ID:       uuid.New(),
		Name:     "duplicate-agent",
		Metadata: datatypes.JSON([]byte(`{}`)),
	})
	require.Error(t, err)
	require.True(t, database.IsUniqueViolation(err))
}
