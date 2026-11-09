package client

import (
	"context"
	"reflect"
	"testing"
)

func TestGetOrganizationAccess_ReportsViewerRoleInEachOrg(t *testing.T) {
	c, vars := gqlServer(t,
		`{"data":{"viewer":{"user":{"id":"u1","email":"jane@x.com","accountRole":"member"}}}}`,
		`{"data":{"account":{"organizations":{"edges":[
			{"node":{"id":"o1","name":"Main","accountOrganizationUsers":{"edges":[
				{"node":{"role":"member","membership":"EXPLICIT_MEMBERSHIP","user":{"id":"u1"}}}]}}},
			{"node":{"id":"o2","name":"Other","accountOrganizationUsers":{"edges":[
				{"node":{"role":"admin","membership":"EXPLICIT_MEMBERSHIP","user":{"id":"u9"}}}]}}}
		]}}}}`,
	)

	got, err := c.GetOrganizationAccess(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	want := &OrganizationAccess{
		UserEmail: "jane@x.com", AccountRole: "member",
		Organizations: []OrganizationRole{
			{ID: "o1", Name: "Main", Role: "member", Membership: "EXPLICIT_MEMBERSHIP"},
			{ID: "o2", Name: "Other"}, // search matched someone else: not a member
		},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("GetOrganizationAccess() = %+v, want %+v", got, want)
	}
	if (*vars)[1]["email"] != "jane@x.com" {
		t.Fatalf("orgs query variables = %v, want the viewer's email as the search", (*vars)[1])
	}
}

func TestIsUnauthorized(t *testing.T) {
	c, _ := gqlServer(t, `{"errors":[{"message":"Not Authorized","path":["createSpace"],"extensions":{"code":"UNAUTHORIZED"}}],"data":{"createSpace":null}}`)

	_, err := c.CreateSpace(context.Background(), &CreateSpaceInput{Name: "x", AccountOrganizationID: "o1"})

	if !IsUnauthorized(err) {
		t.Fatalf("IsUnauthorized(%v) = false, want true", err)
	}
	if IsUnauthorized(nil) {
		t.Fatal("IsUnauthorized(nil) = true")
	}
}
