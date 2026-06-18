package job

import (
	"encoding/json"
)

type GetResponse struct {
	ID         string          `json:"id"`
	Automation string          `json:"automation"`
	Provider   string          `json:"provider"`
	Status     string          `json:"status"`
	Location   string          `json:"location"`
	URL        string          `json:"url"`
	Metadata   json.RawMessage `json:"metadata"`
	Output     string          `json:"output"`
}

type CreateRequest struct {
	ID       string          `json:"id" form:"id" binding:"required"`
	Status   string          `json:"status" form:"status" binding:"required,oneof=started skipped success failed"`
	Location string          `json:"location" form:"location" binding:"required"`
	URL      string          `json:"url" form:"url" binding:"required,url"`
	Metadata json.RawMessage `json:"metadata" form:"metadata" binding:"omitempty,json"`
	Output   string          `json:"output" form:"output" binding:"omitempty"`
}

type UpdateRequest struct {
	Status   string          `json:"status" form:"status" binding:"required,oneof=started skipped success failed"`
	Metadata json.RawMessage `json:"metadata" form:"metadata" binding:"omitempty,json"`
	Output   string          `json:"output" form:"output" binding:"omitempty"`
}
