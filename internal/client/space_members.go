package client

import (
	"context"
	"errors"
	"fmt"
)

var (
	ErrSpaceMemberNotFound = errors.New("space member not found")
	ErrSpaceNotFound       = errors.New("space not found")
)

// SpaceUser is an entry in Space.spaceUsers. Account admins are listed with
// membership ACCOUNT_ADMIN and no role whether or not they were also added
// explicitly, so only EXPLICIT_MEMBERSHIP entries show an explicit role.
type SpaceUser struct {
	Role       *string `json:"role"`
	Membership string  `json:"membership"`
	User       User    `json:"user"`
}

func (s *SpaceUser) IsExplicit() bool {
	return s.Membership == "EXPLICIT_MEMBERSHIP"
}

const spaceUsersQuery = `
query SpaceUsers($id: ID!, $after: String) {
  node(id: $id) {
    ... on Space {
      spaceUsers(first: 50, after: $after) {
        pageInfo { hasNextPage endCursor }
        edges { node { role membership user { id email name } } }
      }
    }
  }
}
`

// GetSpaceUser returns the user's entry in the space, or ErrSpaceMemberNotFound
// if the user isn't listed or the space no longer exists.
func (c *Client) GetSpaceUser(ctx context.Context, spaceID, userID string) (*SpaceUser, error) {
	var after interface{}
	for {
		var result struct {
			Node *struct {
				SpaceUsers struct {
					PageInfo struct {
						HasNextPage bool   `json:"hasNextPage"`
						EndCursor   string `json:"endCursor"`
					} `json:"pageInfo"`
					Edges []struct {
						Node SpaceUser `json:"node"`
					} `json:"edges"`
				} `json:"spaceUsers"`
			} `json:"node"`
		}
		vars := map[string]interface{}{"id": spaceID, "after": after}
		if err := c.execute(ctx, spaceUsersQuery, vars, &result); err != nil {
			// A deleted space is a NOT_FOUND error, not a null node.
			var gqlErr *GraphQLError
			if errors.As(err, &gqlErr) && gqlErr.Extensions.Code == "NOT_FOUND" {
				return nil, ErrSpaceMemberNotFound
			}
			return nil, fmt.Errorf("failed to list space users: %w", err)
		}
		if result.Node == nil {
			return nil, ErrSpaceMemberNotFound
		}
		for _, e := range result.Node.SpaceUsers.Edges {
			if e.Node.User.ID == userID {
				su := e.Node
				return &su, nil
			}
		}
		page := result.Node.SpaceUsers.PageInfo
		if !page.HasNextPage || page.EndCursor == "" {
			return nil, ErrSpaceMemberNotFound
		}
		after = page.EndCursor
	}
}

const removeSpaceMemberMutation = `
mutation RemoveSpaceMember($input: RemoveSpaceMemberMutationInput!) {
  removeSpaceMember(input: $input) {
    space { id }
  }
}
`

// RemoveSpaceMember removes the user's explicit membership. A membership that
// is already gone is not an error.
func (c *Client) RemoveSpaceMember(ctx context.Context, spaceID, userID string) error {
	var result struct{}
	vars := map[string]interface{}{"input": map[string]string{"spaceId": spaceID, "userId": userID}}
	if err := c.execute(ctx, removeSpaceMemberMutation, vars, &result); err != nil {
		var gqlErr *GraphQLError
		if errors.As(err, &gqlErr) && gqlErr.Extensions.Code == "NOT_FOUND" {
			return nil
		}
		return fmt.Errorf("failed to remove space member: %w", err)
	}
	return nil
}
