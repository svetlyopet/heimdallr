package analytics

import (
	"context"

	"gorm.io/gorm"
)

type Repository interface {
	GetAutomationOverview(ctx context.Context) (AutomationAnalyticsResponse, error)
	GetAutomationOverviewByID(ctx context.Context, automationID string) (AutomationAnalyticsResponse, error)
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

func NewRepository(db *gorm.DB) Repository {
	return &repository{
		db: db,
	}
}
