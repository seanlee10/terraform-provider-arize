package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/arize-ai/terraform-provider-arize/internal/client"
)

var _ datasource.DataSource = (*RolesDataSource)(nil)

func NewRolesDataSource() datasource.DataSource {
	return &RolesDataSource{}
}

type RolesDataSource struct {
	client *client.Client
}

type RolesDataSourceModel struct {
	Roles []RoleDataSourceModel `tfsdk:"roles"`
}

type RoleDataSourceModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	Permissions types.List   `tfsdk:"permissions"`
}

func (d *RolesDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_roles"
}

func (d *RolesDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Fetch all roles from Arize.",
		Attributes: map[string]schema.Attribute{
			"roles": schema.ListNestedAttribute{
				MarkdownDescription: "List of roles.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							MarkdownDescription: "Role ID.",
							Computed:            true,
						},
						"name": schema.StringAttribute{
							MarkdownDescription: "Role name.",
							Computed:            true,
						},
						"description": schema.StringAttribute{
							MarkdownDescription: "Role description.",
							Computed:            true,
						},
						"permissions": schema.ListAttribute{
							ElementType:         types.StringType,
							MarkdownDescription: "Role permissions.",
							Computed:            true,
						},
					},
				},
			},
		},
	}
}

func (d *RolesDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *RolesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data RolesDataSourceModel

	roles, err := d.client.ListRoles(ctx, 100, "")
	if err != nil {
		resp.Diagnostics.AddError("Failed to list roles", err.Error())
		return
	}

	for _, role := range roles {
		permsValue, _ := types.ListValueFrom(ctx, types.StringType, role.Permissions)
		data.Roles = append(data.Roles, RoleDataSourceModel{
			ID:          types.StringValue(role.ID),
			Name:        types.StringValue(role.Name),
			Description: types.StringValue(role.Description),
			Permissions: permsValue,
		})
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
