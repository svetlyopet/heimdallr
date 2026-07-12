package job

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/svetlyopet/heimdallr/internal/requestlimits"
	"github.com/svetlyopet/heimdallr/internal/testutil"
	automation2 "github.com/svetlyopet/heimdallr/modules/automation"
	automationapi "github.com/svetlyopet/heimdallr/modules/automation/api"
	"github.com/svetlyopet/heimdallr/modules/job/api"
	provider2 "github.com/svetlyopet/heimdallr/modules/provider"
	providerapi "github.com/svetlyopet/heimdallr/modules/provider/api"
)

func newJobService(t *testing.T) (Service, automation2.Service, provider2.Service) {
	t.Helper()

	db := testutil.NewPostgresDB(t)
	providerRepo := provider2.NewRepository(db)
	providerSvc := provider2.NewService(providerRepo, nil)
	automationRepo := automation2.NewRepository(db)
	automationSvc := automation2.NewService(automationRepo, providerSvc, nil)
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
	testCases := []struct {
		name       string
		output     api.JobOutput
		maxDecoded int64
	}{
		{name: "invalid encoding", output: "not-base64!", maxDecoded: 1024},
		{name: "decoded output too large", output: "dGVzdA==", maxDecoded: 3},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			lookupCalls := 0
			jobSvc := NewService(stubJobRepository{}, stubAutomationLookup{getByIDCalls: &lookupCalls}, nil)
			ctx := requestlimits.WithContext(context.Background(), requestlimits.Values{
				MaxDecodedOutputBytes: testCase.maxDecoded,
			})

			_, err := jobSvc.Create(ctx, uuid.NewString(), api.JobCreateRequest{
				Id: "1000", Status: api.Started, Location: "global", Url: "https://example.com/job/1", Output: &testCase.output,
			})

			require.Error(t, err)
			require.Zero(t, lookupCalls)
		})
	}
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
	getByIDCalls *int
}

func (s stubAutomationLookup) GetByName(context.Context, string) (automationapi.Automation, error) {
	return automationapi.Automation{}, nil
}

func (s stubAutomationLookup) GetById(context.Context, string) (automationapi.Automation, error) {
	if s.getByIDCalls != nil {
		(*s.getByIDCalls)++
	}
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

func TestServiceCreateReturnsNotFoundForMissingAutomation(t *testing.T) {
	jobSvc := NewService(stubJobRepository{}, stubAutomationLookup{
		getByIdError: automation2.ErrAutomationNotFound,
	}, nil)

	_, err := jobSvc.Create(context.Background(), uuid.NewString(), api.JobCreateRequest{
		Id: "1000", Status: api.Started, Location: "global", Url: "https://example.com/job/1",
	})
	require.ErrorIs(t, err, automation2.ErrAutomationNotFound)
}

func TestServiceCreateReturnsInvalidAutomationID(t *testing.T) {
	jobSvc := NewService(stubJobRepository{}, stubAutomationLookup{}, nil)

	_, err := jobSvc.Create(context.Background(), "not-a-uuid", api.JobCreateRequest{
		Id: "1000", Status: api.Started, Location: "global", Url: "https://example.com/job/1",
	})
	require.ErrorIs(t, err, ErrInvalidAutomationID)
}
