package agent

import (
	"bytes"
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/svetlyopet/heimdallr/internal/logger"
	"github.com/svetlyopet/heimdallr/internal/modules/agent/api"
	"github.com/svetlyopet/heimdallr/internal/testutil"
	"gorm.io/datatypes"
)

func TestMetadataFromEntityPreservesValidShapes(t *testing.T) {
	t.Parallel()

	empty, err := metadataFromEntity(nil)
	require.NoError(t, err)
	require.Empty(t, empty)

	nullish, err := metadataFromEntity(datatypes.JSON([]byte("null")))
	require.NoError(t, err)
	require.Empty(t, nullish)

	object, err := metadataFromEntity(datatypes.JSON([]byte(`{"team":"platform"}`)))
	require.NoError(t, err)
	require.Equal(t, "platform", object["team"])
}

func TestMetadataFromEntityReturnsCorruptMetadataError(t *testing.T) {
	t.Parallel()

	_, err := metadataFromEntity(datatypes.JSON([]byte(`{not-json`)))
	require.ErrorIs(t, err, ErrCorruptMetadata)
}

func TestServiceGetByIdReturnsErrorForCorruptMetadata(t *testing.T) {
	agentID := uuid.New()
	serverID := uuid.New()
	svc := NewService(stubAgentRepository{
		findById: Agent{
			ID:       agentID,
			Name:     "corrupt-agent",
			Metadata: datatypes.JSON([]byte(`{not-json`)),
		},
	}, stubServerLookup{}, testutil.NewPostgresDB(t), nil)

	_, err := svc.GetById(context.Background(), agentID.String(), serverID.String())
	require.ErrorIs(t, err, ErrGetAgent)
}

func TestServiceGetByIdLogsEntityIDWithoutMetadataContents(t *testing.T) {
	var logOutput bytes.Buffer
	appLogger := logger.New(logger.Config{
		Format: logger.FormatJSON,
		Output: &logOutput,
	})

	agentID := uuid.New()
	svc := NewService(stubAgentRepository{
		findById: Agent{
			ID:       agentID,
			Name:     "logged-agent",
			Metadata: datatypes.JSON([]byte(`{bad`)),
		},
	}, stubServerLookup{}, testutil.NewPostgresDB(t), appLogger)

	_, err := svc.GetById(context.Background(), agentID.String(), uuid.New().String())
	require.ErrorIs(t, err, ErrGetAgent)

	logged := logOutput.String()
	require.Contains(t, logged, agentID.String())
	require.Contains(t, logged, "entity_type")
	require.NotContains(t, logged, "{bad")
}

func TestMetadataToEntityStoresRequestMetadata(t *testing.T) {
	t.Parallel()

	metadata := api.ServerMetadata{"team": "infra"}
	raw, err := metadataToEntity(&metadata)
	require.NoError(t, err)
	require.JSONEq(t, `{"team":"infra"}`, string(raw))
}
