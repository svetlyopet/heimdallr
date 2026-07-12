package server

import (
	"bytes"
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/svetlyopet/heimdallr/internal/logger"
	"github.com/svetlyopet/heimdallr/internal/server/api"
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

	object, err := metadataFromEntity(datatypes.JSON([]byte(`{"rack":"A1"}`)))
	require.NoError(t, err)
	require.Equal(t, "A1", object["rack"])
}

func TestMetadataFromEntityReturnsCorruptMetadataError(t *testing.T) {
	t.Parallel()

	_, err := metadataFromEntity(datatypes.JSON([]byte(`{not-json`)))
	require.ErrorIs(t, err, ErrCorruptMetadata)
}

func TestServiceGetByIdReturnsErrorForCorruptMetadata(t *testing.T) {
	serverID := uuid.New()
	svc := NewService(stubServerRepository{
		findById: Server{
			ID:       serverID,
			Hostname: "corrupt-meta.example.com",
			Metadata: datatypes.JSON([]byte(`{not-json`)),
		},
	}, stubAgentAttachment{}, testutil.NewPostgresDB(t), nil)

	_, err := svc.GetById(context.Background(), serverID.String())
	require.ErrorIs(t, err, ErrGetServer)
}

func TestServiceGetAllReturnsErrorForCorruptMetadata(t *testing.T) {
	serverID := uuid.New()
	svc := NewService(stubServerRepository{
		findAllResult: []ServerWithCounts{{
			Server: Server{
				ID:       serverID,
				Hostname: "corrupt-list.example.com",
				Metadata: datatypes.JSON([]byte(`{bad`)),
			},
		}},
		findAllTotal: 1,
	}, stubAgentAttachment{}, testutil.NewPostgresDB(t), nil)

	_, _, err := svc.GetAll(context.Background(), "", 1, 10)
	require.ErrorIs(t, err, ErrListServers)
}

func TestServiceGetByIdLogsEntityIDWithoutMetadataContents(t *testing.T) {
	var logOutput bytes.Buffer
	appLogger := logger.New(logger.Config{
		Format: logger.FormatJSON,
		Output: &logOutput,
	})

	serverID := uuid.New()
	svc := NewService(stubServerRepository{
		findById: Server{
			ID:       serverID,
			Hostname: "logged.example.com",
			Metadata: datatypes.JSON([]byte(`{bad`)),
		},
	}, stubAgentAttachment{}, testutil.NewPostgresDB(t), appLogger)

	_, err := svc.GetById(context.Background(), serverID.String())
	require.ErrorIs(t, err, ErrGetServer)

	logged := logOutput.String()
	require.Contains(t, logged, serverID.String())
	require.Contains(t, logged, "entity_type")
	require.NotContains(t, logged, "{bad")
}

func TestMetadataToEntityStoresRequestMetadata(t *testing.T) {
	t.Parallel()

	metadata := api.ServerMetadata{"rack": "B2"}
	raw, err := metadataToEntity(&metadata)
	require.NoError(t, err)
	require.JSONEq(t, `{"rack":"B2"}`, string(raw))
}
