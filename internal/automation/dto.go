package automation

import "github.com/google/uuid"

type GetResponse struct {
	ID          uuid.UUID `json:"id" form:"id" binding:"required"`
	Name        string    `json:"name" form:"name" binding:"required,min=2,max=100"`
	Provider    string    `json:"provider" form:"provider" binding:"required,min=2,max=100"`
	ProviderID  uuid.UUID `json:"provider_id" form:"provider_id" binding:"required"`
	URL         string    `json:"url" form:"url" binding:"omitempty,url"`
	CostSavings float64   `json:"cost_savings" form:"cost_savings" binding:"omitempty,min=0"`
}

type CreateRequest struct {
	Name        string    `json:"name" form:"name" binding:"required,min=2,max=100"`
	ProviderID  uuid.UUID `json:"provider_id" form:"provider_id" binding:"required"`
	URL         string    `json:"url" form:"url" binding:"omitempty,url"`
	CostSavings float64   `json:"cost_savings" form:"cost_savings" binding:"omitempty,min=0"`
}

type UpdateRequest struct {
	URL         string  `json:"url" form:"url" binding:"omitempty,url"`
	CostSavings float64 `json:"cost_savings" form:"cost_savings" binding:"omitempty,min=0"`
}
