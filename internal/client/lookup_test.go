package client

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGlobalIDType(t *testing.T) {
	enc := func(s string) string { return base64.StdEncoding.EncodeToString([]byte(s)) }
	cases := []struct {
		in, typ string
		ok      bool
	}{
		{"TGxtSW50ZWdyYXRpb246MTIwOnNLU3c=", "LlmIntegration", true},
		{"QWNjb3VudE9yZ2FuaXphdGlvbjo2Njc6YkVkYw==", "AccountOrganization", true},
		{"QWNjb3VudE9yZ2FuaXphdGlvbjo2Njc6YkVkYw", "AccountOrganization", true}, // unpadded
		{"Omnia", "", false},
		{"Experimentation", "", false},
		{"", "", false},
		{enc("openai"), "", false},                           // name encoded by hand
		{enc("TGxtSW50ZWdyYXRpb246MTIwOnNLU3c="), "", false}, // double-encoded
		{enc("Not a type:1:x"), "", false},
		{"8b589e5a-9d64-4350-be1c-5f6aca6e3835", "", false}, // UUID
	}
	for _, c := range cases {
		typ, ok := GlobalIDType(c.in)
		if typ != c.typ || ok != c.ok {
			t.Errorf("GlobalIDType(%q) = %q, %v; want %q, %v", c.in, typ, ok, c.typ, c.ok)
		}
	}
}

func TestListAIIntegrationsPages(t *testing.T) {
	var queries []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		queries = append(queries, r.URL.RawQuery)
		if r.URL.Path != "/v2/ai-integrations" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		if r.URL.Query().Get("cursor") == "" {
			fmt.Fprint(w, `{"ai_integrations":[{"id":"a","name":"Omnia","provider":"OPEN_AI"}],"pagination":{"has_more":true,"next_cursor":"c1"}}`)
			return
		}
		fmt.Fprint(w, `{"ai_integrations":[{"id":"b","name":"Portkey API"}],"pagination":{"has_more":false}}`)
	}))
	defer srv.Close()

	got, err := NewREST(srv.URL, "k").ListAIIntegrations(context.Background(), "", "U3BhY2U6MTox")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0] != (NamedID{"a", "Omnia"}) || got[1] != (NamedID{"b", "Portkey API"}) {
		t.Fatalf("got %+v", got)
	}
	if len(queries) != 2 || queries[0] != "limit=100&space_id=U3BhY2U6MTox" || queries[1] != "cursor=c1&limit=100&space_id=U3BhY2U6MTox" {
		t.Fatalf("queries = %q", queries)
	}
}

func TestListOrganizationsPages(t *testing.T) {
	c, vars := gqlServer(t,
		`{"data":{"account":{"organizations":{"pageInfo":{"hasNextPage":true,"endCursor":"e1"},"edges":[{"node":{"id":"o1","name":"Customer"}}]}}}}`,
		`{"data":{"account":{"organizations":{"pageInfo":{"hasNextPage":false,"endCursor":"e2"},"edges":[{"node":{"id":"o2","name":"Experimentation"}}]}}}}`,
	)
	got, err := c.ListOrganizations(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[1] != (NamedID{"o2", "Experimentation"}) {
		t.Fatalf("got %+v", got)
	}
	if (*vars)[0]["after"] != nil || (*vars)[1]["after"] != "e1" {
		t.Fatalf("after = %v, %v", (*vars)[0]["after"], (*vars)[1]["after"])
	}
}
