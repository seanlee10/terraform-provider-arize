package provider

import (
	"testing"

	"github.com/arize-ai/terraform-provider-arize/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestApplyRESTUser_ActiveUserTakesServerValues(t *testing.T) {
	data := UserResourceModel{Name: types.StringValue("Old Name"), UserType: types.StringValue("human")}

	applyRESTUser(&data, &client.RESTUser{
		ID: "u1", Name: "Test User2", Email: "a@example.com", Status: "ACTIVE", CreatedAt: "2025-08-27T23:04:04.003Z",
	})

	if data.ID.ValueString() != "u1" || data.Email.ValueString() != "a@example.com" ||
		data.Name.ValueString() != "Test User2" || data.CreatedAt.ValueString() != "2025-08-27T23:04:04.003Z" {
		t.Fatalf("applyRESTUser() = %+v, want server id, email, name, created_at", data)
	}
	if data.Status.ValueString() != "active" {
		t.Fatalf("status = %q, want lowercase \"active\" (same as the GraphQL value in existing state)", data.Status.ValueString())
	}
}

func TestApplyRESTUser_InvitedUserKeepsConfiguredName(t *testing.T) {
	data := UserResourceModel{Name: types.StringValue("TF Test User")}

	// For invitations the API reports the email as the name.
	applyRESTUser(&data, &client.RESTUser{ID: "inv1", Name: "tf@example.com", Email: "tf@example.com", Status: "INVITED"})

	if data.Name.ValueString() != "TF Test User" {
		t.Fatalf("name = %q, want configured name kept while invited", data.Name.ValueString())
	}
	if data.Status.ValueString() != "invited" {
		t.Fatalf("status = %q, want \"invited\"", data.Status.ValueString())
	}
}

func TestApplyRESTUser_UserTypeKeptOrDefaultsToHuman(t *testing.T) {
	kept := UserResourceModel{UserType: types.StringValue("bot")}
	applyRESTUser(&kept, &client.RESTUser{ID: "u1", Status: "ACTIVE"})
	if kept.UserType.ValueString() != "bot" {
		t.Fatalf("user_type = %q, want existing value kept", kept.UserType.ValueString())
	}

	fresh := UserResourceModel{UserType: types.StringNull()}
	applyRESTUser(&fresh, &client.RESTUser{ID: "u1", Status: "INVITED"})
	if fresh.UserType.ValueString() != "human" {
		t.Fatalf("user_type = %q, want \"human\" for users created through the API", fresh.UserType.ValueString())
	}
}

func TestApplyRESTUser_KeepsConfiguredEmailCase(t *testing.T) {
	data := UserResourceModel{Email: types.StringValue("Alice@Example.com")}

	applyRESTUser(&data, &client.RESTUser{ID: "u1", Email: "alice@example.com", Status: "ACTIVE"})

	if data.Email.ValueString() != "Alice@Example.com" {
		t.Fatalf("email = %q, want configured case kept (Arize stores emails lowercased)", data.Email.ValueString())
	}
}

func TestApplyRESTUser_TakesServerEmailWhenDifferent(t *testing.T) {
	data := UserResourceModel{Email: types.StringValue("old@example.com")}

	applyRESTUser(&data, &client.RESTUser{ID: "u1", Email: "new@example.com", Status: "ACTIVE"})

	if data.Email.ValueString() != "new@example.com" {
		t.Fatalf("email = %q, want server value so the drift shows in plan", data.Email.ValueString())
	}
}

func TestEmailChangeRequiresReplace(t *testing.T) {
	cases := []struct {
		state, plan string
		want        bool
	}{
		{"alice@example.com", "bob@example.com", true},
		{"alice@example.com", "Alice@Example.com", false},
		{"alice@example.com", "alice@example.com", false},
	}
	for _, c := range cases {
		if got := emailChanged(c.state, c.plan); got != c.want {
			t.Errorf("emailChanged(%q, %q) = %v, want %v", c.state, c.plan, got, c.want)
		}
	}
}
