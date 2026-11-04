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

// UpdateSpaceInput is the REST PATCH /v2/spaces/{id} body. Every field is
// sent: a nil Description is sent as null, which clears it.
type UpdateSpaceInput struct {
	Name        string  `json:"name"`
	Description *string `json:"description"`
	IsPrivate   bool    `json:"is_private"`
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
