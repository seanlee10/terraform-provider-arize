package client

type User struct {
	ID        string `json:"id"`
	Email     string `json:"email"`
	Name      string `json:"name"`
	Status    string `json:"status"`
	UserType  string `json:"userType"`
	CreatedAt string `json:"createdAt"`
}
