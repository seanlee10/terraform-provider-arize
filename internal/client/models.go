package client

type User struct {
	ID        string `json:"id"`
	Email     string `json:"email"`
	Name      string `json:"name"`
	Status    string `json:"status"`
	UserType  string `json:"userType"`
	CreatedAt string `json:"createdAt"`
}

type CreateUserInput struct {
	Email string `json:"email"`
	Name  string `json:"name,omitempty"`
}

type UpdateUserInput struct {
	Name string `json:"name,omitempty"`
}

type APIKey struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Key         string   `json:"key,omitempty"`
	Permissions []string `json:"permissions"`
	LastUsedAt  string   `json:"lastUsedAt,omitempty"`
	CreatedAt   string   `json:"createdAt"`
	ExpiresAt   string   `json:"expiresAt,omitempty"`
}

type CreateAPIKeyInput struct {
	Name        string   `json:"name"`
	Permissions []string `json:"permissions,omitempty"`
	ExpiresAt   string   `json:"expiresAt,omitempty"`
}
