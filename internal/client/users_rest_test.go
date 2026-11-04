package client

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
)

// restRecorder serves one canned response and records the request.
type restRecorder struct {
	method, path string
	body         map[string]interface{}
}

func newRESTRecorder(t *testing.T, status int, response string) (*restRecorder, *RESTClient) {
	rec := &restRecorder{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rec.method, rec.path = r.Method, r.URL.Path
		b, _ := io.ReadAll(r.Body)
		if len(b) > 0 {
			_ = json.Unmarshal(b, &rec.body)
		}
		w.WriteHeader(status)
		fmt.Fprint(w, response)
	}))
	t.Cleanup(srv.Close)
	return rec, NewREST(srv.URL, "test-key")
}

func TestCreateUserREST_InvitesMemberByEmail(t *testing.T) {
	rec, c := newRESTRecorder(t, http.StatusCreated,
		`{"id":"inv1","name":"tf@example.com","email":"tf@example.com","status":"INVITED","created_at":"2026-11-04T06:49:53Z"}`)

	u, err := c.CreateUser(context.Background(), "TF Test User", "tf@example.com")
	if err != nil {
		t.Fatal(err)
	}

	want := &RESTUser{ID: "inv1", Name: "tf@example.com", Email: "tf@example.com", Status: "INVITED", CreatedAt: "2026-11-04T06:49:53Z"}
	if !reflect.DeepEqual(u, want) {
		t.Fatalf("CreateUser() = %+v, want %+v", u, want)
	}
	if rec.method != http.MethodPost || rec.path != "/v2/users" {
		t.Fatalf("request = %s %s, want POST /v2/users", rec.method, rec.path)
	}
	wantBody := map[string]interface{}{
		"name":        "TF Test User",
		"email":       "tf@example.com",
		"role":        map[string]interface{}{"type": "PREDEFINED", "name": "MEMBER"},
		"invite_mode": "EMAIL_LINK",
	}
	if !reflect.DeepEqual(rec.body, wantBody) {
		t.Fatalf("body = %v, want %v", rec.body, wantBody)
	}
}

func TestGetUserREST_NotFoundReturnsSentinel(t *testing.T) {
	_, c := newRESTRecorder(t, http.StatusNotFound, `{"status":404,"title":"Not Found"}`)

	_, err := c.GetUserREST(context.Background(), "gone")

	if !errors.Is(err, ErrUserNotFound) {
		t.Fatalf("GetUserREST() error = %v, want ErrUserNotFound", err)
	}
}

func TestFindUserByEmail_MatchesExactlyAcrossPages(t *testing.T) {
	var queries []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		queries = append(queries, r.URL.RawQuery)
		if r.URL.Query().Get("cursor") == "" {
			// Partial match from the API: a different address containing the same text.
			fmt.Fprint(w, `{"users":[{"id":"u2","email":"tf2@example.com","status":"ACTIVE"}],"pagination":{"has_more":true,"next_cursor":"p2"}}`)
			return
		}
		fmt.Fprint(w, `{"users":[{"id":"u1","email":"TF@example.com","status":"ACTIVE"}],"pagination":{"has_more":false}}`)
	}))
	t.Cleanup(srv.Close)
	c := NewREST(srv.URL, "k")

	u, err := c.FindUserByEmail(context.Background(), "tf@example.com")
	if err != nil {
		t.Fatal(err)
	}

	if u.ID != "u1" {
		t.Fatalf("FindUserByEmail() id = %q, want u1 (case-insensitive exact match)", u.ID)
	}
	if len(queries) != 2 || !strings.Contains(queries[0], "email=tf%40example.com") {
		t.Fatalf("queries = %v, want 2 pages filtered by email", queries)
	}
}

func TestFindUserByEmail_NoMatchReturnsSentinel(t *testing.T) {
	_, c := newRESTRecorder(t, http.StatusOK, `{"users":[{"id":"u2","email":"tf2@example.com"}],"pagination":{"has_more":false}}`)

	_, err := c.FindUserByEmail(context.Background(), "tf@example.com")

	if !errors.Is(err, ErrUserNotFound) {
		t.Fatalf("FindUserByEmail() error = %v, want ErrUserNotFound", err)
	}
}

func TestRenameUserREST_PatchesName(t *testing.T) {
	rec, c := newRESTRecorder(t, http.StatusOK, `{}`)

	if err := c.RenameUser(context.Background(), "u1", "New Name"); err != nil {
		t.Fatal(err)
	}

	if rec.method != http.MethodPatch || rec.path != "/v2/users/u1" {
		t.Fatalf("request = %s %s, want PATCH /v2/users/u1", rec.method, rec.path)
	}
	if want := map[string]interface{}{"name": "New Name"}; !reflect.DeepEqual(rec.body, want) {
		t.Fatalf("body = %v, want %v", rec.body, want)
	}
}

func TestDeleteUserREST_SendsDeleteAndToleratesNotFound(t *testing.T) {
	rec, c := newRESTRecorder(t, http.StatusNotFound, `{"status":404,"title":"Not Found"}`)

	if err := c.DeleteUser(context.Background(), "u1"); err != nil {
		t.Fatalf("DeleteUser() error = %v, want nil for 404", err)
	}
	if rec.method != http.MethodDelete || rec.path != "/v2/users/u1" {
		t.Fatalf("request = %s %s, want DELETE /v2/users/u1", rec.method, rec.path)
	}
}
