package analytics

import (
	"context"

	"gorm.io/gorm"
)

type Repository interface {
	GetAutomationOverview(ctx context.Context) (AutomationAnalyticsResponse, error)
	GetAutomationOverviewByID(ctx context.Context, automationID string) (AutomationAnalyticsResponse, error)
	GetComplianceOverview(ctx context.Context) (ComplianceAnalyticsResponse, error)
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

func (r repository) GetAutomationOverview(ctx context.Context) (AutomationAnalyticsResponse, error) {
	var totalAutomations int64
	if err := r.db.WithContext(ctx).
		Table("automations").
		Count(&totalAutomations).Error; err != nil {
		return AutomationAnalyticsResponse{}, err
	}

	totals, err := r.getTotals(ctx, "")
	if err != nil {
		return AutomationAnalyticsResponse{}, err
	}

	byLocation, err := r.getByLocation(ctx, "")
	if err != nil {
		return AutomationAnalyticsResponse{}, err
	}

	byAutomation, err := r.getByAutomation(ctx, "")
	if err != nil {
		return AutomationAnalyticsResponse{}, err
	}

	return AutomationAnalyticsResponse{
		TotalAutomations: totalAutomations,
		TotalJobs:        totals.TotalJobs,
		SuccessfulJobs:   totals.SuccessfulJobs,
		FailedJobs:       totals.FailedJobs,
		StartedJobs:      totals.StartedJobs,
		SuccessRate:      calculateSuccessRate(totals.SuccessfulJobs, totals.TotalJobs),
		ByLocation:       byLocation,
		ByAutomation:     byAutomation,
	}, nil
}

func (r repository) GetAutomationOverviewByID(ctx context.Context, automationID string) (AutomationAnalyticsResponse, error) {
	var totalAutomations int64
	if err := r.db.WithContext(ctx).
		Table("automations").
		Where("id = ?", automationID).
		Count(&totalAutomations).Error; err != nil {
		return AutomationAnalyticsResponse{}, err
	}

	if totalAutomations == 0 {
		return AutomationAnalyticsResponse{}, ErrAutomationNotFound
	}

	totals, err := r.getTotals(ctx, automationID)
	if err != nil {
		return AutomationAnalyticsResponse{}, err
	}

	byLocation, err := r.getByLocation(ctx, automationID)
	if err != nil {
		return AutomationAnalyticsResponse{}, err
	}

	byAutomation, err := r.getByAutomation(ctx, automationID)
	if err != nil {
		return AutomationAnalyticsResponse{}, err
	}

	return AutomationAnalyticsResponse{
		TotalAutomations: totalAutomations,
		TotalJobs:        totals.TotalJobs,
		SuccessfulJobs:   totals.SuccessfulJobs,
		FailedJobs:       totals.FailedJobs,
		StartedJobs:      totals.StartedJobs,
		SuccessRate:      calculateSuccessRate(totals.SuccessfulJobs, totals.TotalJobs),
		ByLocation:       byLocation,
		ByAutomation:     byAutomation,
	}, nil
}

func (r repository) getTotals(ctx context.Context, automationID string) (jobTotals, error) {
	var totals jobTotals

	query := r.db.WithContext(ctx).
		Table("jobs").
		Select(`
			COUNT(*) AS total_jobs,
			SUM(CASE WHEN status = 'success' THEN 1 ELSE 0 END) AS successful_jobs,
			SUM(CASE WHEN status = 'failed' THEN 1 ELSE 0 END) AS failed_jobs,
			SUM(CASE WHEN status = 'started' THEN 1 ELSE 0 END) AS started_jobs
		`)

	if automationID != "" {
		query = query.Where("automation_id = ?", automationID)
	}

	if err := query.Take(&totals).Error; err != nil {
		return jobTotals{}, err
	}

	return totals, nil
}

func (r repository) getByLocation(ctx context.Context, automationID string) ([]LocationJobAnalytics, error) {
	var rows []LocationJobAnalytics

	query := r.db.WithContext(ctx).
		Table("jobs").
		Select(`
			location,
			COUNT(*) AS total_jobs,
			SUM(CASE WHEN status = 'success' THEN 1 ELSE 0 END) AS successful_jobs,
			SUM(CASE WHEN status = 'failed' THEN 1 ELSE 0 END) AS failed_jobs,
			SUM(CASE WHEN status = 'started' THEN 1 ELSE 0 END) AS started_jobs
		`).
		Group("location").
		Order("total_jobs DESC")

	if automationID != "" {
		query = query.Where("automation_id = ?", automationID)
	}

	if err := query.Find(&rows).Error; err != nil {
		return nil, err
	}

	for index := range rows {
		rows[index].SuccessRate = calculateSuccessRate(rows[index].SuccessfulJobs, rows[index].TotalJobs)
	}

	return rows, nil
}

func (r repository) getByAutomation(ctx context.Context, automationID string) ([]AutomationJobAnalytics, error) {
	var rows []AutomationJobAnalytics

	query := r.db.WithContext(ctx).
		Table("jobs").
		Select(`
			automation_id,
			automation,
			provider,
			COUNT(*) AS total_jobs,
			SUM(CASE WHEN status = 'success' THEN 1 ELSE 0 END) AS successful_jobs,
			SUM(CASE WHEN status = 'failed' THEN 1 ELSE 0 END) AS failed_jobs,
			SUM(CASE WHEN status = 'started' THEN 1 ELSE 0 END) AS started_jobs
		`).
		Group("automation_id, automation, provider").
		Order("total_jobs DESC")

	if automationID != "" {
		query = query.Where("automation_id = ?", automationID)
	}

	if err := query.Find(&rows).Error; err != nil {
		return nil, err
	}

	for index := range rows {
		rows[index].SuccessRate = calculateSuccessRate(rows[index].SuccessfulJobs, rows[index].TotalJobs)
	}

	return rows, nil
}

func calculateSuccessRate(successfulJobs int64, totalJobs int64) float64 {
	if totalJobs == 0 {
		return 0
	}

	return float64(successfulJobs) / float64(totalJobs) * 100
}

func (r repository) GetComplianceOverview(ctx context.Context) (ComplianceAnalyticsResponse, error) {
	var totalApplications int64
	if err := r.db.WithContext(ctx).
		Table("applications").
		Count(&totalApplications).Error; err != nil {
		return ComplianceAnalyticsResponse{}, err
	}

	var totalReleases int64
	if err := r.db.WithContext(ctx).
		Table("releases").
		Count(&totalReleases).Error; err != nil {
		return ComplianceAnalyticsResponse{}, err
	}

	type reportTotals struct {
		TotalReports      int64
		SuccessfulReports int64
		FailedReports     int64
		StartedReports    int64
	}

	var totals reportTotals
	if err := r.db.WithContext(ctx).
		Table("reports").
		Select(`
			COUNT(*) AS total_reports,
			SUM(CASE WHEN status = 'success' THEN 1 ELSE 0 END) AS successful_reports,
			SUM(CASE WHEN status = 'failed' THEN 1 ELSE 0 END) AS failed_reports,
			SUM(CASE WHEN status = 'started' THEN 1 ELSE 0 END) AS started_reports
		`).
		Take(&totals).Error; err != nil {
		return ComplianceAnalyticsResponse{}, err
	}

	type latestReleaseRow struct {
		ApplicationID   string
		Application     string
		LatestReleaseID string
		LatestVersion   string
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
		Joins(`
			INNER JOIN (
				SELECT application_id, MAX(created_at) AS max_created_at
				FROM releases
				GROUP BY application_id
			) latest ON latest.application_id = r.application_id AND latest.max_created_at = r.created_at
		`).
		Order("r.application ASC").
		Find(&latestReleases).Error; err != nil {
		return ComplianceAnalyticsResponse{}, err
	}

	byApplication := make([]ApplicationComplianceStats, 0, len(latestReleases))
	for _, row := range latestReleases {
		var releaseTotals reportTotals
		if err := r.db.WithContext(ctx).
			Table("reports").
			Select(`
				COUNT(*) AS total_reports,
				SUM(CASE WHEN status = 'success' THEN 1 ELSE 0 END) AS successful_reports,
				SUM(CASE WHEN status = 'failed' THEN 1 ELSE 0 END) AS failed_reports,
				SUM(CASE WHEN status = 'started' THEN 1 ELSE 0 END) AS started_reports
			`).
			Where("release_id = ?", row.LatestReleaseID).
			Take(&releaseTotals).Error; err != nil {
			return ComplianceAnalyticsResponse{}, err
		}

		byApplication = append(byApplication, ApplicationComplianceStats{
			ApplicationID:     row.ApplicationID,
			Application:       row.Application,
			LatestReleaseID:   row.LatestReleaseID,
			LatestVersion:     row.LatestVersion,
			TotalReports:      releaseTotals.TotalReports,
			SuccessfulReports: releaseTotals.SuccessfulReports,
			FailedReports:     releaseTotals.FailedReports,
			StartedReports:    releaseTotals.StartedReports,
			SuccessRate:       calculateSuccessRate(releaseTotals.SuccessfulReports, releaseTotals.TotalReports),
		})
	}

	return ComplianceAnalyticsResponse{
		TotalApplications: totalApplications,
		TotalReleases:     totalReleases,
		TotalReports:      totals.TotalReports,
		SuccessfulReports: totals.SuccessfulReports,
		FailedReports:     totals.FailedReports,
		StartedReports:    totals.StartedReports,
		SuccessRate:       calculateSuccessRate(totals.SuccessfulReports, totals.TotalReports),
		ByApplication:     byApplication,
	}, nil
}

func NewRepository(db *gorm.DB) Repository {
	return &repository{
		db: db,
	}
}
