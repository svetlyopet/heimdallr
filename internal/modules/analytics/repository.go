package analytics

import (
	"context"
	"strings"

	"github.com/google/uuid"
	"github.com/svetlyopet/heimdallr/internal/fleetcompliance"
	"github.com/svetlyopet/heimdallr/internal/modules/analytics/api"
	"gorm.io/gorm"
)

type Repository interface {
	GetAutomationOverview(ctx context.Context) (api.AutomationAnalytics, error)
	GetAutomationOverviewByID(ctx context.Context, automationID string) (api.AutomationAnalytics, error)
	GetComplianceOverview(ctx context.Context) (api.ComplianceAnalytics, error)
	GetFleetComplianceOverview(ctx context.Context) (api.FleetComplianceAnalytics, error)
	ListNonCompliantServers(ctx context.Context, page int, limit int) ([]api.ServerFleetComplianceDetail, int64, error)
}

type repository struct {
	db *gorm.DB
}

type jobTotals struct {
	TotalJobs      int64
	SuccessfulJobs int64
	FailedJobs     int64
}

type locationJobRow struct {
	Location       string
	TotalJobs      int64
	SuccessfulJobs int64
	FailedJobs     int64
}

type automationJobRow struct {
	AutomationID   string
	Automation     string
	Provider       string
	TotalJobs      int64
	SuccessfulJobs int64
	FailedJobs     int64
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
			SUM(CASE WHEN jobs.status = 'failed' THEN 1 ELSE 0 END) AS failed_jobs
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
			SUM(CASE WHEN jobs.status = 'failed' THEN 1 ELSE 0 END) AS failed_jobs
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
			SUM(CASE WHEN jobs.status = 'failed' THEN 1 ELSE 0 END) AS failed_jobs
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

var serverCompliantCondition = fleetcompliance.ServerCompliantCondition("servers.id")

type requiredAgentCoverageRow struct {
	AgentName      string
	ServersWith    int64
	ServersMissing int64
}

type locationFleetRow struct {
	Location            string
	TotalServers        int64
	CompliantServers    int64
	NonCompliantServers int64
}

type nonCompliantServerRow struct {
	ServerID      string
	Hostname      string
	Location      string
	MissingAgents string
}

func (r repository) GetFleetComplianceOverview(ctx context.Context) (api.FleetComplianceAnalytics, error) {
	var totalServers int64
	if err := r.db.WithContext(ctx).
		Table("servers").
		Where("deleted_at IS NULL").
		Count(&totalServers).Error; err != nil {
		return api.FleetComplianceAnalytics{}, err
	}

	var totalRequiredAgents int64
	if err := r.db.WithContext(ctx).
		Table("required_agents").
		Where("deleted_at IS NULL").
		Count(&totalRequiredAgents).Error; err != nil {
		return api.FleetComplianceAnalytics{}, err
	}

	if totalRequiredAgents == 0 {
		return api.FleetComplianceAnalytics{
			TotalServers:          int(totalServers),
			CompliantServers:      int(totalServers),
			NonCompliantServers:   0,
			ComplianceRate:        calculateSuccessRate(totalServers, totalServers),
			TotalRequiredAgents:   0,
			RequiredAgentCoverage: []api.RequiredAgentCoverage{},
			ByLocation:            []api.LocationFleetCompliance{},
		}, nil
	}

	var compliantServers int64
	if err := r.db.WithContext(ctx).
		Table("servers").
		Where("deleted_at IS NULL").
		Where(serverCompliantCondition).
		Count(&compliantServers).Error; err != nil {
		return api.FleetComplianceAnalytics{}, err
	}

	nonCompliantServers := totalServers - compliantServers

	var coverageRows []requiredAgentCoverageRow
	if err := r.db.WithContext(ctx).
		Table("required_agents ra").
		Where("ra.deleted_at IS NULL").
		Select(`
			ra.agent_name AS agent_name,
			(
				SELECT COUNT(DISTINCT sa.server_id)
				FROM server_agents sa
				INNER JOIN agents a ON a.id = sa.agent_id AND a.deleted_at IS NULL
				INNER JOIN servers s ON s.id = sa.server_id AND s.deleted_at IS NULL
				WHERE a.name = ra.agent_name
			) AS servers_with,
			(
				SELECT COUNT(*)
				FROM servers s
				WHERE s.deleted_at IS NULL
					AND NOT EXISTS (
						SELECT 1
						FROM server_agents sa
						INNER JOIN agents a ON a.id = sa.agent_id AND a.deleted_at IS NULL
						WHERE sa.server_id = s.id
							AND a.name = ra.agent_name
					)
			) AS servers_missing
		`).
		Order("ra.agent_name ASC").
		Find(&coverageRows).Error; err != nil {
		return api.FleetComplianceAnalytics{}, err
	}

	requiredAgentCoverage := make([]api.RequiredAgentCoverage, 0, len(coverageRows))
	for _, row := range coverageRows {
		requiredAgentCoverage = append(requiredAgentCoverage, api.RequiredAgentCoverage{
			AgentName:      row.AgentName,
			ServersWith:    int(row.ServersWith),
			ServersMissing: int(row.ServersMissing),
			CoverageRate:   calculateSuccessRate(row.ServersWith, totalServers),
		})
	}

	var locationRows []locationFleetRow
	if err := r.db.WithContext(ctx).
		Table("servers").
		Where("servers.deleted_at IS NULL").
		Select(`
			servers.location AS location,
			COUNT(*) AS total_servers,
			SUM(CASE WHEN (` + serverCompliantCondition + `) THEN 1 ELSE 0 END) AS compliant_servers,
			SUM(CASE WHEN NOT (` + serverCompliantCondition + `) THEN 1 ELSE 0 END) AS non_compliant_servers
		`).
		Group("servers.location").
		Order("total_servers DESC").
		Find(&locationRows).Error; err != nil {
		return api.FleetComplianceAnalytics{}, err
	}

	byLocation := make([]api.LocationFleetCompliance, 0, len(locationRows))
	for _, row := range locationRows {
		byLocation = append(byLocation, api.LocationFleetCompliance{
			Location:            row.Location,
			TotalServers:        int(row.TotalServers),
			CompliantServers:    int(row.CompliantServers),
			NonCompliantServers: int(row.NonCompliantServers),
			ComplianceRate:      calculateSuccessRate(row.CompliantServers, row.TotalServers),
		})
	}

	return api.FleetComplianceAnalytics{
		TotalServers:          int(totalServers),
		CompliantServers:      int(compliantServers),
		NonCompliantServers:   int(nonCompliantServers),
		ComplianceRate:        calculateSuccessRate(compliantServers, totalServers),
		TotalRequiredAgents:   int(totalRequiredAgents),
		RequiredAgentCoverage: requiredAgentCoverage,
		ByLocation:            byLocation,
	}, nil
}

func (r repository) ListNonCompliantServers(ctx context.Context, page int, limit int) ([]api.ServerFleetComplianceDetail, int64, error) {
	var totalRequiredAgents int64
	if err := r.db.WithContext(ctx).
		Table("required_agents").
		Where("deleted_at IS NULL").
		Count(&totalRequiredAgents).Error; err != nil {
		return nil, 0, err
	}

	if totalRequiredAgents == 0 {
		return []api.ServerFleetComplianceDetail{}, 0, nil
	}

	serverCompliantForAlias := fleetcompliance.ServerCompliantCondition("s.id")
	nonCompliantCondition := "NOT (" + serverCompliantForAlias + ")"

	var total int64
	if err := r.db.WithContext(ctx).
		Table("servers s").
		Where("s.deleted_at IS NULL").
		Where(nonCompliantCondition).
		Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * limit

	var detailRows []nonCompliantServerRow
	if err := r.db.WithContext(ctx).
		Table("servers s").
		Where("s.deleted_at IS NULL").
		Where(nonCompliantCondition).
		Select(`
			s.id AS server_id,
			s.hostname AS hostname,
			s.location AS location,
			COALESCE(
				(
					SELECT string_agg(ra.agent_name, ',' ORDER BY ra.agent_name)
					FROM required_agents ra
					WHERE ra.deleted_at IS NULL
						AND NOT EXISTS (
							SELECT 1
							FROM server_agents sa
							INNER JOIN agents a ON a.id = sa.agent_id AND a.deleted_at IS NULL
							WHERE sa.server_id = s.id
								AND a.name = ra.agent_name
						)
				),
				''
			) AS missing_agents
		`).
		Order("s.hostname ASC").
		Limit(limit).
		Offset(offset).
		Find(&detailRows).Error; err != nil {
		return nil, 0, err
	}

	details := make([]api.ServerFleetComplianceDetail, 0, len(detailRows))
	for _, row := range detailRows {
		serverID, err := uuid.Parse(row.ServerID)
		if err != nil {
			return nil, 0, err
		}

		details = append(details, api.ServerFleetComplianceDetail{
			ServerId:      serverID,
			Hostname:      row.Hostname,
			Location:      row.Location,
			MissingAgents: splitMissingAgents(row.MissingAgents),
		})
	}

	return details, total, nil
}

func splitMissingAgents(raw string) []string {
	if raw == "" {
		return []string{}
	}

	return strings.Split(raw, ",")
}

func NewRepository(db *gorm.DB) Repository {
	return &repository{
		db: db,
	}
}
