package client

import (
	"context"
	"errors"
	"fmt"
)

const createSpaceMutation = `
mutation CreateSpace($input: CreateSpaceMutationInput!) {
  createSpace(input: $input) {
    space {
      id
      name
      uuid
      description
      createdAt
      private
      organization {
        id
        name
      }
    }
  }
}
`

func (c *Client) CreateSpace(ctx context.Context, input *CreateSpaceInput) (*Space, error) {
	var result struct {
		CreateSpace struct {
			Space *Space `json:"space"`
		} `json:"createSpace"`
	}

	variables := map[string]interface{}{
		"input": input,
	}

	if err := c.execute(ctx, createSpaceMutation, variables, &result); err != nil {
		return nil, fmt.Errorf("failed to create space: %w", err)
	}

	return result.CreateSpace.Space, nil
}

const getSpaceQuery = `
query GetSpace($id: ID!) {
  node(id: $id) {
    __typename
    ... on Space {
      id
      name
      uuid
      description
      createdAt
      private
      organization {
        id
        name
      }
    }
  }
}
`

func (c *Client) GetSpace(ctx context.Context, spaceID string) (*Space, error) {
	var result struct {
		Node *Space `json:"node"`
	}

	variables := map[string]interface{}{
		"id": spaceID,
	}

	if err := c.execute(ctx, getSpaceQuery, variables, &result); err != nil {
		// A deleted space is a NOT_FOUND error, not a null node.
		var gqlErr *GraphQLError
		if errors.As(err, &gqlErr) && gqlErr.Extensions.Code == "NOT_FOUND" {
			return nil, ErrSpaceNotFound
		}
		return nil, fmt.Errorf("failed to get space: %w", err)
	}

	if result.Node == nil {
		return nil, ErrSpaceNotFound
	}

	return result.Node, nil
}

const assignSpaceMembershipMutation = `
mutation AssignSpaceMembership($input: AssignSpaceMembershipMutationInput!) {
  assignSpaceMembership(input: $input) {
    spaceMemberships {
      id
      user {
        id
        email
        name
      }
    }
  }
}
`

func (c *Client) AssignSpaceMembership(ctx context.Context, input *AssignSpaceMembershipInput) ([]*SpaceMember, error) {
	var result struct {
		AssignSpaceMembership struct {
			SpaceMemberships []*SpaceMember `json:"spaceMemberships"`
		} `json:"assignSpaceMembership"`
	}

	variables := map[string]interface{}{
		"input": input,
	}

	if err := c.execute(ctx, assignSpaceMembershipMutation, variables, &result); err != nil {
		return nil, fmt.Errorf("failed to assign space membership: %w", err)
	}

	return result.AssignSpaceMembership.SpaceMemberships, nil
}
