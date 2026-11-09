package client

import (
	"context"
	"errors"
	"fmt"
)

// OrganizationAccess describes the API key's user and their role in each
// organization the key can see. It explains authorization failures.
type OrganizationAccess struct {
	UserEmail     string
	AccountRole   string
	Organizations []OrganizationRole
}

// OrganizationRole is the user's place in one organization. Role and
// Membership are empty when the user isn't a member; account admins have
// Membership ACCOUNT_ADMIN and no role.
type OrganizationRole struct {
	ID, Name   string
	Role       string
	Membership string
}

const viewerQuery = `query { viewer { user { id email accountRole } } }`

const organizationRolesQuery = `
query OrganizationRoles($email: String) {
  account {
    organizations(first: 50) {
      edges { node {
        id
        name
        accountOrganizationUsers(first: 10, search: $email) {
          edges { node { role membership user { id } } }
        }
      } }
    }
  }
}
`

func (c *Client) GetOrganizationAccess(ctx context.Context) (*OrganizationAccess, error) {
	var viewer struct {
		Viewer struct {
			User struct {
				ID          string `json:"id"`
				Email       string `json:"email"`
				AccountRole string `json:"accountRole"`
			} `json:"user"`
		} `json:"viewer"`
	}
	if err := c.execute(ctx, viewerQuery, nil, &viewer); err != nil {
		return nil, fmt.Errorf("failed to get the API key's user: %w", err)
	}
	u := viewer.Viewer.User

	var orgs struct {
		Account struct {
			Organizations struct {
				Edges []struct {
					Node struct {
						ID    string `json:"id"`
						Name  string `json:"name"`
						Users struct {
							Edges []struct {
								Node struct {
									Role       *string `json:"role"`
									Membership string  `json:"membership"`
									User       struct {
										ID string `json:"id"`
									} `json:"user"`
								} `json:"node"`
							} `json:"edges"`
						} `json:"accountOrganizationUsers"`
					} `json:"node"`
				} `json:"edges"`
			} `json:"organizations"`
		} `json:"account"`
	}
	if err := c.execute(ctx, organizationRolesQuery, map[string]interface{}{"email": u.Email}, &orgs); err != nil {
		return nil, fmt.Errorf("failed to list organization roles: %w", err)
	}

	access := &OrganizationAccess{UserEmail: u.Email, AccountRole: u.AccountRole}
	for _, e := range orgs.Account.Organizations.Edges {
		or := OrganizationRole{ID: e.Node.ID, Name: e.Node.Name}
		// The search is a partial match: pick the viewer's own entry.
		for _, ue := range e.Node.Users.Edges {
			if ue.Node.User.ID == u.ID {
				or.Membership = ue.Node.Membership
				if ue.Node.Role != nil {
					or.Role = *ue.Node.Role
				}
			}
		}
		access.Organizations = append(access.Organizations, or)
	}
	return access, nil
}

// IsUnauthorized reports whether err is Arize's "Not Authorized" GraphQL error.
func IsUnauthorized(err error) bool {
	var gqlErr *GraphQLError
	return errors.As(err, &gqlErr) && gqlErr.Extensions.Code == "UNAUTHORIZED"
}
