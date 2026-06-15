package job

type GetResponse struct {
	ID         string `json:"id"`
	Automation string `json:"automation"`
	Provider   string `json:"provider"`
	Status     string `json:"status"`
	Location   string `json:"location"`
	URL        string `json:"url"`
}

type CreateRequest struct {
	ID       string `json:"id" form:"id" binding:"required"`
	Status   string `json:"status" form:"status" binding:"required,oneof=started success failed"`
	Location string `json:"location" form:"location" binding:"required"`
	URL      string `json:"url" form:"url" binding:"required,url"`
}

type UpdateRequest struct {
	Status string `json:"status" form:"status" binding:"required,oneof=started success failed"`
}
