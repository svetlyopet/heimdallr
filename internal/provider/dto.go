package provider

import "github.com/google/uuid"

type GetResponse struct {
	ID   uuid.UUID `json:"id" form:"id" binding:"required"`
	Name string    `json:"name" form:"name" binding:"required,min=2,max=100"`
	URL  string    `json:"url" form:"url" binding:"required,url"`
}

type CreateRequest struct {
	Name string `json:"name" form:"name" binding:"required,min=2,max=100"`
	URL  string `json:"url" form:"url" binding:"required,url"`
}
