package provider

import (
	"strings"
	"testing"

	"github.com/arize-ai/terraform-provider-arize/internal/client"
)

var testAccess = &client.OrganizationAccess{
	UserEmail: "jane@x.com", AccountRole: "member",
	Organizations: []client.OrganizationRole{
		{ID: "o1", Name: "Main", Role: "member", Membership: "EXPLICIT_MEMBERSHIP"},
		{ID: "o2", Name: "Team", Role: "admin", Membership: "EXPLICIT_MEMBERSHIP"},
		{ID: "o3", Name: "Other"},
	},
}

func TestSpaceCreateUnauthorizedDetail_DefaultOrg(t *testing.T) {
	got := spaceCreateUnauthorizedDetail("o1", "", testAccess)

	for _, want := range []string{
		`organization "Main" (o1)`,
		"organization_id and organization_name aren't set",
		"jane@x.com (account role: member)",
		`role "member" in that organization`,
		`- "Team" (o2): admin`,
		`- "Other" (o3): not a member`,
		"Set organization_id",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("detail is missing %q:\n%s", want, got)
		}
	}
}

func TestSpaceCreateUnauthorizedDetail_ConfiguredOrgNotVisible(t *testing.T) {
	got := spaceCreateUnauthorizedDetail("zzz", "organization_id", testAccess)

	for _, want := range []string{"organization zzz", "set by organization_id", "isn't one of the organizations this API key can see"} {
		if !strings.Contains(got, want) {
			t.Errorf("detail is missing %q:\n%s", want, got)
		}
	}
}

func TestSpaceCreateUnauthorizedDetail_AccountAdmin(t *testing.T) {
	access := &client.OrganizationAccess{UserEmail: "a@x.com", AccountRole: "admin",
		Organizations: []client.OrganizationRole{{ID: "o1", Name: "Main", Membership: "ACCOUNT_ADMIN"}}}

	got := spaceCreateUnauthorizedDetail("o1", "organization_name", access)

	if !strings.Contains(got, "an account admin") {
		t.Errorf("detail should say the user is an account admin:\n%s", got)
	}
}
