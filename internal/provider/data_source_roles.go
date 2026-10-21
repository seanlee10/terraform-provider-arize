package provider

import (
	"context"
	"fmt"

	"github.com/arize-ai/terraform-provider-arize/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = (*RolesDataSource)(nil)

func NewRolesDataSource() datasource.DataSource {
	return &RolesDataSource{}
}

type RolesDataSource struct {
	client *client.RESTClient
}

type RolesDataSourceModel struct {
	Roles []RoleDataSourceModel `tfsdk:"roles"`
}

type RoleDataSourceModel struct {
	ID           types.String `tfsdk:"id"`
	Name         types.String `tfsdk:"name"`
	Description  types.String `tfsdk:"description"`
	Permissions  types.Set    `tfsdk:"permissions"`
	IsPredefined types.Bool   `tfsdk:"is_predefined"`
}

func (d *RolesDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_roles"
}

func (d *RolesDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Lists all predefined and custom roles in the Arize account, including ones created in the Arize UI.",
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
						"permissions": schema.SetAttribute{
							ElementType:         types.StringType,
							MarkdownDescription: "Permissions granted by the role.",
							Computed:            true,
						},
						"is_predefined": schema.BoolAttribute{
							MarkdownDescription: "Whether this is a predefined Arize role (read-only) rather than a custom role.",
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

	d.client = client.NewREST(config.RESTEndpoint.ValueString(), config.APIKey.ValueString())
}

func (d *RolesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data RolesDataSourceModel

	roles, err := d.client.ListRoles(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Failed to list roles", err.Error())
		return
	}

	data.Roles = make([]RoleDataSourceModel, 0, len(roles))
	for _, role := range roles {
		perms, diags := types.SetValueFrom(ctx, types.StringType, role.Permissions)
		resp.Diagnostics.Append(diags...)
		data.Roles = append(data.Roles, RoleDataSourceModel{
			ID:           types.StringValue(role.ID),
			Name:         types.StringValue(role.Name),
			Description:  types.StringPointerValue(role.Description),
			Permissions:  perms,
			IsPredefined: types.BoolValue(role.IsPredefined),
		})
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
