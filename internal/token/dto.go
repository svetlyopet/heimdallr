package token

import "github.com/google/uuid"

type GetResponse struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	Scopes    []string  `json:"scopes"`
	CreatedBy *uuid.UUID `json:"created_by,omitempty"`
}

type CreateRequest struct {
	Name   string   `json:"name" binding:"required,min=2,max=100"`
	Scopes []string `json:"scopes" binding:"required,min=1"`
}

type CreateResponse struct {
	GetResponse
	Token string `json:"token"`
}
