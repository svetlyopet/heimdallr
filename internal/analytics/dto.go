package analytics

type AutomationAnalyticsResponse struct {
	TotalAutomations int64                    `json:"total_automations"`
	TotalJobs        int64                    `json:"total_jobs"`
	SuccessfulJobs   int64                    `json:"successful_jobs"`
	FailedJobs       int64                    `json:"failed_jobs"`
	StartedJobs      int64                    `json:"started_jobs"`
	SuccessRate      float64                  `json:"success_rate"`
	ByLocation       []LocationJobAnalytics   `json:"by_location"`
	ByAutomation     []AutomationJobAnalytics `json:"by_automation"`
}

type LocationJobAnalytics struct {
	Location       string  `json:"location"`
	TotalJobs      int64   `json:"total_jobs"`
	SuccessfulJobs int64   `json:"successful_jobs"`
	FailedJobs     int64   `json:"failed_jobs"`
	StartedJobs    int64   `json:"started_jobs"`
	SuccessRate    float64 `json:"success_rate"`
}

type AutomationJobAnalytics struct {
	AutomationID   string  `json:"automation_id"`
	Automation     string  `json:"automation"`
	Provider       string  `json:"provider"`
	TotalJobs      int64   `json:"total_jobs"`
	SuccessfulJobs int64   `json:"successful_jobs"`
	FailedJobs     int64   `json:"failed_jobs"`
	StartedJobs    int64   `json:"started_jobs"`
	SuccessRate    float64 `json:"success_rate"`
}

type ComplianceAnalyticsResponse struct {
	TotalApplications int64                        `json:"total_applications"`
	TotalReleases     int64                        `json:"total_releases"`
	TotalReports      int64                        `json:"total_reports"`
	SuccessfulReports int64                        `json:"successful_reports"`
	FailedReports     int64                        `json:"failed_reports"`
	StartedReports    int64                        `json:"started_reports"`
	SuccessRate       float64                      `json:"success_rate"`
	ByApplication     []ApplicationComplianceStats `json:"by_application"`
}

type ApplicationComplianceStats struct {
	ApplicationID     string  `json:"application_id"`
	Application       string  `json:"application"`
	LatestReleaseID   string  `json:"latest_release_id"`
	LatestVersion     string  `json:"latest_version"`
	TotalReports      int64   `json:"total_reports"`
	SuccessfulReports int64   `json:"successful_reports"`
	FailedReports     int64   `json:"failed_reports"`
	StartedReports    int64   `json:"started_reports"`
	SuccessRate       float64 `json:"success_rate"`
}
