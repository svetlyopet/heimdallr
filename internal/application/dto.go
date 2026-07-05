package application

import "github.com/google/uuid"

type GetResponse struct {
	ID            uuid.UUID `json:"id"`
	Name          string    `json:"name"`
	Description   string    `json:"description"`
	RepositoryURL string    `json:"repository_url"`
}

type CreateRequest struct {
	Name          string `json:"name" binding:"required,min=2,max=100"`
	Description   string `json:"description" binding:"omitempty,max=1000"`
	RepositoryURL string `json:"repository_url" binding:"omitempty,url"`
}
