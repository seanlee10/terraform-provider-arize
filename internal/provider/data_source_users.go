package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/arize-ai/terraform-provider-arize/internal/client"
)

var _ datasource.DataSource = (*UsersDataSource)(nil)

func NewUsersDataSource() datasource.DataSource {
	return &UsersDataSource{}
}

type UsersDataSource struct {
	client *client.Client
}

type UsersDataSourceModel struct {
	Users []UserDataSourceModel `tfsdk:"users"`
}

type UserDataSourceModel struct {
	ID        types.String `tfsdk:"id"`
	Email     types.String `tfsdk:"email"`
	FirstName types.String `tfsdk:"first_name"`
	LastName  types.String `tfsdk:"last_name"`
	UserType  types.String `tfsdk:"user_type"`
}

func (d *UsersDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_users"
}

func (d *UsersDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Fetch all users from Arize.",
		Attributes: map[string]schema.Attribute{
			"users": schema.ListNestedAttribute{
				MarkdownDescription: "List of users.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							MarkdownDescription: "User ID.",
							Computed:            true,
						},
						"email": schema.StringAttribute{
							MarkdownDescription: "User email.",
							Computed:            true,
						},
						"first_name": schema.StringAttribute{
							MarkdownDescription: "User first name.",
							Computed:            true,
						},
						"last_name": schema.StringAttribute{
							MarkdownDescription: "User last name.",
							Computed:            true,
						},
						"user_type": schema.StringAttribute{
							MarkdownDescription: "User type (human or bot).",
							Computed:            true,
						},
					},
				},
			},
		},
	}
}

func (d *UsersDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *UsersDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data UsersDataSourceModel

	users, err := d.client.ListUsers(ctx, 100, "")
	if err != nil {
		resp.Diagnostics.AddError("Failed to list users", err.Error())
		return
	}

	for _, user := range users {
		data.Users = append(data.Users, UserDataSourceModel{
			ID:        types.StringValue(user.ID),
			Email:     types.StringValue(user.Email),
			FirstName: types.StringValue(user.FirstName),
			LastName:  types.StringValue(user.LastName),
			UserType:  types.StringValue(user.UserType),
		})
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
