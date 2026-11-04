package client

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

// gqlServer answers each GraphQL request with the next canned response and
// records the request variables.
func gqlServer(t *testing.T, responses ...string) (*Client, *[]map[string]interface{}) {
	var vars []map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		var req struct {
			Variables map[string]interface{} `json:"variables"`
		}
		_ = json.Unmarshal(b, &req)
		vars = append(vars, req.Variables)
		if len(vars) > len(responses) {
			t.Fatalf("unexpected request %d", len(vars))
		}
		fmt.Fprint(w, responses[len(vars)-1])
	}))
	t.Cleanup(srv.Close)
	return New(srv.URL, "k"), &vars
}

func spaceUsersPage(hasNext bool, cursor string, users ...string) string {
	edges := ""
	for i, u := range users {
		if i > 0 {
			edges += ","
		}
		edges += `{"node":` + u + `}`
	}
	return fmt.Sprintf(`{"data":{"node":{"spaceUsers":{"pageInfo":{"hasNextPage":%t,"endCursor":%q},"edges":[%s]}}}}`, hasNext, cursor, edges)
}

func TestGetSpaceUser_FindsUserOnLaterPage(t *testing.T) {
	c, vars := gqlServer(t,
		spaceUsersPage(true, "c1", `{"role":null,"membership":"ACCOUNT_ADMIN","user":{"id":"admin","email":"a@x.com"}}`),
		spaceUsersPage(false, "", `{"role":"member","membership":"EXPLICIT_MEMBERSHIP","user":{"id":"u1","email":"u1@x.com","name":"U One"}}`),
	)

	su, err := c.GetSpaceUser(context.Background(), "sp1", "u1")
	if err != nil {
		t.Fatal(err)
	}

	if su.User.ID != "u1" || su.Role == nil || *su.Role != "member" || !su.IsExplicit() {
		t.Fatalf("GetSpaceUser() = %+v, want explicit member u1", su)
	}
	if got := (*vars)[1]["after"]; got != "c1" {
		t.Fatalf("second page after = %v, want c1", got)
	}
}

func TestGetSpaceUser_ImplicitAdminIsNotExplicit(t *testing.T) {
	c, _ := gqlServer(t, spaceUsersPage(false, "", `{"role":null,"membership":"ACCOUNT_ADMIN","user":{"id":"admin"}}`))

	su, err := c.GetSpaceUser(context.Background(), "sp1", "admin")
	if err != nil {
		t.Fatal(err)
	}
	if su.IsExplicit() {
		t.Fatalf("IsExplicit() = true for an ACCOUNT_ADMIN entry, want false")
	}
}

func TestGetSpaceUser_MissingUserOrSpaceReturnsSentinel(t *testing.T) {
	c, _ := gqlServer(t,
		spaceUsersPage(false, "", `{"role":"member","membership":"EXPLICIT_MEMBERSHIP","user":{"id":"other"}}`),
		`{"data":{"node":null}}`,
		`{"errors":[{"message":"No object found.","path":["node"],"extensions":{"code":"NOT_FOUND"}}],"data":{"node":null}}`,
	)

	if _, err := c.GetSpaceUser(context.Background(), "sp1", "u1"); !errors.Is(err, ErrSpaceMemberNotFound) {
		t.Fatalf("user not listed: error = %v, want ErrSpaceMemberNotFound", err)
	}
	if _, err := c.GetSpaceUser(context.Background(), "gone", "u1"); !errors.Is(err, ErrSpaceMemberNotFound) {
		t.Fatalf("space gone: error = %v, want ErrSpaceMemberNotFound", err)
	}
	if _, err := c.GetSpaceUser(context.Background(), "deleted", "u1"); !errors.Is(err, ErrSpaceMemberNotFound) {
		t.Fatalf("space deleted (NOT_FOUND error): error = %v, want ErrSpaceMemberNotFound", err)
	}
}

func TestRemoveSpaceMember_SendsIDs(t *testing.T) {
	c, vars := gqlServer(t, `{"data":{"removeSpaceMember":{"space":{"id":"sp1"}}}}`)

	if err := c.RemoveSpaceMember(context.Background(), "sp1", "u1"); err != nil {
		t.Fatal(err)
	}

	in, _ := (*vars)[0]["input"].(map[string]interface{})
	if in["spaceId"] != "sp1" || in["userId"] != "u1" {
		t.Fatalf("input = %v, want spaceId sp1, userId u1", (*vars)[0]["input"])
	}
}

func TestRemoveSpaceMember_AlreadyRemovedIsNotAnError(t *testing.T) {
	c, _ := gqlServer(t, `{"errors":[{"message":"Space member not found.","path":["removeSpaceMember"],"extensions":{"code":"NOT_FOUND"}}],"data":{"removeSpaceMember":null}}`)

	if err := c.RemoveSpaceMember(context.Background(), "sp1", "u1"); err != nil {
		t.Fatalf("RemoveSpaceMember() error = %v, want nil when already removed", err)
	}
}

func TestRemoveSpaceMember_OtherErrorsFail(t *testing.T) {
	c, _ := gqlServer(t, `{"errors":[{"message":"Forbidden","extensions":{"code":"FORBIDDEN"}}],"data":null}`)

	if err := c.RemoveSpaceMember(context.Background(), "sp1", "u1"); err == nil {
		t.Fatal("RemoveSpaceMember() error = nil, want the API error")
	}
}

func TestGetSpace_DeletedSpaceReturnsSentinel(t *testing.T) {
	c, _ := gqlServer(t,
		`{"errors":[{"message":"No object found.","path":["node"],"extensions":{"code":"NOT_FOUND"}}],"data":{"node":null}}`,
		`{"data":{"node":null}}`,
	)

	for _, name := range []string{"NOT_FOUND error", "null node"} {
		if _, err := c.GetSpace(context.Background(), "gone"); !errors.Is(err, ErrSpaceNotFound) {
			t.Fatalf("%s: error = %v, want ErrSpaceNotFound", name, err)
		}
	}
}
