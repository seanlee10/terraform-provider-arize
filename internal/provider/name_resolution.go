package provider

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/arize-ai/terraform-provider-arize/internal/client"
)

// Attributes such as organization_id and integration_id accept either an
// Arize ID or a name, like the ax CLI and SDK do. The configured value is
// kept in state; the ID is looked up whenever it is sent to Arize.

// isArizeID reports whether value is an Arize global ID (of any type).
func isArizeID(value string) bool {
	_, ok := client.GlobalIDType(value)
	return ok
}

// matchName picks the one item named name: an exact match first, then a
// case-insensitive one. kind ("organization") and hint (where IDs come from)
// are used in errors, which list the available names and IDs.
func matchName(kind, name, hint string, items []client.NamedID) (string, error) {
	for _, fold := range []bool{false, true} {
		var found []client.NamedID
		for _, it := range items {
			if it.Name == name || fold && strings.EqualFold(it.Name, name) {
				found = append(found, it)
			}
		}
		switch {
		case len(found) == 1:
			return found[0].ID, nil
		case len(found) > 1:
			return "", fmt.Errorf("%q matches %d %ss. Use the ID instead:\n%s",
				name, len(found), kind, listNamedIDs(found))
		}
	}
	if len(items) == 0 {
		return "", fmt.Errorf("%q isn't an Arize ID, and no %s has that name (none are available to this API key). %s",
			name, kind, hint)
	}
	return "", fmt.Errorf("%q isn't an Arize ID, and no %s has that name. Use one of these names or IDs (%s):\n%s",
		name, kind, hint, listNamedIDs(items))
}

func listNamedIDs(items []client.NamedID) string {
	sorted := append([]client.NamedID(nil), items...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Name < sorted[j].Name })
	var b strings.Builder
	for _, it := range sorted {
		fmt.Fprintf(&b, "- %q: %s\n", it.Name, it.ID)
	}
	return strings.TrimRight(b.String(), "\n")
}

// resolveOrganizationID returns value if it is an ID, otherwise the ID of the
// organization with that name.
func resolveOrganizationID(ctx context.Context, c *client.Client, value string) (string, error) {
	if isArizeID(value) {
		return value, nil
	}
	orgs, err := c.ListOrganizations(ctx)
	if err != nil {
		return "", err
	}
	return matchName("organization", value, "IDs are listed by `ax organizations list`", orgs)
}

// resolveIntegrationID returns value if it is an ID, otherwise the ID of the
// LLM integration with that name. spaceID, if it is a known ID, limits the
// search to integrations available to that space.
func resolveIntegrationID(ctx context.Context, rc *client.RESTClient, value, spaceID string) (string, error) {
	if isArizeID(value) {
		return value, nil
	}
	if !isArizeID(spaceID) {
		spaceID = ""
	}
	// List all integrations rather than filtering by name, so a failed
	// lookup can show what is available.
	integrations, err := rc.ListAIIntegrations(ctx, "", spaceID)
	if err != nil {
		return "", err
	}
	hint := "IDs are listed by `ax ai-integrations list`"
	if spaceID != "" {
		hint = "integrations available to this space; " + hint
	}
	id, err := matchName("LLM integration", value, hint, integrations)
	if err != nil && spaceID != "" {
		// Say so if the integration exists but is limited to other
		// organizations or spaces.
		if all, lerr := rc.ListAIIntegrations(ctx, "", ""); lerr == nil {
			if _, merr := matchName("LLM integration", value, "", all); merr == nil {
				return "", fmt.Errorf("LLM integration %q isn't available to this space: it is limited to other organizations or spaces. "+
					"Change its scope in Arize, or use one of these (%s):\n%s", value, hint, listNamedIDs(integrations))
			}
		}
	}
	return id, err
}

// integrationNameExists checks that value is an ID or that some LLM
// integration in the account has that name (any case), without requiring it
// to be unique: used at plan time when the space isn't known yet.
func integrationNameExists(ctx context.Context, rc *client.RESTClient, value string) error {
	if isArizeID(value) {
		return nil
	}
	integrations, err := rc.ListAIIntegrations(ctx, "", "")
	if err != nil {
		return err
	}
	for _, it := range integrations {
		if strings.EqualFold(it.Name, value) {
			return nil
		}
	}
	_, err = matchName("LLM integration", value, "IDs are listed by `ax ai-integrations list`", integrations)
	return err
}
