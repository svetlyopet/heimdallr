package job

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/svetlyopet/heimdallr/internal/automation"
	automationapi "github.com/svetlyopet/heimdallr/internal/automation/api"
	"github.com/svetlyopet/heimdallr/internal/job/api"
	"github.com/svetlyopet/heimdallr/internal/provider"
	providerapi "github.com/svetlyopet/heimdallr/internal/provider/api"
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

	prov, err := providerSvc.Create(context.Background(), providerapi.ProviderCreateRequest{
		Name: "awx",
		Url:  providerapi.URL("https://awx.example.com"),
	})
	require.NoError(t, err)

	url := "https://awx.example.com/#/templates/job_template/1"
	auto, err := automationSvc.Create(context.Background(), automationapi.AutomationCreateRequest{
		Name:       "deploy",
		Url:        &url,
		ProviderId: prov.Id,
	})
	require.NoError(t, err)

	metadata := api.JobMetadata{"inventory": "true"}
	created, err := jobSvc.Create(context.Background(), auto.Id.String(), api.JobCreateRequest{
		Id:       "1000",
		Status:   api.Started,
		Location: "global",
		Url:      "https://example.com/#/jobs/playbook/200",
		Metadata: &metadata,
	})
	require.NoError(t, err)
	require.Equal(t, api.JobStatus("started"), created.Status)

	updateMetadata := api.JobMetadata{"result": "ok"}
	output := api.JobOutput("dGVzdA==")
	updated, err := jobSvc.Update(context.Background(), auto.Id.String(), "1000", api.JobUpdateRequest{
		Status:   api.Success,
		Metadata: &updateMetadata,
		Output:   &output,
	})
	require.NoError(t, err)
	require.Equal(t, api.Success, updated.Status)
	require.NotNil(t, updated.Output)
	require.Equal(t, "dGVzdA==", string(*updated.Output))
}

func TestServiceCreateReturnsInvalidOutputForBadBase64(t *testing.T) {
	jobSvc, automationSvc, providerSvc := newJobService(t)

	prov, err := providerSvc.Create(context.Background(), providerapi.ProviderCreateRequest{Name: "awx", Url: providerapi.URL("https://awx.example.com")})
	require.NoError(t, err)

	url := "https://awx.example.com/#/templates/job_template/1"
	auto, err := automationSvc.Create(context.Background(), automationapi.AutomationCreateRequest{
		Name: "deploy", Url: &url, ProviderId: prov.Id,
	})
	require.NoError(t, err)

	output := api.JobOutput("not-base64!")
	_, err = jobSvc.Create(context.Background(), auto.Id.String(), api.JobCreateRequest{
		Id: "1000", Status: api.Started, Location: "global", Url: "https://example.com/job/1", Output: &output,
	})
	require.Error(t, err)
}

func TestServiceGetByIdReturnsNotFound(t *testing.T) {
	jobSvc, automationSvc, providerSvc := newJobService(t)

	prov, err := providerSvc.Create(context.Background(), providerapi.ProviderCreateRequest{Name: "awx", Url: providerapi.URL("https://awx.example.com")})
	require.NoError(t, err)

	url := "https://awx.example.com/#/templates/job_template/1"
	auto, err := automationSvc.Create(context.Background(), automationapi.AutomationCreateRequest{
		Name: "deploy", Url: &url, ProviderId: prov.Id,
	})
	require.NoError(t, err)

	_, err = jobSvc.GetById(context.Background(), "missing", auto.Id.String())
	require.ErrorIs(t, err, ErrJobNotFound)
}

type stubAutomationLookup struct {
	getByIdError error
}

func (s stubAutomationLookup) GetByName(context.Context, string) (automationapi.Automation, error) {
	return automationapi.Automation{}, nil
}

func (s stubAutomationLookup) GetById(context.Context, string) (automationapi.Automation, error) {
	if s.getByIdError != nil {
		return automationapi.Automation{}, s.getByIdError
	}

	return automationapi.Automation{}, nil
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
