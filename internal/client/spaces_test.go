package client

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func patchRecorder(t *testing.T) (*RESTClient, *string, *map[string]interface{}) {
	var path string
	var body map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Errorf("method = %s, want PATCH", r.Method)
		}
		path = r.URL.Path
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &body)
		w.Write([]byte(`{}`))
	}))
	t.Cleanup(srv.Close)
	return NewREST(srv.URL, "k"), &path, &body
}

func TestUpdateSpace_SendsAllFields(t *testing.T) {
	c, path, body := patchRecorder(t)
	desc := "ML models"

	err := c.UpdateSpace(context.Background(), "sp1", &UpdateSpaceInput{Name: "prod", Description: &desc, IsPrivate: true})
	if err != nil {
		t.Fatal(err)
	}

	if *path != "/v2/spaces/sp1" {
		t.Fatalf("path = %q, want /v2/spaces/sp1", *path)
	}
	want := map[string]interface{}{"name": "prod", "description": "ML models", "is_private": true}
	if !reflect.DeepEqual(*body, want) {
		t.Fatalf("body = %v, want %v", *body, want)
	}
}

func TestUpdateSpace_NilDescriptionSendsNullToClearIt(t *testing.T) {
	c, _, body := patchRecorder(t)

	if err := c.UpdateSpace(context.Background(), "sp1", &UpdateSpaceInput{Name: "prod"}); err != nil {
		t.Fatal(err)
	}

	want := map[string]interface{}{"name": "prod", "description": nil, "is_private": false}
	if !reflect.DeepEqual(*body, want) {
		t.Fatalf("body = %v, want %v (explicit null clears the description)", *body, want)
	}
}
