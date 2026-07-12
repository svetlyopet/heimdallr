package application

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/svetlyopet/heimdallr/internal/application/api"
	"github.com/svetlyopet/heimdallr/internal/testutil"
	"gorm.io/gorm"
)

func newApplicationService(t *testing.T) (Service, *gorm.DB) {
	t.Helper()

	db := testutil.NewPostgresDB(t)
	repo := NewRepository(db)
	return NewService(repo, nil), db
}

func TestServiceCreateReturnsApplication(t *testing.T) {
	svc, _ := newApplicationService(t)

	repoURL := api.URL("https://example.com/new")
	created, err := svc.Create(context.Background(), api.ApplicationCreateRequest{
		Name:          "new-app",
		Description:   ptr("desc"),
		RepositoryUrl: &repoURL,
	})
	require.NoError(t, err)
	require.Equal(t, "new-app", created.Name)
}

func TestServiceCreateReturnsConflictForDuplicateName(t *testing.T) {
	svc, _ := newApplicationService(t)

	repoURL := api.URL("https://example.com/dup")
	req := api.ApplicationCreateRequest{
		Name:          "dup-app",
		Description:   ptr("desc"),
		RepositoryUrl: &repoURL,
	}
	_, err := svc.Create(context.Background(), req)
	require.NoError(t, err)

	_, err = svc.Create(context.Background(), req)
	require.ErrorIs(t, err, ErrApplicationAlreadyExists)
}

func TestServiceGetByIdReturnsNotFound(t *testing.T) {
	svc, _ := newApplicationService(t)

	_, err := svc.GetById(context.Background(), uuid.New().String())
	require.ErrorIs(t, err, ErrApplicationNotFound)
}

type stubApplicationRepository struct {
	findAllErr error
}

func (s stubApplicationRepository) FindAll(context.Context, int, int) ([]Application, int64, error) {
	return nil, 0, s.findAllErr
}

func (s stubApplicationRepository) FindById(context.Context, string) (Application, error) {
	return Application{}, gorm.ErrRecordNotFound
}

func (s stubApplicationRepository) FindByName(context.Context, string) (Application, error) {
	return Application{}, gorm.ErrRecordNotFound
}

func (s stubApplicationRepository) Create(context.Context, Application) (Application, error) {
	return Application{}, nil
}

func TestServiceGetAllReturnsListError(t *testing.T) {
	svc := NewService(stubApplicationRepository{findAllErr: errors.New("boom")}, nil)

	_, _, err := svc.GetAll(context.Background(), 1, 10)
	require.ErrorIs(t, err, ErrListApplications)
}
