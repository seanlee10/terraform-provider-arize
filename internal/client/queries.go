package client

import (
	"context"
	"fmt"
)

const getUserQuery = `
query GetUser($id: ID!) {
  node(id: $id) {
    __typename
    ... on User {
      id
      email
      name
      status
      userType
      createdAt
    }
  }
}
`

func (c *Client) GetUser(ctx context.Context, userID string) (*User, error) {
	var result struct {
		Node *User `json:"node"`
	}

	variables := map[string]interface{}{
		"id": userID,
	}

	if err := c.execute(ctx, getUserQuery, variables, &result); err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	if result.Node == nil {
		return nil, fmt.Errorf("user not found")
	}

	return result.Node, nil
}

const listUsersQuery = `
query ListUsers($first: Int, $after: String) {
  account {
    users(first: $first, after: $after) {
      edges {
        node {
          id
          email
          name
          status
          userType
          createdAt
        }
      }
    }
  }
}
`

func (c *Client) ListUsers(ctx context.Context, first int, after string) ([]*User, error) {
	var result struct {
		Account struct {
			Users struct {
				Edges []struct {
					Node *User `json:"node"`
				} `json:"edges"`
			} `json:"users"`
		} `json:"account"`
	}

	variables := map[string]interface{}{
		"first": first,
	}
	if after != "" {
		variables["after"] = after
	}

	if err := c.execute(ctx, listUsersQuery, variables, &result); err != nil {
		return nil, fmt.Errorf("failed to list users: %w", err)
	}

	users := make([]*User, 0)
	for _, edge := range result.Account.Users.Edges {
		if edge.Node != nil {
			users = append(users, edge.Node)
		}
	}

	return users, nil
}

const getRoleQuery = `
query GetRole($id: ID!) {
  role(id: $id) {
    id
    name
    description
    permissions
    createdAt
    updatedAt
  }
}
`

func (c *Client) GetRole(ctx context.Context, roleID string) (*Role, error) {
	var result struct {
		Role *Role `json:"role"`
	}

	variables := map[string]interface{}{
		"id": roleID,
	}

	if err := c.execute(ctx, getRoleQuery, variables, &result); err != nil {
		return nil, fmt.Errorf("failed to get role: %w", err)
	}

	if result.Role == nil {
		return nil, fmt.Errorf("role not found")
	}

	return result.Role, nil
}

const listRolesQuery = `
query ListRoles($first: Int, $after: String) {
  roles(first: $first, after: $after) {
    edges {
      node {
        id
        name
        description
        permissions
        createdAt
        updatedAt
      }
    }
  }
}
`

func (c *Client) ListRoles(ctx context.Context, first int, after string) ([]*Role, error) {
	var result struct {
		Roles struct {
			Edges []struct {
				Node *Role `json:"node"`
			} `json:"edges"`
		} `json:"roles"`
	}

	variables := map[string]interface{}{
		"first": first,
	}
	if after != "" {
		variables["after"] = after
	}

	if err := c.execute(ctx, listRolesQuery, variables, &result); err != nil {
		return nil, fmt.Errorf("failed to list roles: %w", err)
	}

	roles := make([]*Role, 0)
	for _, edge := range result.Roles.Edges {
		if edge.Node != nil {
			roles = append(roles, edge.Node)
		}
	}

	return roles, nil
}

const getAPIKeyQuery = `
query GetAPIKey($id: ID!) {
  apiKey(id: $id) {
    id
    name
    permissions
    lastUsedAt
    createdAt
    expiresAt
  }
}
`

func (c *Client) GetAPIKey(ctx context.Context, keyID string) (*APIKey, error) {
	var result struct {
		APIKey *APIKey `json:"apiKey"`
	}

	variables := map[string]interface{}{
		"id": keyID,
	}

	if err := c.execute(ctx, getAPIKeyQuery, variables, &result); err != nil {
		return nil, fmt.Errorf("failed to get API key: %w", err)
	}

	if result.APIKey == nil {
		return nil, fmt.Errorf("API key not found")
	}

	return result.APIKey, nil
}

const listAPIKeysQuery = `
query ListAPIKeys($first: Int, $after: String) {
  apiKeys(first: $first, after: $after) {
    edges {
      node {
        id
        name
        permissions
        lastUsedAt
        createdAt
        expiresAt
      }
    }
  }
}
`

func (c *Client) ListAPIKeys(ctx context.Context, first int, after string) ([]*APIKey, error) {
	var result struct {
		APIKeys struct {
			Edges []struct {
				Node *APIKey `json:"node"`
			} `json:"edges"`
		} `json:"apiKeys"`
	}

	variables := map[string]interface{}{
		"first": first,
	}
	if after != "" {
		variables["after"] = after
	}

	if err := c.execute(ctx, listAPIKeysQuery, variables, &result); err != nil {
		return nil, fmt.Errorf("failed to list API keys: %w", err)
	}

	apiKeys := make([]*APIKey, 0)
	for _, edge := range result.APIKeys.Edges {
		if edge.Node != nil {
			apiKeys = append(apiKeys, edge.Node)
		}
	}

	return apiKeys, nil
}
