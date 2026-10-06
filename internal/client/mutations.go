package client

import (
	"context"
	"fmt"
)

const createUserMutation = `
mutation CreateUser($input: CreateUserInput!) {
  createUser(input: $input) {
    user {
      id
      email
      firstName
      lastName
      userType
      createdAt
      updatedAt
    }
    error
  }
}
`

func (c *Client) CreateUser(ctx context.Context, input *CreateUserInput) (*User, error) {
	var result struct {
		CreateUser struct {
			User  *User  `json:"user"`
			Error string `json:"error"`
		} `json:"createUser"`
	}

	variables := map[string]interface{}{
		"input": input,
	}

	if err := c.execute(ctx, createUserMutation, variables, &result); err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	if result.CreateUser.Error != "" {
		return nil, fmt.Errorf("failed to create user: %s", result.CreateUser.Error)
	}

	return result.CreateUser.User, nil
}

const updateUserMutation = `
mutation UpdateUser($id: ID!, $input: UpdateUserInput!) {
  updateUser(id: $id, input: $input) {
    user {
      id
      email
      firstName
      lastName
      userType
      createdAt
      updatedAt
    }
    error
  }
}
`

func (c *Client) UpdateUser(ctx context.Context, userID string, input *UpdateUserInput) (*User, error) {
	var result struct {
		UpdateUser struct {
			User  *User  `json:"user"`
			Error string `json:"error"`
		} `json:"updateUser"`
	}

	variables := map[string]interface{}{
		"id":    userID,
		"input": input,
	}

	if err := c.execute(ctx, updateUserMutation, variables, &result); err != nil {
		return nil, fmt.Errorf("failed to update user: %w", err)
	}

	if result.UpdateUser.Error != "" {
		return nil, fmt.Errorf("failed to update user: %s", result.UpdateUser.Error)
	}

	return result.UpdateUser.User, nil
}

const deleteUserMutation = `
mutation DeleteUser($id: ID!) {
  deleteUser(id: $id) {
    success
    error
  }
}
`

func (c *Client) DeleteUser(ctx context.Context, userID string) error {
	var result struct {
		DeleteUser struct {
			Success bool   `json:"success"`
			Error   string `json:"error"`
		} `json:"deleteUser"`
	}

	variables := map[string]interface{}{
		"id": userID,
	}

	if err := c.execute(ctx, deleteUserMutation, variables, &result); err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	if result.DeleteUser.Error != "" {
		return fmt.Errorf("failed to delete user: %s", result.DeleteUser.Error)
	}

	return nil
}

const createRoleMutation = `
mutation CreateRole($input: CreateRoleInput!) {
  createRole(input: $input) {
    role {
      id
      name
      description
      permissions
      createdAt
      updatedAt
    }
    error
  }
}
`

func (c *Client) CreateRole(ctx context.Context, input *CreateRoleInput) (*Role, error) {
	var result struct {
		CreateRole struct {
			Role  *Role  `json:"role"`
			Error string `json:"error"`
		} `json:"createRole"`
	}

	variables := map[string]interface{}{
		"input": input,
	}

	if err := c.execute(ctx, createRoleMutation, variables, &result); err != nil {
		return nil, fmt.Errorf("failed to create role: %w", err)
	}

	if result.CreateRole.Error != "" {
		return nil, fmt.Errorf("failed to create role: %s", result.CreateRole.Error)
	}

	return result.CreateRole.Role, nil
}

const updateRoleMutation = `
mutation UpdateRole($id: ID!, $input: UpdateRoleInput!) {
  updateRole(id: $id, input: $input) {
    role {
      id
      name
      description
      permissions
      createdAt
      updatedAt
    }
    error
  }
}
`

func (c *Client) UpdateRole(ctx context.Context, roleID string, input *UpdateRoleInput) (*Role, error) {
	var result struct {
		UpdateRole struct {
			Role  *Role  `json:"role"`
			Error string `json:"error"`
		} `json:"updateRole"`
	}

	variables := map[string]interface{}{
		"id":    roleID,
		"input": input,
	}

	if err := c.execute(ctx, updateRoleMutation, variables, &result); err != nil {
		return nil, fmt.Errorf("failed to update role: %w", err)
	}

	if result.UpdateRole.Error != "" {
		return nil, fmt.Errorf("failed to update role: %s", result.UpdateRole.Error)
	}

	return result.UpdateRole.Role, nil
}

const deleteRoleMutation = `
mutation DeleteRole($id: ID!) {
  deleteRole(id: $id) {
    success
    error
  }
}
`

func (c *Client) DeleteRole(ctx context.Context, roleID string) error {
	var result struct {
		DeleteRole struct {
			Success bool   `json:"success"`
			Error   string `json:"error"`
		} `json:"deleteRole"`
	}

	variables := map[string]interface{}{
		"id": roleID,
	}

	if err := c.execute(ctx, deleteRoleMutation, variables, &result); err != nil {
		return fmt.Errorf("failed to delete role: %w", err)
	}

	if result.DeleteRole.Error != "" {
		return fmt.Errorf("failed to delete role: %s", result.DeleteRole.Error)
	}

	return nil
}

const createAPIKeyMutation = `
mutation CreateAPIKey($input: CreateAPIKeyInput!) {
  createApiKey(input: $input) {
    apiKey {
      id
      name
      key
      permissions
      createdAt
      expiresAt
    }
    error
  }
}
`

func (c *Client) CreateAPIKey(ctx context.Context, input *CreateAPIKeyInput) (*APIKey, error) {
	var result struct {
		CreateAPIKey struct {
			APIKey *APIKey `json:"apiKey"`
			Error  string  `json:"error"`
		} `json:"createApiKey"`
	}

	variables := map[string]interface{}{
		"input": input,
	}

	if err := c.execute(ctx, createAPIKeyMutation, variables, &result); err != nil {
		return nil, fmt.Errorf("failed to create API key: %w", err)
	}

	if result.CreateAPIKey.Error != "" {
		return nil, fmt.Errorf("failed to create API key: %s", result.CreateAPIKey.Error)
	}

	return result.CreateAPIKey.APIKey, nil
}

const deleteAPIKeyMutation = `
mutation DeleteAPIKey($id: ID!) {
  deleteApiKey(id: $id) {
    success
    error
  }
}
`

func (c *Client) DeleteAPIKey(ctx context.Context, keyID string) error {
	var result struct {
		DeleteAPIKey struct {
			Success bool   `json:"success"`
			Error   string `json:"error"`
		} `json:"deleteApiKey"`
	}

	variables := map[string]interface{}{
		"id": keyID,
	}

	if err := c.execute(ctx, deleteAPIKeyMutation, variables, &result); err != nil {
		return fmt.Errorf("failed to delete API key: %w", err)
	}

	if result.DeleteAPIKey.Error != "" {
		return fmt.Errorf("failed to delete API key: %s", result.DeleteAPIKey.Error)
	}

	return nil
}
