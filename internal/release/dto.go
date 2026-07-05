package release

import "github.com/google/uuid"

type GetResponse struct {
	ID            uuid.UUID `json:"id"`
	ApplicationID uuid.UUID `json:"application_id"`
	Application   string    `json:"application"`
	Version       string    `json:"version"`
	CommitSHA     string    `json:"commit_sha"`
	PipelineURL   string    `json:"pipeline_url"`
	Branch        string    `json:"branch"`
}

type CreateRequest struct {
	Version     string `json:"version" binding:"required,min=1,max=255"`
	CommitSHA   string `json:"commit_sha" binding:"omitempty,max=255"`
	PipelineURL string `json:"pipeline_url" binding:"omitempty,url"`
	Branch      string `json:"branch" binding:"omitempty,max=255"`
}

type ReportSummary struct {
	Type   string `json:"type"`
	Status string `json:"status"`
	Count  int64  `json:"count"`
}

type ComplianceSummary struct {
	TotalReports      int64           `json:"total_reports"`
	SuccessfulReports int64           `json:"successful_reports"`
	FailedReports     int64           `json:"failed_reports"`
	StartedReports    int64           `json:"started_reports"`
	SuccessRate       float64         `json:"success_rate"`
	ByType            []ReportSummary `json:"by_type"`
}

type GetWithSummaryResponse struct {
	GetResponse
	Compliance ComplianceSummary `json:"compliance"`
}
