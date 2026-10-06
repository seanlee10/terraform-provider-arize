package client

type Space struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	UUID          string `json:"uuid"`
	Description   string `json:"description,omitempty"`
	CreatedAt     string `json:"createdAt"`
	Organization  *Org   `json:"organization,omitempty"`
	Private       bool   `json:"private"`
}

type Org struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type CreateSpaceInput struct {
	Name                   string `json:"name"`
	AccountOrganizationID  string `json:"accountOrganizationId"`
	Private                bool   `json:"private"`
	Description            string `json:"description,omitempty"`
}

type UpdateSpaceInput struct {
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
	Private     bool   `json:"private,omitempty"`
}

type SpaceMember struct {
	ID     string `json:"id"`
	UserID string `json:"userId"`
	SpaceID string `json:"spaceId"`
	Role   string `json:"role"`
	User   *User  `json:"user,omitempty"`
}

type SpaceMemberInput struct {
	UserID  string `json:"userId"`
	SpaceID string `json:"spaceId"`
	Role    string `json:"role"`
}

type AssignSpaceMembershipInput struct {
	SpaceMemberships []SpaceMemberInput `json:"spaceMemberships"`
}
