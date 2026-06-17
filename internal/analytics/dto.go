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
