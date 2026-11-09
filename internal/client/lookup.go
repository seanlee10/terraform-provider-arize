package client

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
)

// NamedID is a resource's ID and display name, used to resolve names to IDs.
type NamedID struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

const listOrganizationsQuery = `
query ListOrganizations($after: String) {
  account {
    organizations(first: 50, after: $after) {
      pageInfo { hasNextPage endCursor }
      edges { node { id name } }
    }
  }
}
`

// ListOrganizations returns every organization in the API key's account.
func (c *Client) ListOrganizations(ctx context.Context) ([]NamedID, error) {
	var out []NamedID
	var after interface{}
	for {
		var result struct {
			Account struct {
				Organizations struct {
					PageInfo struct {
						HasNextPage bool   `json:"hasNextPage"`
						EndCursor   string `json:"endCursor"`
					} `json:"pageInfo"`
					Edges []struct {
						Node NamedID `json:"node"`
					} `json:"edges"`
				} `json:"organizations"`
			} `json:"account"`
		}
		if err := c.execute(ctx, listOrganizationsQuery, map[string]interface{}{"after": after}, &result); err != nil {
			return nil, fmt.Errorf("failed to list organizations: %w", err)
		}
		orgs := result.Account.Organizations
		for _, e := range orgs.Edges {
			out = append(out, e.Node)
		}
		if !orgs.PageInfo.HasNextPage || orgs.PageInfo.EndCursor == "" {
			return out, nil
		}
		after = orgs.PageInfo.EndCursor
	}
}

// ListAIIntegrations returns the LLM integrations whose name contains name
// (case-insensitive; "" for all). A non-empty spaceID limits the list to
// integrations available to that space.
func (c *RESTClient) ListAIIntegrations(ctx context.Context, name, spaceID string) ([]NamedID, error) {
	var out []NamedID
	cursor := ""
	for {
		q := url.Values{"limit": {"100"}}
		if name != "" {
			q.Set("name", name)
		}
		if spaceID != "" {
			q.Set("space_id", spaceID)
		}
		if cursor != "" {
			q.Set("cursor", cursor)
		}
		var page struct {
			Integrations []NamedID `json:"ai_integrations"`
			Pagination   struct {
				HasMore    bool   `json:"has_more"`
				NextCursor string `json:"next_cursor"`
			} `json:"pagination"`
		}
		if err := c.do(ctx, http.MethodGet, "/v2/ai-integrations?"+q.Encode(), nil, &page); err != nil {
			return nil, fmt.Errorf("failed to list AI integrations: %w", err)
		}
		out = append(out, page.Integrations...)
		if !page.Pagination.HasMore || page.Pagination.NextCursor == "" {
			return out, nil
		}
		cursor = page.Pagination.NextCursor
	}
}
