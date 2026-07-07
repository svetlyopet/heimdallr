package agent

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/svetlyopet/heimdallr/internal/server"
	"github.com/svetlyopet/heimdallr/internal/testutil"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func newAgentTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	return testutil.NewSQLiteDB(t, &server.Server{}, &Agent{})
}

func TestRepositoryCreatePopulatesServerHostname(t *testing.T) {
	db := newAgentTestDB(t)
	repo := NewRepository(db)

	serverID := uuid.New()
	require.NoError(t, db.Create(&server.Server{
		ID:       serverID,
		Hostname: "web-01.example.com",
		Metadata: datatypes.JSON([]byte(`{}`)),
	}).Error)

	created, err := repo.CreateOnServer(context.Background(), Agent{
		ID:       uuid.New(),
		ServerID: uuidPtr(serverID),
		Name:     "datadog",
		Type:     "monitoring",
		Version:  "7.0.0",
		Metadata: datatypes.JSON([]byte(`{}`)),
	})
	require.NoError(t, err)
	require.Equal(t, "web-01.example.com", created.Server)
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
	require.Nil(t, created.ServerID)
	require.Equal(t, "", created.Server)
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

	require.NoError(t, repo.AttachToServer(context.Background(), serverID, "attach-host.example.com", []uuid.UUID{created.ID}))

	found, err := repo.FindById(context.Background(), created.ID.String(), serverID.String())
	require.NoError(t, err)
	require.Equal(t, "attach-host.example.com", found.Server)
}

func TestRepositoryAttachToServerRejectsAssignedAgent(t *testing.T) {
	db := newAgentTestDB(t)
	repo := NewRepository(db)

	serverID := uuid.New()
	require.NoError(t, db.Create(&server.Server{
		ID:       serverID,
		Hostname: "host-a.example.com",
		Metadata: datatypes.JSON([]byte(`{}`)),
	}).Error)

	created, err := repo.CreateOnServer(context.Background(), Agent{
		ID:       uuid.New(),
		ServerID: uuidPtr(serverID),
		Name:     "assigned",
		Metadata: datatypes.JSON([]byte(`{}`)),
	})
	require.NoError(t, err)

	otherServerID := uuid.New()
	require.NoError(t, db.Create(&server.Server{
		ID:       otherServerID,
		Hostname: "host-b.example.com",
		Metadata: datatypes.JSON([]byte(`{}`)),
	}).Error)

	err = repo.AttachToServer(context.Background(), otherServerID, "host-b.example.com", []uuid.UUID{created.ID})
	require.ErrorIs(t, err, ErrAgentAlreadyAssigned)
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

	_, err := repo.CreateOnServer(context.Background(), Agent{
		ID:       uuid.New(),
		ServerID: uuidPtr(serverID),
		Name:     "crowdstrike",
		Metadata: datatypes.JSON([]byte(`{}`)),
	})
	require.NoError(t, err)

	agents, total, err := repo.FindAll(context.Background(), serverID.String(), 10, 0)
	require.NoError(t, err)
	require.Equal(t, int64(1), total)
	require.Len(t, agents, 1)
	require.Equal(t, "db-01.example.com", agents[0].Server)
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

	_, err = repo.CreateOnServer(context.Background(), Agent{
		ID:       uuid.New(),
		ServerID: uuidPtr(serverID),
		Name:     "assigned",
		Metadata: datatypes.JSON([]byte(`{}`)),
	})
	require.NoError(t, err)

	agents, total, err := repo.FindAllGlobal(context.Background(), true, 10, 0)
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
	_, err := repo.CreateOnServer(context.Background(), Agent{
		ID:       agentID,
		ServerID: uuidPtr(serverID),
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
}

func TestRepositoryFindByIdReturnsNotFound(t *testing.T) {
	db := newAgentTestDB(t)
	repo := NewRepository(db)

	_, err := repo.FindById(context.Background(), uuid.New().String(), uuid.New().String())
	require.Error(t, err)
}

func TestRepositoryDeleteRemovesAgent(t *testing.T) {
	db := newAgentTestDB(t)
	repo := NewRepository(db)

	serverID := uuid.New()
	require.NoError(t, db.Create(&server.Server{
		ID:       serverID,
		Hostname: "del-01.example.com",
		Metadata: datatypes.JSON([]byte(`{}`)),
	}).Error)

	agentID := uuid.New()
	_, err := repo.CreateOnServer(context.Background(), Agent{
		ID:       agentID,
		ServerID: uuidPtr(serverID),
		Name:     "temp-agent",
		Metadata: datatypes.JSON([]byte(`{}`)),
	})
	require.NoError(t, err)

	require.NoError(t, repo.Delete(context.Background(), serverID.String(), agentID.String()))

	_, err = repo.FindById(context.Background(), agentID.String(), serverID.String())
	require.Error(t, err)
}
