package analytics

import (
	"context"

	"github.com/svetlyopet/heimdallr/internal/analytics/api"
	"gorm.io/gorm"
)

type Repository interface {
	GetAutomationOverview(ctx context.Context) (api.AutomationAnalytics, error)
	GetAutomationOverviewByID(ctx context.Context, automationID string) (api.AutomationAnalytics, error)
	GetComplianceOverview(ctx context.Context) (api.ComplianceAnalytics, error)
}

type repository struct {
	db *gorm.DB
}

type jobTotals struct {
	TotalJobs      int64
	SuccessfulJobs int64
	FailedJobs     int64
	StartedJobs    int64
}

type locationJobRow struct {
	Location       string
	TotalJobs      int64
	SuccessfulJobs int64
	FailedJobs     int64
	StartedJobs    int64
}

type automationJobRow struct {
	AutomationID   string
	Automation     string
	Provider       string
	TotalJobs      int64
	SuccessfulJobs int64
	FailedJobs     int64
	StartedJobs    int64
}

type reportTotals struct {
	TotalReports      int64
	SuccessfulReports int64
	FailedReports     int64
	StartedReports    int64
}

type latestReleaseRow struct {
	ApplicationID   string
	Application     string
	LatestReleaseID string
	LatestVersion   string
}

func (r repository) GetAutomationOverview(ctx context.Context) (api.AutomationAnalytics, error) {
	var totalAutomations int64
	if err := r.db.WithContext(ctx).
		Table("automations").
		Where("deleted_at IS NULL").
		Count(&totalAutomations).Error; err != nil {
		return api.AutomationAnalytics{}, err
	}

	totals, err := r.getTotals(ctx, "")
	if err != nil {
		return api.AutomationAnalytics{}, err
	}

	byLocation, err := r.getByLocation(ctx, "")
	if err != nil {
		return api.AutomationAnalytics{}, err
	}

	byAutomation, err := r.getByAutomation(ctx, "")
	if err != nil {
		return api.AutomationAnalytics{}, err
	}

	return buildAutomationAnalytics(totalAutomations, totals, byLocation, byAutomation), nil
}

func (r repository) GetAutomationOverviewByID(ctx context.Context, automationID string) (api.AutomationAnalytics, error) {
	var totalAutomations int64
	if err := r.db.WithContext(ctx).
		Table("automations").
		Where("id = ? AND deleted_at IS NULL", automationID).
		Count(&totalAutomations).Error; err != nil {
		return api.AutomationAnalytics{}, err
	}

	if totalAutomations == 0 {
		return api.AutomationAnalytics{}, ErrAutomationNotFound
	}

	totals, err := r.getTotals(ctx, automationID)
	if err != nil {
		return api.AutomationAnalytics{}, err
	}

	byLocation, err := r.getByLocation(ctx, automationID)
	if err != nil {
		return api.AutomationAnalytics{}, err
	}

	byAutomation, err := r.getByAutomation(ctx, automationID)
	if err != nil {
		return api.AutomationAnalytics{}, err
	}

	return buildAutomationAnalytics(totalAutomations, totals, byLocation, byAutomation), nil
}

func (r repository) getTotals(ctx context.Context, automationID string) (jobTotals, error) {
	var totals jobTotals

	query := r.db.WithContext(ctx).
		Table("jobs").
		Joins("INNER JOIN automations ON automations.id = jobs.automation_id AND automations.deleted_at IS NULL").
		Where("jobs.deleted_at IS NULL").
		Select(`
			COUNT(*) AS total_jobs,
			SUM(CASE WHEN jobs.status = 'success' THEN 1 ELSE 0 END) AS successful_jobs,
			SUM(CASE WHEN jobs.status = 'failed' THEN 1 ELSE 0 END) AS failed_jobs,
			SUM(CASE WHEN jobs.status = 'started' THEN 1 ELSE 0 END) AS started_jobs
		`)

	if automationID != "" {
		query = query.Where("jobs.automation_id = ?", automationID)
	}

	if err := query.Take(&totals).Error; err != nil {
		return jobTotals{}, err
	}

	return totals, nil
}

func (r repository) getByLocation(ctx context.Context, automationID string) ([]api.LocationJobAnalytics, error) {
	var rows []locationJobRow

	query := r.db.WithContext(ctx).
		Table("jobs").
		Joins("INNER JOIN automations ON automations.id = jobs.automation_id AND automations.deleted_at IS NULL").
		Where("jobs.deleted_at IS NULL").
		Select(`
			jobs.location AS location,
			COUNT(*) AS total_jobs,
			SUM(CASE WHEN jobs.status = 'success' THEN 1 ELSE 0 END) AS successful_jobs,
			SUM(CASE WHEN jobs.status = 'failed' THEN 1 ELSE 0 END) AS failed_jobs,
			SUM(CASE WHEN jobs.status = 'started' THEN 1 ELSE 0 END) AS started_jobs
		`).
		Group("jobs.location").
		Order("total_jobs DESC")

	if automationID != "" {
		query = query.Where("jobs.automation_id = ?", automationID)
	}

	if err := query.Find(&rows).Error; err != nil {
		return nil, err
	}

	return mapLocationRows(rows), nil
}

func (r repository) getByAutomation(ctx context.Context, automationID string) ([]api.AutomationJobAnalytics, error) {
	var rows []automationJobRow

	query := r.db.WithContext(ctx).
		Table("jobs").
		Joins("INNER JOIN automations ON automations.id = jobs.automation_id AND automations.deleted_at IS NULL").
		Where("jobs.deleted_at IS NULL").
		Select(`
			jobs.automation_id,
			jobs.automation,
			jobs.provider,
			COUNT(*) AS total_jobs,
			SUM(CASE WHEN jobs.status = 'success' THEN 1 ELSE 0 END) AS successful_jobs,
			SUM(CASE WHEN jobs.status = 'failed' THEN 1 ELSE 0 END) AS failed_jobs,
			SUM(CASE WHEN jobs.status = 'started' THEN 1 ELSE 0 END) AS started_jobs
		`).
		Group("jobs.automation_id, jobs.automation, jobs.provider").
		Order("total_jobs DESC")

	if automationID != "" {
		query = query.Where("jobs.automation_id = ?", automationID)
	}

	if err := query.Find(&rows).Error; err != nil {
		return nil, err
	}

	return mapAutomationRows(rows), nil
}

func calculateSuccessRate(successful int64, total int64) float64 {
	if total == 0 {
		return 0
	}

	return float64(successful) / float64(total) * 100
}

func buildAutomationAnalytics(
	totalAutomations int64,
	totals jobTotals,
	byLocation []api.LocationJobAnalytics,
	byAutomation []api.AutomationJobAnalytics,
) api.AutomationAnalytics {
	return api.AutomationAnalytics{
		TotalAutomations: int(totalAutomations),
		TotalJobs:        int(totals.TotalJobs),
		SuccessfulJobs:   int(totals.SuccessfulJobs),
		FailedJobs:       int(totals.FailedJobs),
		StartedJobs:      int(totals.StartedJobs),
		SuccessRate:      calculateSuccessRate(totals.SuccessfulJobs, totals.TotalJobs),
		ByLocation:       byLocation,
		ByAutomation:     byAutomation,
	}
}

func mapLocationRows(rows []locationJobRow) []api.LocationJobAnalytics {
	result := make([]api.LocationJobAnalytics, 0, len(rows))
	for _, row := range rows {
		result = append(result, api.LocationJobAnalytics{
			Location:       row.Location,
			TotalJobs:      int(row.TotalJobs),
			SuccessfulJobs: int(row.SuccessfulJobs),
			FailedJobs:     int(row.FailedJobs),
			StartedJobs:    int(row.StartedJobs),
			SuccessRate:    calculateSuccessRate(row.SuccessfulJobs, row.TotalJobs),
		})
	}

	return result
}

func mapAutomationRows(rows []automationJobRow) []api.AutomationJobAnalytics {
	result := make([]api.AutomationJobAnalytics, 0, len(rows))
	for _, row := range rows {
		result = append(result, api.AutomationJobAnalytics{
			AutomationId:   row.AutomationID,
			Automation:     row.Automation,
			Provider:       row.Provider,
			TotalJobs:      int(row.TotalJobs),
			SuccessfulJobs: int(row.SuccessfulJobs),
			FailedJobs:     int(row.FailedJobs),
			StartedJobs:    int(row.StartedJobs),
			SuccessRate:    calculateSuccessRate(row.SuccessfulJobs, row.TotalJobs),
		})
	}

	return result
}

func (r repository) GetComplianceOverview(ctx context.Context) (api.ComplianceAnalytics, error) {
	var totalApplications int64
	if err := r.db.WithContext(ctx).
		Table("applications").
		Where("deleted_at IS NULL").
		Count(&totalApplications).Error; err != nil {
		return api.ComplianceAnalytics{}, err
	}

	var totalReleases int64
	if err := r.db.WithContext(ctx).
		Table("releases").
		Where("deleted_at IS NULL").
		Count(&totalReleases).Error; err != nil {
		return api.ComplianceAnalytics{}, err
	}

	var totals reportTotals
	if err := r.db.WithContext(ctx).
		Table("reports").
		Where("deleted_at IS NULL").
		Select(`
			COUNT(*) AS total_reports,
			SUM(CASE WHEN status = 'success' THEN 1 ELSE 0 END) AS successful_reports,
			SUM(CASE WHEN status = 'failed' THEN 1 ELSE 0 END) AS failed_reports,
			SUM(CASE WHEN status = 'started' THEN 1 ELSE 0 END) AS started_reports
		`).
		Take(&totals).Error; err != nil {
		return api.ComplianceAnalytics{}, err
	}

	var latestReleases []latestReleaseRow
	if err := r.db.WithContext(ctx).
		Table("releases AS r").
		Select(`
			r.application_id AS application_id,
			r.application AS application,
			r.id AS latest_release_id,
			r.version AS latest_version
		`).
		Where(`
			r.deleted_at IS NULL
			AND NOT EXISTS (
				SELECT 1 FROM releases r2
				WHERE r2.application_id = r.application_id
					AND r2.deleted_at IS NULL
					AND (
						r2.created_at > r.created_at
						OR (r2.created_at = r.created_at AND r2.id > r.id)
					)
			)
		`).
		Order("r.application ASC").
		Find(&latestReleases).Error; err != nil {
		return api.ComplianceAnalytics{}, err
	}

	byApplication := make([]api.ApplicationComplianceStats, 0, len(latestReleases))
	for _, row := range latestReleases {
		var releaseTotals reportTotals
		if err := r.db.WithContext(ctx).
			Table("reports").
			Where("deleted_at IS NULL AND release_id = ?", row.LatestReleaseID).
			Select(`
				COUNT(*) AS total_reports,
				SUM(CASE WHEN status = 'success' THEN 1 ELSE 0 END) AS successful_reports,
				SUM(CASE WHEN status = 'failed' THEN 1 ELSE 0 END) AS failed_reports,
				SUM(CASE WHEN status = 'started' THEN 1 ELSE 0 END) AS started_reports
			`).
			Take(&releaseTotals).Error; err != nil {
			return api.ComplianceAnalytics{}, err
		}

		byApplication = append(byApplication, api.ApplicationComplianceStats{
			ApplicationId:     row.ApplicationID,
			Application:       row.Application,
			LatestReleaseId:   row.LatestReleaseID,
			LatestVersion:     row.LatestVersion,
			TotalReports:      int(releaseTotals.TotalReports),
			SuccessfulReports: int(releaseTotals.SuccessfulReports),
			FailedReports:     int(releaseTotals.FailedReports),
			StartedReports:    int(releaseTotals.StartedReports),
			SuccessRate:       calculateSuccessRate(releaseTotals.SuccessfulReports, releaseTotals.TotalReports),
		})
	}

	return api.ComplianceAnalytics{
		TotalApplications: int(totalApplications),
		TotalReleases:     int(totalReleases),
		TotalReports:      int(totals.TotalReports),
		SuccessfulReports: int(totals.SuccessfulReports),
		FailedReports:     int(totals.FailedReports),
		StartedReports:    int(totals.StartedReports),
		SuccessRate:       calculateSuccessRate(totals.SuccessfulReports, totals.TotalReports),
		ByApplication:     byApplication,
	}, nil
}

func NewRepository(db *gorm.DB) Repository {
	return &repository{
		db: db,
	}
}
