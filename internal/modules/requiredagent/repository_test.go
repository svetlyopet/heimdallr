package requiredagent

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/svetlyopet/heimdallr/internal/testutil"
)

func TestRepositoryCreateAndFindRequiredAgent(t *testing.T) {
	db := testutil.NewPostgresDB(t)
	repo := NewRepository(db)

	created, err := repo.Create(context.Background(), RequiredAgent{
		ID:        uuid.New(),
		AgentName: "req-alpha",
		AgentType: "security",
	})
	require.NoError(t, err)

	found, err := repo.FindById(context.Background(), created.ID.String())
	require.NoError(t, err)
	require.Equal(t, "req-alpha", found.AgentName)
}

func TestRepositoryFindByNameReturnsConflictCandidate(t *testing.T) {
	db := testutil.NewPostgresDB(t)
	repo := NewRepository(db)

	_, err := repo.Create(context.Background(), RequiredAgent{
		ID:        uuid.New(),
		AgentName: "req-beta",
		AgentType: "monitoring",
	})
	require.NoError(t, err)

	_, err = repo.FindByName(context.Background(), "req-beta")
	require.NoError(t, err)
}
