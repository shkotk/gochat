package requests

type Auth struct {
	Username string `json:"username" binding:"required,username"`
	Password string `json:"password" binding:"required,min=8"`
}
