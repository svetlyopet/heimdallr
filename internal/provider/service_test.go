package provider

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/svetlyopet/heimdallr/internal/provider/api"
	"github.com/svetlyopet/heimdallr/internal/testutil"
	"gorm.io/gorm"
)

func newProviderService(t *testing.T) Service {
	t.Helper()

	db := testutil.NewSQLiteDB(t, &Provider{})
	return NewService(NewRepository(db), nil)
}

func TestServiceCreateReturnsProvider(t *testing.T) {
	svc := newProviderService(t)

	created, err := svc.Create(context.Background(), api.ProviderCreateRequest{
		Name: "awx",
		Url:  api.URL("https://awx.example.com"),
	})
	require.NoError(t, err)
	require.Equal(t, "awx", created.Name)
}

func TestServiceCreateReturnsConflictForDuplicateName(t *testing.T) {
	svc := newProviderService(t)

	req := api.ProviderCreateRequest{Name: "dup-provider", Url: api.URL("https://awx.example.com")}
	_, err := svc.Create(context.Background(), req)
	require.NoError(t, err)

	_, err = svc.Create(context.Background(), req)
	require.ErrorIs(t, err, ErrProviderAlreadyExists)
}

func TestServiceGetByIdReturnsNotFound(t *testing.T) {
	svc := newProviderService(t)

	_, err := svc.GetById(context.Background(), uuid.New().String())
	require.ErrorIs(t, err, ErrProviderNotFound)
}

type stubProviderRepository struct {
	findAllErr    error
	findByNameErr error
}

func (s stubProviderRepository) FindAll(context.Context, int, int) ([]Provider, int64, error) {
	return nil, 0, s.findAllErr
}

func (s stubProviderRepository) FindById(context.Context, string) (Provider, error) {
	return Provider{}, gorm.ErrRecordNotFound
}

func (s stubProviderRepository) FindByIdWithAutomations(context.Context, string) (Provider, error) {
	return Provider{}, gorm.ErrRecordNotFound
}

func (s stubProviderRepository) FindByName(context.Context, string) (Provider, error) {
	if s.findByNameErr != nil {
		return Provider{}, s.findByNameErr
	}
	return Provider{}, gorm.ErrRecordNotFound
}

func (s stubProviderRepository) Create(context.Context, Provider) (Provider, error) {
	return Provider{}, nil
}

func TestServiceGetAllReturnsListError(t *testing.T) {
	svc := NewService(stubProviderRepository{findAllErr: errors.New("boom")}, nil)

	_, _, err := svc.GetAll(context.Background(), 1, 10)
	require.ErrorIs(t, err, ErrListProviders)
}

func TestServiceGetByNameReturnsGetProviderOnDatabaseError(t *testing.T) {
	svc := NewService(stubProviderRepository{findByNameErr: errors.New("connection refused")}, nil)

	_, err := svc.GetByName(context.Background(), "awx")
	require.ErrorIs(t, err, ErrGetProvider)
}
