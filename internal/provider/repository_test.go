package provider

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/svetlyopet/heimdallr/internal/testutil"
	"gorm.io/gorm"
)

func newProviderTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	return testutil.NewSQLiteDB(t, &Provider{})
}

func TestRepositoryCreateAndFindByName(t *testing.T) {
	db := newProviderTestDB(t)
	repo := NewRepository(db)

	created, err := repo.Create(context.Background(), Provider{
		ID:   uuid.New(),
		Name: "awx",
		Url:  "https://awx.example.com",
	})
	require.NoError(t, err)

	found, err := repo.FindByName(context.Background(), "awx")
	require.NoError(t, err)
	require.Equal(t, created.ID, found.ID)
}

func TestRepositoryFindByIdReturnsNotFound(t *testing.T) {
	db := newProviderTestDB(t)
	repo := NewRepository(db)

	_, err := repo.FindById(context.Background(), uuid.New().String())
	require.Error(t, err)
}
