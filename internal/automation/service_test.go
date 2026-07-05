package automation

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/svetlyopet/heimdallr/internal/provider"
	"github.com/svetlyopet/heimdallr/internal/testutil"
)

func newAutomationService(t *testing.T) (Service, provider.Service) {
	t.Helper()

	db := testutil.NewSQLiteDB(t, &provider.Provider{}, &Automation{})
	providerRepo := provider.NewRepository(db)
	providerSvc := provider.NewService(providerRepo, nil)
	automationRepo := NewRepository(db)
	automationSvc := NewService(automationRepo, providerSvc, nil)

	return automationSvc, providerSvc
}

func TestServiceCreateReturnsAutomation(t *testing.T) {
	automationSvc, providerSvc := newAutomationService(t)

	prov, err := providerSvc.Create(context.Background(), provider.CreateRequest{
		Name: "awx",
		URL:  "https://awx.example.com",
	})
	require.NoError(t, err)

	created, err := automationSvc.Create(context.Background(), CreateRequest{
		Name:       "deploy",
		URL:        "https://awx.example.com/#/templates/job_template/1",
		ProviderID: prov.ID,
	})
	require.NoError(t, err)
	require.Equal(t, "deploy", created.Name)
	require.Equal(t, "awx", created.Provider)
}

func TestServiceCreateReturnsProviderNotFound(t *testing.T) {
	automationSvc, _ := newAutomationService(t)

	_, err := automationSvc.Create(context.Background(), CreateRequest{
		Name:       "deploy",
		URL:        "https://awx.example.com/#/templates/job_template/1",
		ProviderID: uuid.New(),
	})
	require.ErrorIs(t, err, ErrCreateAutomation)
}

type stubProviderLookup struct {
	getByIdResp  provider.GetResponse
	getByIdError error
}

func (s stubProviderLookup) GetById(_ context.Context, _ string) (provider.GetResponse, error) {
	if s.getByIdError != nil {
		return provider.GetResponse{}, s.getByIdError
	}

	return s.getByIdResp, nil
}

func (s stubProviderLookup) GetByName(_ context.Context, _ string) (provider.GetResponse, error) {
	return provider.GetResponse{}, nil
}

type stubAutomationRepository struct {
	findAllErr error
}

func (s stubAutomationRepository) FindAll(context.Context, int, int) ([]Automation, int64, error) {
	return nil, 0, s.findAllErr
}

func (s stubAutomationRepository) FindByName(context.Context, string) (Automation, error) {
	return Automation{}, nil
}

func (s stubAutomationRepository) FindById(context.Context, string) (Automation, error) {
	return Automation{}, nil
}

func (s stubAutomationRepository) ExistsByName(context.Context, string) (bool, error) {
	return false, nil
}

func (s stubAutomationRepository) Create(context.Context, Automation) (Automation, error) {
	return Automation{}, nil
}

func (s stubAutomationRepository) Update(context.Context, Automation) (Automation, error) {
	return Automation{}, nil
}

func (s stubAutomationRepository) Delete(context.Context, string) error {
	return nil
}

func TestServiceGetAllReturnsListError(t *testing.T) {
	svc := NewService(stubAutomationRepository{findAllErr: errors.New("boom")}, stubProviderLookup{}, nil)

	_, _, err := svc.GetAll(context.Background(), 1, 10)
	require.ErrorIs(t, err, ErrListAutomations)
}
