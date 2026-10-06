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

type Role struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description,omitempty"`
	Permissions []string `json:"permissions"`
	CreatedAt   string   `json:"createdAt"`
	UpdatedAt   string   `json:"updatedAt"`
}

type CreateRoleInput struct {
	Name        string   `json:"name"`
	Description string   `json:"description,omitempty"`
	Permissions []string `json:"permissions,omitempty"`
}

type UpdateRoleInput struct {
	Name        string   `json:"name,omitempty"`
	Description string   `json:"description,omitempty"`
	Permissions []string `json:"permissions,omitempty"`
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
