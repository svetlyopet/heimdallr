package application

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/svetlyopet/heimdallr/internal/testutil"
	"gorm.io/gorm"
)

func newApplicationTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	return testutil.NewSQLiteDB(t, &Application{})
}

func TestRepositoryFindAllReturnsApplications(t *testing.T) {
	db := newApplicationTestDB(t)
	repo := NewRepository(db)

	created, err := repo.Create(context.Background(), Application{
		ID:            uuid.New(),
		Name:          "demo-app",
		Description:   "demo",
		RepositoryURL: "https://example.com/demo",
	})
	require.NoError(t, err)

	applications, total, err := repo.FindAll(context.Background(), 10, 0)
	require.NoError(t, err)
	require.Equal(t, int64(1), total)
	require.Len(t, applications, 1)
	require.Equal(t, created.ID, applications[0].ID)
}

func TestRepositoryFindByIdReturnsNotFound(t *testing.T) {
	db := newApplicationTestDB(t)
	repo := NewRepository(db)

	_, err := repo.FindById(context.Background(), uuid.New().String())
	require.Error(t, err)
}

func TestRepositoryFindByNameReturnsApplication(t *testing.T) {
	db := newApplicationTestDB(t)
	repo := NewRepository(db)

	created, err := repo.Create(context.Background(), Application{
		ID:            uuid.New(),
		Name:          "named-app",
		Description:   "demo",
		RepositoryURL: "https://example.com/named",
	})
	require.NoError(t, err)

	found, err := repo.FindByName(context.Background(), "named-app")
	require.NoError(t, err)
	require.Equal(t, created.ID, found.ID)
}
