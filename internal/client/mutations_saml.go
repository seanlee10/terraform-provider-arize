package client

import (
	"context"
	"errors"
	"fmt"
)

var ErrSAMLIdPNotFound = errors.New("SAML IdP not found")

const samlIdPFields = `
      id
      metadataUrl
      metadataXml
      enforceSaml
      syncUserRoles
      signAuthn
      defaultOrgId
      defaultOrgRoleId
      defaultSpaceId
      defaultSpaceRoleId
      emailDomainsList { domain }
      allowLoginWithDefaults
      roleMappings {
        attributesMap
        isAccountAdmin
        accountRole
        orgRole { orgId roleId }
        spaceRolesMap
        spaceRbacRolesMap
        projectRolesMap
        projectRolesManaged
      }
      createdAt
      updatedAt
`

const createSAMLIdPMutation = `
mutation CreateSAMLIdP($input: CreateSAMLIdPInput!) {
  createSAMLIdP(input: $input) {
    error
    idp {` + samlIdPFields + `}
  }
}
`

func (c *Client) CreateSAMLIdP(ctx context.Context, input *SAMLIdPInput) (*SAMLIdP, error) {
	var result struct {
		CreateSAMLIdP struct {
			Error string   `json:"error"`
			IdP   *SAMLIdP `json:"idp"`
		} `json:"createSAMLIdP"`
	}

	if err := c.execute(ctx, createSAMLIdPMutation, map[string]interface{}{"input": input}, &result); err != nil {
		return nil, fmt.Errorf("failed to create SAML IdP: %w", err)
	}
	if result.CreateSAMLIdP.Error != "" {
		return nil, fmt.Errorf("failed to create SAML IdP: %s", result.CreateSAMLIdP.Error)
	}
	if result.CreateSAMLIdP.IdP == nil {
		return nil, fmt.Errorf("no SAML IdP returned from create")
	}

	return result.CreateSAMLIdP.IdP, nil
}

const updateSAMLIdPMutation = `
mutation UpdateSAMLIdP($input: UpdateSAMLIdPInput!) {
  updateSAMLIdP(input: $input) {
    error
    idp {` + samlIdPFields + `}
  }
}
`

func (c *Client) UpdateSAMLIdP(ctx context.Context, input *SAMLIdPInput) (*SAMLIdP, error) {
	var result struct {
		UpdateSAMLIdP struct {
			Error string   `json:"error"`
			IdP   *SAMLIdP `json:"idp"`
		} `json:"updateSAMLIdP"`
	}

	if err := c.execute(ctx, updateSAMLIdPMutation, map[string]interface{}{"input": input}, &result); err != nil {
		return nil, fmt.Errorf("failed to update SAML IdP: %w", err)
	}
	if result.UpdateSAMLIdP.Error != "" {
		return nil, fmt.Errorf("failed to update SAML IdP: %s", result.UpdateSAMLIdP.Error)
	}
	if result.UpdateSAMLIdP.IdP == nil {
		return nil, fmt.Errorf("no SAML IdP returned from update")
	}

	return result.UpdateSAMLIdP.IdP, nil
}

const listSAMLIdPsQuery = `
query ListSAMLIdPs($after: String) {
  account {
    samlIdPs(first: 50, after: $after) {
      pageInfo { hasNextPage endCursor }
      edges {
        node {` + samlIdPFields + `}
      }
    }
  }
}
`

// ListSAMLIdPs returns the account's active SAML IdPs.
func (c *Client) ListSAMLIdPs(ctx context.Context) ([]*SAMLIdP, error) {
	var idps []*SAMLIdP
	after := ""
	for {
		var result struct {
			Account struct {
				SAMLIdPs struct {
					PageInfo struct {
						HasNextPage bool   `json:"hasNextPage"`
						EndCursor   string `json:"endCursor"`
					} `json:"pageInfo"`
					Edges []struct {
						Node *SAMLIdP `json:"node"`
					} `json:"edges"`
				} `json:"samlIdPs"`
			} `json:"account"`
		}

		variables := map[string]interface{}{}
		if after != "" {
			variables["after"] = after
		}
		if err := c.execute(ctx, listSAMLIdPsQuery, variables, &result); err != nil {
			return nil, fmt.Errorf("failed to list SAML IdPs: %w", err)
		}

		for _, edge := range result.Account.SAMLIdPs.Edges {
			if edge.Node != nil {
				idps = append(idps, edge.Node)
			}
		}
		page := result.Account.SAMLIdPs.PageInfo
		if !page.HasNextPage || page.EndCursor == "" {
			return idps, nil
		}
		after = page.EndCursor
	}
}

// GetSAMLIdP looks the IdP up in the account listing, so an IdP that was
// removed in the Arize UI reports ErrSAMLIdPNotFound.
func (c *Client) GetSAMLIdP(ctx context.Context, samlIdPID string) (*SAMLIdP, error) {
	idps, err := c.ListSAMLIdPs(ctx)
	if err != nil {
		return nil, err
	}
	for _, idp := range idps {
		if idp.ID == samlIdPID {
			return idp, nil
		}
	}
	return nil, ErrSAMLIdPNotFound
}
