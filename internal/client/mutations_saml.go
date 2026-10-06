package client

import (
	"context"
	"fmt"
)

const createSAMLIdPMutation = `
mutation CreateSAMLIdP($input: CreateSAMLIdPInput!) {
  createSAMLIdP(input: $input) {
    samlIdP {
      id
      metadataUrl
      metadataXml
      enforceSaml
      syncUserRoles
      defaultOrgId
      defaultOrgRoleId
      defaultSpaceId
      defaultSpaceRoleId
      emailDomainsList
      allowLoginWithDefaults
      createdAt
      updatedAt
      roleMappings {
        id
        attributesMap
        isAccountAdmin
        orgRole
        spaceRolesMap
        spaceRbacRolesMap
      }
    }
  }
}
`

func (c *Client) CreateSAMLIdP(ctx context.Context, input *CreateSAMLIdPInput) (*SAMLIdP, error) {
	var result struct {
		CreateSAMLIdP struct {
			SAMLIdP *SAMLIdP `json:"samlIdP"`
		} `json:"createSAMLIdP"`
	}

	variables := map[string]interface{}{
		"input": input,
	}

	if err := c.execute(ctx, createSAMLIdPMutation, variables, &result); err != nil {
		return nil, fmt.Errorf("failed to create SAML IdP: %w", err)
	}

	if result.CreateSAMLIdP.SAMLIdP == nil {
		return nil, fmt.Errorf("no SAML IdP returned from create")
	}

	return result.CreateSAMLIdP.SAMLIdP, nil
}

const updateSAMLIdPMutation = `
mutation UpdateSAMLIdP($input: UpdateSAMLIdPInput!) {
  updateSAMLIdP(input: $input) {
    samlIdP {
      id
      metadataUrl
      metadataXml
      enforceSaml
      syncUserRoles
      defaultOrgId
      defaultOrgRoleId
      defaultSpaceId
      defaultSpaceRoleId
      emailDomainsList
      allowLoginWithDefaults
      createdAt
      updatedAt
      roleMappings {
        id
        attributesMap
        isAccountAdmin
        orgRole
        spaceRolesMap
        spaceRbacRolesMap
      }
    }
  }
}
`

func (c *Client) UpdateSAMLIdP(ctx context.Context, input *UpdateSAMLIdPInput) (*SAMLIdP, error) {
	var result struct {
		UpdateSAMLIdP struct {
			SAMLIdP *SAMLIdP `json:"samlIdP"`
		} `json:"updateSAMLIdP"`
	}

	variables := map[string]interface{}{
		"input": input,
	}

	if err := c.execute(ctx, updateSAMLIdPMutation, variables, &result); err != nil {
		return nil, fmt.Errorf("failed to update SAML IdP: %w", err)
	}

	if result.UpdateSAMLIdP.SAMLIdP == nil {
		return nil, fmt.Errorf("no SAML IdP returned from update")
	}

	return result.UpdateSAMLIdP.SAMLIdP, nil
}

const getSAMLIdPQuery = `
query GetSAMLIdP($id: ID!) {
  node(id: $id) {
    __typename
    ... on SAMLIdP {
      id
      metadataUrl
      metadataXml
      enforceSaml
      syncUserRoles
      defaultOrgId
      defaultOrgRoleId
      defaultSpaceId
      defaultSpaceRoleId
      emailDomainsList
      allowLoginWithDefaults
      createdAt
      updatedAt
      roleMappings {
        id
        attributesMap
        isAccountAdmin
        orgRole
        spaceRolesMap
        spaceRbacRolesMap
      }
    }
  }
}
`

func (c *Client) GetSAMLIdP(ctx context.Context, samlIdPID string) (*SAMLIdP, error) {
	var result struct {
		Node *SAMLIdP `json:"node"`
	}

	variables := map[string]interface{}{
		"id": samlIdPID,
	}

	if err := c.execute(ctx, getSAMLIdPQuery, variables, &result); err != nil {
		return nil, fmt.Errorf("failed to get SAML IdP: %w", err)
	}

	if result.Node == nil {
		return nil, fmt.Errorf("SAML IdP not found")
	}

	return result.Node, nil
}

const listSAMLIdPsQuery = `
query ListSAMLIdPs($first: Int, $after: String) {
  account {
    samlIdPs(first: $first, after: $after) {
      edges {
        node {
          id
          metadataUrl
          metadataXml
          enforceSaml
          syncUserRoles
          defaultOrgId
          defaultOrgRoleId
          defaultSpaceId
          defaultSpaceRoleId
          emailDomainsList
          allowLoginWithDefaults
          createdAt
          updatedAt
        }
      }
    }
  }
}
`

func (c *Client) ListSAMLIdPs(ctx context.Context, first int, after string) ([]*SAMLIdP, error) {
	var result struct {
		Account struct {
			SAMLIdPs struct {
				Edges []struct {
					Node *SAMLIdP `json:"node"`
				} `json:"edges"`
			} `json:"samlIdPs"`
		} `json:"account"`
	}

	variables := map[string]interface{}{
		"first": first,
	}
	if after != "" {
		variables["after"] = after
	}

	if err := c.execute(ctx, listSAMLIdPsQuery, variables, &result); err != nil {
		return nil, fmt.Errorf("failed to list SAML IdPs: %w", err)
	}

	samlIdPs := make([]*SAMLIdP, 0)
	for _, edge := range result.Account.SAMLIdPs.Edges {
		if edge.Node != nil {
			samlIdPs = append(samlIdPs, edge.Node)
		}
	}

	return samlIdPs, nil
}
