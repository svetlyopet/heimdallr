package auth

type GetResponse struct {
	ID       string   `json:"id" form:"id" binding:"required"`
	Username string   `json:"username" form:"username" binding:"required"`
	Email    string   `json:"email" form:"email" binding:"required,email"`
	Roles    []string `json:"roles" form:"roles" binding:"required"`
}

type CreateRequest struct {
	Username string   `json:"username" form:"username" binding:"required"`
	Email    string   `json:"email" form:"email" binding:"required,email"`
	Password string   `json:"password" form:"password" binding:"required"`
	Roles    []string `json:"roles" form:"roles" binding:"omitempty"`
}

type UpdateRequest struct {
	Email    string   `json:"email" form:"email" binding:"omitempty,email"`
	Password string   `json:"password" form:"password" binding:"omitempty"`
	Roles    []string `json:"roles" form:"roles" binding:"omitempty"`
}
