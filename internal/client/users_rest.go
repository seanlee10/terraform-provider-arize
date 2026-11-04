package client

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// Users are managed through the REST API: the GraphQL API has no createUser,
// updateUser, or deleteUser mutations, and GraphQL can't read invited users.

var ErrUserNotFound = errors.New("user not found")

// RESTUser is a user or a pending invitation. Until the invitation is accepted
// its ID is an invitation ID and Status is INVITED (or EXPIRED).
type RESTUser struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Email     string `json:"email"`
	Status    string `json:"status"`
	CreatedAt string `json:"created_at"`
}

// CreateUser invites a user with the predefined MEMBER account role and an
// email invitation link. The API is idempotent on email: a pending invitation
// is returned as-is.
func (c *RESTClient) CreateUser(ctx context.Context, name, email string) (*RESTUser, error) {
	body := map[string]interface{}{
		"name":        name,
		"email":       email,
		"role":        map[string]string{"type": "PREDEFINED", "name": "MEMBER"},
		"invite_mode": "EMAIL_LINK",
	}
	var u RESTUser
	if err := c.do(ctx, http.MethodPost, "/v2/users", body, &u); err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}
	return &u, nil
}

func (c *RESTClient) GetUserREST(ctx context.Context, userID string) (*RESTUser, error) {
	var u RESTUser
	if err := c.do(ctx, http.MethodGet, "/v2/users/"+url.PathEscape(userID), nil, &u); err != nil {
		var apiErr *APIError
		if errors.As(err, &apiErr) && apiErr.Status == http.StatusNotFound {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	return &u, nil
}

// FindUserByEmail returns the user or invitation with exactly this email
// (case-insensitive); the API's email filter is a partial match.
func (c *RESTClient) FindUserByEmail(ctx context.Context, email string) (*RESTUser, error) {
	cursor := ""
	for {
		q := url.Values{"limit": {"100"}, "email": {email}}
		if cursor != "" {
			q.Set("cursor", cursor)
		}
		var page struct {
			Users      []RESTUser `json:"users"`
			Pagination struct {
				HasMore    bool   `json:"has_more"`
				NextCursor string `json:"next_cursor"`
			} `json:"pagination"`
		}
		if err := c.do(ctx, http.MethodGet, "/v2/users?"+q.Encode(), nil, &page); err != nil {
			return nil, fmt.Errorf("failed to list users: %w", err)
		}
		for i := range page.Users {
			if strings.EqualFold(page.Users[i].Email, email) {
				return &page.Users[i], nil
			}
		}
		if !page.Pagination.HasMore || page.Pagination.NextCursor == "" {
			return nil, ErrUserNotFound
		}
		cursor = page.Pagination.NextCursor
	}
}

func (c *RESTClient) RenameUser(ctx context.Context, userID, name string) error {
	if err := c.do(ctx, http.MethodPatch, "/v2/users/"+url.PathEscape(userID), map[string]string{"name": name}, nil); err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}
	return nil
}

// DeleteUser deactivates an active user or cancels a pending invitation.
func (c *RESTClient) DeleteUser(ctx context.Context, userID string) error {
	if err := c.do(ctx, http.MethodDelete, "/v2/users/"+url.PathEscape(userID), nil, nil); err != nil {
		var apiErr *APIError
		if errors.As(err, &apiErr) && apiErr.Status == http.StatusNotFound {
			return nil
		}
		return fmt.Errorf("failed to delete user: %w", err)
	}
	return nil
}
