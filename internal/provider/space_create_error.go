package provider

import (
	"fmt"
	"strings"

	"github.com/arize-ai/terraform-provider-arize/internal/client"
)

// spaceCreateUnauthorizedDetail explains a "Not Authorized" from createSpace:
// which organization was used and why, who the API key belongs to, and their
// role in each organization the key can see. source is the attribute that
// chose the organization, or "" when it was the default.
func spaceCreateUnauthorizedDetail(orgID, source string, access *client.OrganizationAccess) string {
	var b strings.Builder

	how := "set by " + source
	if source == "" {
		how = "used by default because organization_id and organization_name aren't set"
	}
	var target *client.OrganizationRole
	for i := range access.Organizations {
		if access.Organizations[i].ID == orgID {
			target = &access.Organizations[i]
		}
	}
	if target != nil {
		fmt.Fprintf(&b, "Arize refused to create the space in organization %q (%s), %s.\n", target.Name, orgID, how)
	} else {
		fmt.Fprintf(&b, "Arize refused to create the space in organization %s (%s). It isn't one of the organizations this API key can see.\n", orgID, how)
	}

	fmt.Fprintf(&b, "The API key belongs to %s (account role: %s)", access.UserEmail, access.AccountRole)
	if target != nil {
		fmt.Fprintf(&b, ", who %s.\n", describeOrgRole(*target, "in that organization"))
	} else {
		b.WriteString(".\n")
	}

	b.WriteString("\nThis user's roles in the organizations this API key can see:\n")
	for _, o := range access.Organizations {
		fmt.Fprintf(&b, "- %q (%s): %s\n", o.Name, o.ID, shortOrgRole(o))
	}
	b.WriteString("\nSet organization_id to an organization where this user can create spaces, or use an API key from a user who can.")
	return b.String()
}

func describeOrgRole(o client.OrganizationRole, where string) string {
	switch {
	case o.Membership == "ACCOUNT_ADMIN":
		return "is an account admin"
	case o.Role != "":
		return fmt.Sprintf("has role %q %s", o.Role, where)
	default:
		return "isn't a member " + strings.Replace(where, "in ", "of ", 1)
	}
}

func shortOrgRole(o client.OrganizationRole) string {
	switch {
	case o.Membership == "ACCOUNT_ADMIN":
		return "account admin"
	case o.Role != "":
		return o.Role
	default:
		return "not a member"
	}
}
