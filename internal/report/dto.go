package report

import (
	"encoding/json"
)

type GetResponse struct {
	ID          string          `json:"id"`
	Application string          `json:"application"`
	Version     string          `json:"version"`
	Type        string          `json:"type"`
	Status      string          `json:"status"`
	Location    string          `json:"location"`
	URL         string          `json:"url"`
	Metadata    json.RawMessage `json:"metadata"`
	Output      string          `json:"output"`
}

type CreateRequest struct {
	ID       string          `json:"id" binding:"required"`
	Type     string          `json:"type" binding:"required,oneof=sast dast sbom code_coverage custom"`
	Status   string          `json:"status" binding:"required,oneof=started skipped success failed"`
	Location string          `json:"location" binding:"omitempty"`
	URL      string          `json:"url" binding:"omitempty,url"`
	Metadata json.RawMessage `json:"metadata" binding:"omitempty,json"`
	Output   string          `json:"output" binding:"omitempty"`
}

type UpdateRequest struct {
	Status   string          `json:"status" binding:"required,oneof=started skipped success failed"`
	Metadata json.RawMessage `json:"metadata" binding:"omitempty,json"`
	Output   string          `json:"output" binding:"omitempty"`
}
