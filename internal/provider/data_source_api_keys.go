package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/arize-ai/terraform-provider-arize/internal/client"
)

var _ datasource.DataSource = (*APIKeysDataSource)(nil)

func NewAPIKeysDataSource() datasource.DataSource {
	return &APIKeysDataSource{}
}

type APIKeysDataSource struct {
	client *client.Client
}

type APIKeysDataSourceModel struct {
	APIKeys []APIKeyDataSourceModel `tfsdk:"api_keys"`
}

type APIKeyDataSourceModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Permissions types.List   `tfsdk:"permissions"`
	CreatedAt   types.String `tfsdk:"created_at"`
	LastUsedAt  types.String `tfsdk:"last_used_at"`
}

func (d *APIKeysDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_api_keys"
}

func (d *APIKeysDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Fetch all API keys from Arize.",
		Attributes: map[string]schema.Attribute{
			"api_keys": schema.ListNestedAttribute{
				MarkdownDescription: "List of API keys.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							MarkdownDescription: "API key ID.",
							Computed:            true,
						},
						"name": schema.StringAttribute{
							MarkdownDescription: "API key name.",
							Computed:            true,
						},
						"permissions": schema.ListAttribute{
							ElementType:         types.StringType,
							MarkdownDescription: "API key permissions.",
							Computed:            true,
						},
						"created_at": schema.StringAttribute{
							MarkdownDescription: "When the API key was created.",
							Computed:            true,
						},
						"last_used_at": schema.StringAttribute{
							MarkdownDescription: "When the API key was last used.",
							Computed:            true,
						},
					},
				},
			},
		},
	}
}

func (d *APIKeysDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	config, ok := req.ProviderData.(*ArizeProviderModel)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected DataSource Configure Type",
			fmt.Sprintf("Expected *ArizeProviderModel, got: %T", req.ProviderData),
		)
		return
	}

	d.client = client.New(config.Endpoint.ValueString(), config.APIKey.ValueString())
}

func (d *APIKeysDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data APIKeysDataSourceModel

	apiKeys, err := d.client.ListAPIKeys(ctx, 100, "")
	if err != nil {
		resp.Diagnostics.AddError("Failed to list API keys", err.Error())
		return
	}

	for _, apiKey := range apiKeys {
		permsValue, _ := types.ListValueFrom(ctx, types.StringType, apiKey.Permissions)
		data.APIKeys = append(data.APIKeys, APIKeyDataSourceModel{
			ID:          types.StringValue(apiKey.ID),
			Name:        types.StringValue(apiKey.Name),
			Permissions: permsValue,
			CreatedAt:   types.StringValue(apiKey.CreatedAt),
			LastUsedAt:  types.StringValue(apiKey.LastUsedAt),
		})
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
