package provider

import (
	"testing"

	"github.com/arize-ai/terraform-provider-arize/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func strPtr(s string) *string { return &s }

func TestApplySpaceUser_ExplicitMemberRefreshesRole(t *testing.T) {
	data := SpaceMemberResourceModel{Role: types.StringValue("member")}

	applySpaceUser(&data, &client.SpaceUser{
		Role: strPtr("readOnly"), Membership: "EXPLICIT_MEMBERSHIP",
		User: client.User{ID: "u1", Email: "u1@x.com", Name: "U One"},
	})

	if data.Role.ValueString() != "readOnly" || data.Email.ValueString() != "u1@x.com" || data.Name.ValueString() != "U One" {
		t.Fatalf("applySpaceUser() = %+v, want role readOnly, email and name from Arize", data)
	}
}

func TestApplySpaceUser_AccountAdminKeepsState(t *testing.T) {
	data := SpaceMemberResourceModel{Role: types.StringValue("readOnly"), Email: types.StringValue("a@x.com")}

	applySpaceUser(&data, &client.SpaceUser{Membership: "ACCOUNT_ADMIN", User: client.User{ID: "admin", Email: "a@x.com"}})

	if data.Role.ValueString() != "readOnly" {
		t.Fatalf("role = %q, want state kept (Arize hides an admin's explicit role)", data.Role.ValueString())
	}
}
