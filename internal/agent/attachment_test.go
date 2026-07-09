package agent

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/svetlyopet/heimdallr/internal/server"
	serverapi "github.com/svetlyopet/heimdallr/internal/server/api"
	"gorm.io/datatypes"
)

func TestAttachmentServiceAttachAgentIDs(t *testing.T) {
	db := newAgentTestDB(t)
	repo := NewRepository(db)

	serverID := uuid.New()
	require.NoError(t, db.Create(&server.Server{
		ID:       serverID,
		Hostname: "attach-host.example.com",
		Metadata: datatypes.JSON([]byte(`{}`)),
	}).Error)

	orphan, err := repo.CreateUnassigned(context.Background(), Agent{
		ID:       uuid.New(),
		Name:     "orphan",
		Metadata: datatypes.JSON([]byte(`{}`)),
	})
	require.NoError(t, err)

	attachment := NewAttachmentService(repo, stubServerLookup{
		resp: serverapi.Server{Id: serverID, Hostname: "attach-host.example.com"},
	}, nil)

	require.NoError(t, attachment.AttachAgentIDs(context.Background(), serverID, []uuid.UUID{orphan.ID}))

	found, err := repo.FindById(context.Background(), orphan.ID.String(), serverID.String())
	require.NoError(t, err)
	require.Equal(t, "orphan", found.Name)
}

func TestAttachmentServiceCreateAgentsOnServer(t *testing.T) {
	db := newAgentTestDB(t)
	repo := NewRepository(db)

	serverID := uuid.New()
	require.NoError(t, db.Create(&server.Server{
		ID:       serverID,
		Hostname: "inline-host.example.com",
		Metadata: datatypes.JSON([]byte(`{}`)),
	}).Error)

	attachment := NewAttachmentService(repo, stubServerLookup{
		resp: serverapi.Server{Id: serverID, Hostname: "inline-host.example.com"},
	}, nil)

	agentType := "security"
	agentVersion := "1.0.0"
	require.NoError(t, attachment.CreateAgentsOnServer(context.Background(), serverID, []serverapi.AgentCreateRequest{{
		Name:    "crowdstrike",
		Type:    &agentType,
		Version: &agentVersion,
	}}))

	agents, total, err := repo.FindAll(context.Background(), serverID.String(), 10, 0)
	require.NoError(t, err)
	require.Equal(t, int64(1), total)
	require.Equal(t, "crowdstrike", agents[0].Name)
}
