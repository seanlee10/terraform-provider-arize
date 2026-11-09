package provider

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ provider.Provider = (*ArizeProvider)(nil)
)

type ArizeProvider struct {
	version string
}

type ArizeProviderModel struct {
	APIKey       types.String `tfsdk:"api_key"`
	Endpoint     types.String `tfsdk:"endpoint"`
	RESTEndpoint types.String `tfsdk:"rest_endpoint"`
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &ArizeProvider{
			version: version,
		}
	}
}

func (p *ArizeProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "arize"
	resp.Version = p.version
}

func (p *ArizeProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"api_key": schema.StringAttribute{
				MarkdownDescription: "Arize API Key. May also be provided via ARIZE_API_KEY environment variable.",
				Optional:            true,
				Sensitive:           true,
			},
			"endpoint": schema.StringAttribute{
				MarkdownDescription: "Arize GraphQL endpoint. Defaults to https://app.arize.com/graphql. EU accounts use https://app.eu-west-1a.arize.com/graphql.",
				Optional:            true,
			},
			"rest_endpoint": schema.StringAttribute{
				MarkdownDescription: "Arize REST API base URL, used for users, roles, datasets, space updates and deletes, and integration names. Defaults to the REST API of the endpoint's region (https://api.eu-west-1a.arize.com for https://app.eu-west-1a.arize.com/graphql), or https://api.arize.com.",
				Optional:            true,
			},
		},
	}
}

func (p *ArizeProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var config ArizeProviderModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	for _, a := range []struct {
		name  string
		value types.String
	}{{"api_key", config.APIKey}, {"endpoint", config.Endpoint}, {"rest_endpoint", config.RESTEndpoint}} {
		if a.value.IsUnknown() {
			resp.Diagnostics.AddAttributeError(path.Root(a.name), "Unknown provider "+a.name,
				fmt.Sprintf("The provider's %s depends on a value that isn't known until apply. "+
					"Set it to a known value (a variable or the ARIZE_API_KEY environment variable), "+
					"or apply the resource it depends on first with -target.", a.name))
		}
	}
	if resp.Diagnostics.HasError() {
		return
	}

	// Use environment variable if not set in config
	if config.APIKey.IsNull() {
		config.APIKey = types.StringValue(os.Getenv("ARIZE_API_KEY"))
	}
	// A key read from a file often ends in a newline, which can't be sent
	// in a header.
	config.APIKey = types.StringValue(strings.TrimSpace(config.APIKey.ValueString()))
	if config.APIKey.ValueString() == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("api_key"),
			"Missing Arize API key",
			"Set api_key in the provider block or the ARIZE_API_KEY environment variable.",
		)
		return
	}

	if config.Endpoint.IsNull() {
		config.Endpoint = types.StringValue(defaultEndpoint)
	}
	if config.RESTEndpoint.IsNull() {
		rest, ok := restEndpointFor(config.Endpoint.ValueString())
		if !ok {
			resp.Diagnostics.AddAttributeWarning(path.Root("rest_endpoint"), "rest_endpoint not set",
				fmt.Sprintf("Couldn't work out the REST API URL from endpoint %q, so %s is used. "+
					"Set rest_endpoint if your account is on another cluster; otherwise users, roles, "+
					"datasets and space updates fail with HTTP 401.", config.Endpoint.ValueString(), rest))
		}
		config.RESTEndpoint = types.StringValue(rest)
	}

	// Store config for resources to use
	resp.DataSourceData = &config
	resp.ResourceData = &config
}

func (p *ArizeProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewUserResource,
		NewRoleResource,
		NewSpaceResource,
		NewSpaceMemberResource,
		NewSAMLIdPResource,
		NewEvaluatorResource,
		NewDatasetResource,
	}
}

func (p *ArizeProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewUsersDataSource,
		NewRolesDataSource,
		NewEvaluatorsDataSource,
	}
}

const (
	defaultEndpoint     = "https://app.arize.com/graphql"
	defaultRESTEndpoint = "https://api.arize.com"
)

// restEndpointFor returns the REST API URL on the same cluster as the
// GraphQL endpoint: https://app.<region>.arize.com/graphql becomes
// https://api.<region>.arize.com. Keys only work on their own cluster, so
// sending REST calls to the default (US) cluster fails with 401 for EU
// accounts. ok is false if the endpoint isn't in that form.
func restEndpointFor(endpoint string) (string, bool) {
	u, err := url.Parse(endpoint)
	if err != nil || u.Host == "" {
		return defaultRESTEndpoint, false
	}
	rest, found := strings.CutPrefix(u.Host, "app.")
	if !found {
		return defaultRESTEndpoint, false
	}
	return u.Scheme + "://api." + rest, true
}
