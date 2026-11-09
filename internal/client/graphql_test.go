package client

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// An error status with a JSON body that isn't a GraphQL error must be an
// error, not an empty result that Read would take as "deleted".
func TestExecuteErrorStatusWithJSONBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		fmt.Fprint(w, `{"message":"upstream unavailable"}`)
	}))
	defer srv.Close()

	_, err := New(srv.URL, "k").GetSpace(context.Background(), "U3BhY2U6MTox")
	if err == nil || errors.Is(err, ErrSpaceNotFound) || !strings.Contains(err.Error(), "HTTP 502") {
		t.Fatalf("error = %v, want an HTTP 502 error", err)
	}
}

func TestUnauthorizedMentionsRegion(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprint(w, "Unauthorized")
	}))
	defer srv.Close()

	_, err := New(srv.URL, "k").GetSpace(context.Background(), "U3BhY2U6MTox")
	if err == nil || !strings.Contains(err.Error(), "eu-west-1a") {
		t.Fatalf("GraphQL error = %v, want a region hint", err)
	}
	err = NewREST(srv.URL, "k").do(context.Background(), http.MethodGet, "/v2/spaces", nil, nil)
	if err == nil || !strings.Contains(err.Error(), "rest_endpoint") {
		t.Fatalf("REST error = %v, want a region hint", err)
	}
}

// A deleted space makes its evaluators gone, not an error that blocks plans.
func TestGetEvaluatorSpaceDeleted(t *testing.T) {
	notFound := `{"errors":[{"message":"No object found.","path":["node"],"extensions":{"code":"NOT_FOUND"}}],"data":{"node":null}}`
	for name, responses := range map[string][]string{
		"unauthorized, then NOT_FOUND listing": {`{"errors":[{"message":"Unauthorized access"}],"data":{"node":null}}`, notFound},
		"NOT_FOUND, then null space":           {notFound, `{"data":{"node":null}}`},
	} {
		c, _ := gqlServer(t, responses...)
		if _, err := c.GetEvaluator(context.Background(), "RXZhbHVhdG9yOjE6YQ==", "U3BhY2U6MTox"); !errors.Is(err, ErrEvaluatorNotFound) {
			t.Errorf("%s: error = %v, want ErrEvaluatorNotFound", name, err)
		}
	}
}
