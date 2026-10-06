package provider

import (
	"context"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
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
	APIKey   types.String `tfsdk:"api_key"`
	Endpoint types.String `tfsdk:"endpoint"`
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
				MarkdownDescription: "Arize GraphQL endpoint. Defaults to https://app.arize.com/graphql.",
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

	// Use environment variable if not set in config
	if config.APIKey.IsNull() {
		apiKey := os.Getenv("ARIZE_API_KEY")
		if apiKey != "" {
			config.APIKey = types.StringValue(apiKey)
		}
	}

	// Set default endpoint
	if config.Endpoint.IsNull() {
		config.Endpoint = types.StringValue("https://app.arize.com/graphql")
	}

	// Store config for resources to use
	resp.DataSourceData = &config
	resp.ResourceData = &config
}

func (p *ArizeProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewUserResource,
		NewRoleResource,
		NewAPIKeyResource,
	}
}

func (p *ArizeProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewUsersDataSource,
		NewRolesDataSource,
		NewAPIKeysDataSource,
	}
}
