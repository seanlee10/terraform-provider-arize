package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/arize-ai/terraform-provider-arize/internal/client"
)

var _ resource.Resource = (*UserResource)(nil)
var _ resource.ResourceWithImportState = (*UserResource)(nil)

func NewUserResource() resource.Resource {
	return &UserResource{}
}

type UserResource struct {
	client *client.Client
}

type UserResourceModel struct {
	ID        types.String `tfsdk:"id"`
	Email     types.String `tfsdk:"email"`
	Name      types.String `tfsdk:"name"`
	Status    types.String `tfsdk:"status"`
	UserType  types.String `tfsdk:"user_type"`
	CreatedAt types.String `tfsdk:"created_at"`
}

func (r *UserResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_user"
}

func (r *UserResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages an Arize user.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The unique identifier for the user.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"email": schema.StringAttribute{
				MarkdownDescription: "The email address of the user.",
				Required:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The full name of the user.",
				Optional:            true,
			},
			"status": schema.StringAttribute{
				MarkdownDescription: "The status of the user (active, pending, etc).",
				Computed:            true,
			},
			"user_type": schema.StringAttribute{
				MarkdownDescription: "The type of user (human or bot). Defaults to human.",
				Computed:            true,
			},
			"created_at": schema.StringAttribute{
				MarkdownDescription: "Timestamp when the user was created.",
				Computed:            true,
			},
		},
	}
}

func (r *UserResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	config, ok := req.ProviderData.(*ArizeProviderModel)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *ArizeProviderModel, got: %T", req.ProviderData),
		)
		return
	}

	r.client = client.New(config.Endpoint.ValueString(), config.APIKey.ValueString())
}

func (r *UserResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data UserResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	input := &client.CreateUserInput{
		Email: data.Email.ValueString(),
		Name:  data.Name.ValueString(),
	}

	user, err := r.client.CreateUser(ctx, input)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create user", err.Error())
		return
	}

	data.ID = types.StringValue(user.ID)
	data.Status = types.StringValue(user.Status)
	data.UserType = types.StringValue(user.UserType)
	data.CreatedAt = types.StringValue(user.CreatedAt)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *UserResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data UserResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	user, err := r.client.GetUser(ctx, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read user", err.Error())
		return
	}

	data.Email = types.StringValue(user.Email)
	data.Name = types.StringValue(user.Name)
	data.Status = types.StringValue(user.Status)
	data.UserType = types.StringValue(user.UserType)
	data.CreatedAt = types.StringValue(user.CreatedAt)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *UserResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data UserResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	input := &client.UpdateUserInput{
		Name: data.Name.ValueString(),
	}

	_, err := r.client.UpdateUser(ctx, data.ID.ValueString(), input)
	if err != nil {
		resp.Diagnostics.AddError("Failed to update user", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *UserResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data UserResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.DeleteUser(ctx, data.ID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Failed to delete user", err.Error())
		return
	}
}

func (r *UserResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	userID := req.ID
	user, err := r.client.GetUser(ctx, userID)
	if err != nil {
		resp.Diagnostics.AddError("Failed to import user", err.Error())
		return
	}

	data := UserResourceModel{
		ID:        types.StringValue(user.ID),
		Email:     types.StringValue(user.Email),
		Name:      types.StringValue(user.Name),
		Status:    types.StringValue(user.Status),
		UserType:  types.StringValue(user.UserType),
		CreatedAt: types.StringValue(user.CreatedAt),
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
