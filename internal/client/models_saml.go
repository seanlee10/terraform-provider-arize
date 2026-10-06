package client

type SAMLIdP struct {
	ID                      string                    `json:"id"`
	MetadataURL             string                    `json:"metadataUrl"`
	MetadataXML             string                    `json:"metadataXml"`
	EnforceSAML             bool                      `json:"enforceSaml"`
	SyncUserRoles           bool                      `json:"syncUserRoles"`
	DefaultOrgID            string                    `json:"defaultOrgId"`
	DefaultOrgRoleID        string                    `json:"defaultOrgRoleId"`
	DefaultSpaceID          string                    `json:"defaultSpaceId"`
	DefaultSpaceRoleID      string                    `json:"defaultSpaceRoleId"`
	EmailDomainsList        []string                  `json:"emailDomainsList"`
	AllowLoginWithDefaults  bool                      `json:"allowLoginWithDefaults"`
	RoleMappings            []*SAMLRoleMapping        `json:"roleMappings"`
	CreatedAt               string                    `json:"createdAt"`
	UpdatedAt               string                    `json:"updatedAt"`
}

type SAMLRoleMapping struct {
	ID                    string            `json:"id"`
	AttributesMap         map[string]string `json:"attributesMap"`
	IsAccountAdmin        bool              `json:"isAccountAdmin"`
	OrgRole               string            `json:"orgRole"`
	SpaceRolesMap         map[string]string `json:"spaceRolesMap"`
	SpaceRbacRolesMap     map[string]string `json:"spaceRbacRolesMap"`
	ProjectRolesMap       map[string]string `json:"projectRolesMap"`
	ProjectRolesManaged   bool              `json:"projectRolesManaged"`
}

type CreateSAMLIdPInput struct {
	MetadataURL            *string   `json:"metadataUrl,omitempty"`
	MetadataXML            *string   `json:"metadataXml,omitempty"`
	EnforceSAML            *bool     `json:"enforceSaml,omitempty"`
	SyncUserRoles          *bool     `json:"syncUserRoles,omitempty"`
	DefaultOrgID           *string   `json:"defaultOrgId,omitempty"`
	DefaultOrgRoleID       *string   `json:"defaultOrgRoleId,omitempty"`
	DefaultSpaceID         *string   `json:"defaultSpaceId,omitempty"`
	DefaultSpaceRoleID     *string   `json:"defaultSpaceRoleId,omitempty"`
	EmailDomainsList       []string  `json:"emailDomainsList"`
	AllowLoginWithDefaults *bool     `json:"allowLoginWithDefaults,omitempty"`
	RoleMappings           []*SAMLRoleMappingInput `json:"roleMappings,omitempty"`
}

type UpdateSAMLIdPInput struct {
	ID                     string  `json:"id"`
	MetadataURL            *string `json:"metadataUrl,omitempty"`
	MetadataXML            *string `json:"metadataXml,omitempty"`
	EnforceSAML            *bool   `json:"enforceSaml,omitempty"`
	SyncUserRoles          *bool   `json:"syncUserRoles,omitempty"`
	DefaultOrgID           *string `json:"defaultOrgId,omitempty"`
	DefaultOrgRoleID       *string `json:"defaultOrgRoleId,omitempty"`
	DefaultSpaceID         *string `json:"defaultSpaceId,omitempty"`
	DefaultSpaceRoleID     *string `json:"defaultSpaceRoleId,omitempty"`
	EmailDomainsList       []string `json:"emailDomainsList,omitempty"`
	AllowLoginWithDefaults *bool   `json:"allowLoginWithDefaults,omitempty"`
	RoleMappings           []*SAMLRoleMappingInput `json:"roleMappings,omitempty"`
}

type SAMLRoleMappingInput struct {
	ID                  *string            `json:"id,omitempty"`
	AttributesMap       map[string]string  `json:"attributesMap"`
	IsAccountAdmin      *bool              `json:"isAccountAdmin,omitempty"`
	OrgRole             *string            `json:"orgRole,omitempty"`
	SpaceRolesMap       map[string]string  `json:"spaceRolesMap,omitempty"`
	SpaceRbacRolesMap   map[string]string  `json:"spaceRbacRolesMap,omitempty"`
	ProjectRolesMap     map[string]string  `json:"projectRolesMap,omitempty"`
	ProjectRolesManaged *bool              `json:"projectRolesManaged,omitempty"`
}
