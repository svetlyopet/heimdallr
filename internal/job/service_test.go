package job

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/svetlyopet/heimdallr/internal/automation"
	"github.com/svetlyopet/heimdallr/internal/provider"
	"github.com/svetlyopet/heimdallr/internal/testutil"
)

func newJobService(t *testing.T) (Service, automation.Service, provider.Service) {
	t.Helper()

	db := testutil.NewSQLiteDB(t, &provider.Provider{}, &automation.Automation{}, &Job{})
	providerRepo := provider.NewRepository(db)
	providerSvc := provider.NewService(providerRepo, nil)
	automationRepo := automation.NewRepository(db)
	automationSvc := automation.NewService(automationRepo, providerSvc, nil)
	jobRepo := NewRepository(db)
	jobSvc := NewService(jobRepo, automationSvc, nil)

	return jobSvc, automationSvc, providerSvc
}

func TestServiceCreateAndUpdateJobLifecycle(t *testing.T) {
	jobSvc, automationSvc, providerSvc := newJobService(t)

	prov, err := providerSvc.Create(context.Background(), provider.CreateRequest{
		Name: "awx",
		URL:  "https://awx.example.com",
	})
	require.NoError(t, err)

	auto, err := automationSvc.Create(context.Background(), automation.CreateRequest{
		Name:       "deploy",
		URL:        "https://awx.example.com/#/templates/job_template/1",
		ProviderID: prov.ID,
	})
	require.NoError(t, err)

	created, err := jobSvc.Create(context.Background(), auto.ID.String(), CreateRequest{
		ID:       "1000",
		Status:   "started",
		Location: "global",
		URL:      "https://example.com/#/jobs/playbook/200",
		Metadata: json.RawMessage(`{"inventory":"true"}`),
	})
	require.NoError(t, err)
	require.Equal(t, "started", created.Status)

	updated, err := jobSvc.Update(context.Background(), auto.ID.String(), "1000", UpdateRequest{
		Status:   "success",
		Metadata: json.RawMessage(`{"result":"ok"}`),
		Output:   "dGVzdA==",
	})
	require.NoError(t, err)
	require.Equal(t, "success", updated.Status)
	require.Equal(t, "dGVzdA==", updated.Output)
}

func TestServiceCreateReturnsInvalidOutputForBadBase64(t *testing.T) {
	jobSvc, automationSvc, providerSvc := newJobService(t)

	prov, err := providerSvc.Create(context.Background(), provider.CreateRequest{Name: "awx", URL: "https://awx.example.com"})
	require.NoError(t, err)

	auto, err := automationSvc.Create(context.Background(), automation.CreateRequest{
		Name: "deploy", URL: "https://awx.example.com/#/templates/job_template/1", ProviderID: prov.ID,
	})
	require.NoError(t, err)

	_, err = jobSvc.Create(context.Background(), auto.ID.String(), CreateRequest{
		ID: "1000", Status: "started", Location: "global", URL: "https://example.com/job/1", Output: "not-base64!",
	})
	require.Error(t, err)
}

func TestServiceGetByIdReturnsNotFound(t *testing.T) {
	jobSvc, automationSvc, providerSvc := newJobService(t)

	prov, err := providerSvc.Create(context.Background(), provider.CreateRequest{Name: "awx", URL: "https://awx.example.com"})
	require.NoError(t, err)

	auto, err := automationSvc.Create(context.Background(), automation.CreateRequest{
		Name: "deploy", URL: "https://awx.example.com/#/templates/job_template/1", ProviderID: prov.ID,
	})
	require.NoError(t, err)

	_, err = jobSvc.GetById(context.Background(), "missing", auto.ID.String())
	require.ErrorIs(t, err, ErrJobNotFound)
}

type stubAutomationLookup struct {
	getByIdError error
}

func (s stubAutomationLookup) GetByName(context.Context, string) (automation.GetResponse, error) {
	return automation.GetResponse{}, nil
}

func (s stubAutomationLookup) GetById(context.Context, string) (automation.GetResponse, error) {
	if s.getByIdError != nil {
		return automation.GetResponse{}, s.getByIdError
	}

	return automation.GetResponse{}, nil
}

type stubJobRepository struct {
	findAllErr error
}

func (s stubJobRepository) FindAll(context.Context, string, int, int) ([]Job, int64, error) {
	return nil, 0, s.findAllErr
}

func (s stubJobRepository) FindById(context.Context, string, string) (Job, error) {
	return Job{}, nil
}

func (s stubJobRepository) Create(context.Context, Job) (Job, error) {
	return Job{}, nil
}

func (s stubJobRepository) Update(context.Context, Job) (Job, error) {
	return Job{}, nil
}

func TestServiceGetAllReturnsListError(t *testing.T) {
	svc := NewService(stubJobRepository{findAllErr: errors.New("boom")}, stubAutomationLookup{}, nil)

	_, _, err := svc.GetAll(context.Background(), "00000000-0000-0000-0000-000000000001", 1, 10)
	require.ErrorIs(t, err, ErrListJobs)
}
