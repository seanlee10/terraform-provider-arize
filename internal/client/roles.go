package client

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
)

var ErrRoleNotFound = errors.New("role not found")

type Role struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	Description  *string  `json:"description"`
	Permissions  []string `json:"permissions"`
	IsPredefined bool     `json:"is_predefined"`
	CreatedAt    string   `json:"created_at"`
	UpdatedAt    string   `json:"updated_at"`
}

type CreateRoleInput struct {
	Name        string   `json:"name"`
	Description *string  `json:"description,omitempty"`
	Permissions []string `json:"permissions"`
}

// UpdateRoleInput always sends every field: permissions are replaced as a
// whole and a null description clears it.
type UpdateRoleInput struct {
	Name        string   `json:"name"`
	Description *string  `json:"description"`
	Permissions []string `json:"permissions"`
}

func (c *RESTClient) GetRole(ctx context.Context, roleID string) (*Role, error) {
	var role Role
	if err := c.do(ctx, http.MethodGet, "/v2/roles/"+url.PathEscape(roleID), nil, &role); err != nil {
		var apiErr *APIError
		if errors.As(err, &apiErr) && apiErr.Status == http.StatusNotFound {
			return nil, ErrRoleNotFound
		}
		return nil, fmt.Errorf("failed to get role: %w", err)
	}
	return &role, nil
}

// ListRoles returns all predefined and custom roles in the account.
func (c *RESTClient) ListRoles(ctx context.Context) ([]Role, error) {
	var roles []Role
	cursor := ""
	for {
		q := url.Values{"limit": {"100"}}
		if cursor != "" {
			q.Set("cursor", cursor)
		}
		var page struct {
			Roles      []Role `json:"roles"`
			Pagination struct {
				HasMore    bool   `json:"has_more"`
				NextCursor string `json:"next_cursor"`
			} `json:"pagination"`
		}
		if err := c.do(ctx, http.MethodGet, "/v2/roles?"+q.Encode(), nil, &page); err != nil {
			return nil, fmt.Errorf("failed to list roles: %w", err)
		}
		roles = append(roles, page.Roles...)
		if !page.Pagination.HasMore || page.Pagination.NextCursor == "" {
			return roles, nil
		}
		cursor = page.Pagination.NextCursor
	}
}

func (c *RESTClient) CreateRole(ctx context.Context, input *CreateRoleInput) (*Role, error) {
	var role Role
	if err := c.do(ctx, http.MethodPost, "/v2/roles", input, &role); err != nil {
		return nil, fmt.Errorf("failed to create role: %w", err)
	}
	return &role, nil
}

func (c *RESTClient) UpdateRole(ctx context.Context, roleID string, input *UpdateRoleInput) (*Role, error) {
	var role Role
	if err := c.do(ctx, http.MethodPatch, "/v2/roles/"+url.PathEscape(roleID), input, &role); err != nil {
		return nil, fmt.Errorf("failed to update role: %w", err)
	}
	return &role, nil
}

func (c *RESTClient) DeleteRole(ctx context.Context, roleID string) error {
	if err := c.do(ctx, http.MethodDelete, "/v2/roles/"+url.PathEscape(roleID), nil, nil); err != nil {
		var apiErr *APIError
		if errors.As(err, &apiErr) && apiErr.Status == http.StatusNotFound {
			return nil
		}
		return fmt.Errorf("failed to delete role: %w", err)
	}
	return nil
}
