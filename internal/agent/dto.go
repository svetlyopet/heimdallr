package agent

import (
	"encoding/json"

	"github.com/google/uuid"
)

type GetResponse struct {
	ID       uuid.UUID       `json:"id"`
	Server   string          `json:"server,omitempty"`
	ServerID *uuid.UUID      `json:"server_id,omitempty"`
	Name     string          `json:"name"`
	Type     string          `json:"type"`
	Version  string          `json:"version"`
	Metadata json.RawMessage `json:"metadata"`
}

type CreateRequest struct {
	Name     string          `json:"name" binding:"required,min=1,max=255"`
	Type     string          `json:"type" binding:"omitempty,max=255"`
	Version  string          `json:"version" binding:"omitempty,max=255"`
	Metadata json.RawMessage `json:"metadata" binding:"omitempty,json"`
}
