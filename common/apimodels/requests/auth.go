package requests

type Auth struct {
	Username string `json:"username" binding:"required,min=4,max=20,name"`
	Password string `json:"password" binding:"required,min=8"`
}
