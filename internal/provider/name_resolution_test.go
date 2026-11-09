package provider

import (
	"strings"
	"testing"

	"github.com/arize-ai/terraform-provider-arize/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var testOrgs = []client.NamedID{
	{ID: "QWNjb3VudE9yZ2FuaXphdGlvbjo2ODY6VFlZYg==", Name: "Customer"},
	{ID: "QWNjb3VudE9yZ2FuaXphdGlvbjo2Njc6YkVkYw==", Name: "Experimentation"},
	{ID: "id-x1", Name: "Dup"},
	{ID: "id-x2", Name: "dup"},
	{ID: "id-x3", Name: "Twin"},
	{ID: "id-x4", Name: "Twin"},
}

func TestMatchName(t *testing.T) {
	cases := []struct {
		name, want, errHas string
	}{
		{name: "Experimentation", want: "QWNjb3VudE9yZ2FuaXphdGlvbjo2Njc6YkVkYw=="},
		{name: "experimentation", want: "QWNjb3VudE9yZ2FuaXphdGlvbjo2Njc6YkVkYw=="}, // case-insensitive fallback
		{name: "Dup", want: "id-x1"}, // exact match wins over case-insensitive
		{name: "DUP", errHas: `"DUP" matches 2 organizations. Use the ID instead`},
		{name: "Twin", errHas: `"Twin" matches 2 organizations`},
		{name: "Experiment", errHas: `no organization has that name`},
	}
	for _, c := range cases {
		got, err := matchName("organization", c.name, "hint", testOrgs)
		if c.errHas != "" {
			if err == nil || !strings.Contains(err.Error(), c.errHas) {
				t.Errorf("matchName(%q) error = %v, want it to contain %q", c.name, err, c.errHas)
			}
			continue
		}
		if err != nil || got != c.want {
			t.Errorf("matchName(%q) = %q, %v; want %q", c.name, got, err, c.want)
		}
	}
}

func TestMatchNameListsChoices(t *testing.T) {
	_, err := matchName("LLM integration", "Omnia", "IDs are listed by `ax ai-integrations list`",
		[]client.NamedID{{ID: "TGxtSW50ZWdyYXRpb246MjI3Ok5WYnM=", Name: "Portkey API"}, {ID: "a", Name: "Azure"}})
	want := `"Omnia" isn't an Arize ID, and no LLM integration has that name. Use one of these names or IDs (IDs are listed by ` + "`ax ai-integrations list`" + `):
- "Azure": a
- "Portkey API": TGxtSW50ZWdyYXRpb246MjI3Ok5WYnM=`
	if err == nil || err.Error() != want {
		t.Fatalf("error =\n%v\nwant\n%s", err, want)
	}

	_, err = matchName("LLM integration", "Omnia", "hint", nil)
	if err == nil || !strings.Contains(err.Error(), "none are available to this API key") {
		t.Fatalf("error = %v", err)
	}
}

func TestOrganizationIDState(t *testing.T) {
	org := &client.Org{ID: "QWNjb3VudE9yZ2FuaXphdGlvbjo2Njc6YkVkYw==", Name: "Experimentation"}
	cases := []struct {
		current types.String
		want    string
	}{
		{types.StringValue("Experimentation"), "Experimentation"},
		{types.StringValue("experimentation"), "experimentation"},
		{types.StringValue(org.ID), org.ID},
		{types.StringValue("Customer"), org.ID}, // moved outside Terraform: show the real organization
		{types.StringNull(), org.ID},            // import
		{types.StringUnknown(), org.ID},
	}
	for _, c := range cases {
		if got := organizationIDState(c.current, org); got.ValueString() != c.want {
			t.Errorf("organizationIDState(%v) = %q, want %q", c.current, got.ValueString(), c.want)
		}
	}
}

func TestRESTEndpointFor(t *testing.T) {
	for endpoint, want := range map[string]struct {
		url string
		ok  bool
	}{
		"https://app.arize.com/graphql":              {"https://api.arize.com", true},
		"https://app.eu-west-1a.arize.com/graphql":   {"https://api.eu-west-1a.arize.com", true},
		"https://arize.internal.example.com/graphql": {defaultRESTEndpoint, false},
		"not a url": {defaultRESTEndpoint, false},
	} {
		got, ok := restEndpointFor(endpoint)
		if got != want.url || ok != want.ok {
			t.Errorf("restEndpointFor(%q) = %q, %v; want %q, %v", endpoint, got, ok, want.url, want.ok)
		}
	}
}
