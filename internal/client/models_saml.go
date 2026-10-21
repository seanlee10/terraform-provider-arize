package client

type SAMLIdP struct {
	ID                     string        `json:"id"`
	MetadataURL            *string       `json:"metadataUrl"`
	MetadataXML            *string       `json:"metadataXml"`
	EnforceSAML            bool          `json:"enforceSaml"`
	SyncUserRoles          bool          `json:"syncUserRoles"`
	SignAuthn              bool          `json:"signAuthn"`
	DefaultOrgID           *string       `json:"defaultOrgId"`
	DefaultOrgRoleID       *string       `json:"defaultOrgRoleId"`
	DefaultSpaceID         *string       `json:"defaultSpaceId"`
	DefaultSpaceRoleID     *string       `json:"defaultSpaceRoleId"`
	EmailDomainsList       []EmailDomain `json:"emailDomainsList"`
	AllowLoginWithDefaults bool          `json:"allowLoginWithDefaults"`
	RoleMappings           []RoleMapping `json:"roleMappings"`
	CreatedAt              string        `json:"createdAt"`
	UpdatedAt              string        `json:"updatedAt"`
}

type EmailDomain struct {
	Domain string `json:"domain"`
}

func (s *SAMLIdP) Domains() []string {
	domains := make([]string, 0, len(s.EmailDomainsList))
	for _, d := range s.EmailDomainsList {
		domains = append(domains, d.Domain)
	}
	return domains
}

// RoleMapping assigns roles to SAML users whose assertion attributes match
// AttributesMap. The *Map fields are [key, value] tuples on the wire.
type RoleMapping struct {
	AttributesMap       [][]string `json:"attributesMap"`
	IsAccountAdmin      *bool      `json:"isAccountAdmin,omitempty"`
	AccountRole         *string    `json:"accountRole,omitempty"`
	OrgRole             *OrgRole   `json:"orgRole,omitempty"`
	SpaceRolesMap       [][]string `json:"spaceRolesMap"`
	SpaceRbacRolesMap   [][]string `json:"spaceRbacRolesMap"`
	ProjectRolesMap     [][]string `json:"projectRolesMap"`
	ProjectRolesManaged bool       `json:"projectRolesManaged"`
}

type OrgRole struct {
	OrgID  string `json:"orgId"`
	RoleID string `json:"roleId"`
}

type RoleMappingsInput struct {
	MappingsList []RoleMapping `json:"mappingsList"`
}

// SAMLIdPInput is shared by createSAMLIdP and updateSAMLIdP; ID is set only
// for updates. updateSAMLIdP replaces the whole IdP, so omitted fields reset.
type SAMLIdPInput struct {
	ID                     string             `json:"id,omitempty"`
	MetadataURL            *string            `json:"metadataUrl,omitempty"`
	MetadataXML            *string            `json:"metadataXml,omitempty"`
	EnforceSAML            *bool              `json:"enforceSaml,omitempty"`
	SyncUserRoles          *bool              `json:"syncUserRoles,omitempty"`
	SignAuthn              *bool              `json:"signAuthn,omitempty"`
	DefaultOrgID           *string            `json:"defaultOrgId,omitempty"`
	DefaultOrgRoleID       *string            `json:"defaultOrgRoleId,omitempty"`
	DefaultSpaceID         *string            `json:"defaultSpaceId,omitempty"`
	DefaultSpaceRoleID     *string            `json:"defaultSpaceRoleId,omitempty"`
	EmailDomainsList       []EmailDomain      `json:"emailDomainsList"`
	AllowLoginWithDefaults *bool              `json:"allowLoginWithDefaults,omitempty"`
	RoleMappings           *RoleMappingsInput `json:"roleMappings"`
}
