package provider

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/arize-ai/terraform-provider-arize/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = (*UserResource)(nil)
var _ resource.ResourceWithImportState = (*UserResource)(nil)

func NewUserResource() resource.Resource {
	return &UserResource{}
}

type UserResource struct {
	client *client.Client     // GraphQL: user type on import
	rest   *client.RESTClient // REST: everything else
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
		MarkdownDescription: "Manages an Arize user. Creating one sends an email invitation with the Member account role; " +
			"until it is accepted the user has an invitation ID and status `invited`. " +
			"Destroying the user deactivates it in Arize, removing its access; destroying a pending invitation cancels it.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The unique identifier for the user.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"email": schema.StringAttribute{
				MarkdownDescription: "The email address of the user. Arize can't change a user's email, so changing it " +
					"replaces the user: the old user is deactivated and the new email gets an invitation. " +
					"Case-only changes are ignored.",
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplaceIf(
						func(ctx context.Context, req planmodifier.StringRequest, resp *stringplanmodifier.RequiresReplaceIfFuncResponse) {
							resp.RequiresReplace = emailChanged(req.StateValue.ValueString(), req.PlanValue.ValueString())
						},
						"Arize can't change a user's email, so a different email replaces the user.",
						"Arize can't change a user's email, so a different email replaces the user.",
					),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The full name of the user. Required to create a user.",
				Optional:            true,
			},
			"status": schema.StringAttribute{
				MarkdownDescription: "The status of the user: `active`, `invited`, or `expired`.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"user_type": schema.StringAttribute{
				MarkdownDescription: "The type of user (human or bot). Defaults to human.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"created_at": schema.StringAttribute{
				MarkdownDescription: "Timestamp when the user was created.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
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
	r.rest = client.NewREST(config.RESTEndpoint.ValueString(), config.APIKey.ValueString())
}

func (r *UserResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data UserResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if data.Name.ValueString() == "" {
		resp.Diagnostics.AddAttributeError(path.Root("name"), "Missing user name", "Arize requires a name to create a user.")
		return
	}

	// Creates an invitation (MEMBER account role, email link). Its ID is an
	// invitation ID until the user accepts; Read follows the change by email.
	user, err := r.rest.CreateUser(ctx, data.Name.ValueString(), data.Email.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to create user", err.Error())
		return
	}

	applyRESTUser(&data, user)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *UserResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data UserResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	user, err := r.rest.GetUserREST(ctx, data.ID.ValueString())
	if errors.Is(err, client.ErrUserNotFound) {
		// An accepted invitation becomes a user with a new ID: find it by email.
		user, err = r.rest.FindUserByEmail(ctx, data.Email.ValueString())
		if errors.Is(err, client.ErrUserNotFound) {
			resp.State.RemoveResource(ctx)
			return
		}
	}
	if err != nil {
		resp.Diagnostics.AddError("Failed to read user", err.Error())
		return
	}

	applyRESTUser(&data, user)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *UserResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state UserResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// An invitation keeps the configured name in state only; the name is
	// applied once the user has accepted and is active.
	if state.Status.ValueString() == "active" && !plan.Name.Equal(state.Name) {
		if err := r.rest.RenameUser(ctx, state.ID.ValueString(), plan.Name.ValueString()); err != nil {
			resp.Diagnostics.AddError("Failed to update user", err.Error())
			return
		}
	}

	plan.ID = state.ID
	plan.Status = state.Status
	plan.UserType = state.UserType
	plan.CreatedAt = state.CreatedAt
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *UserResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data UserResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Deactivates an active user (gone from REST, "inactive" in GraphQL) or
	// cancels a pending invitation.
	if err := r.rest.DeleteUser(ctx, data.ID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Failed to delete user", err.Error())
		return
	}
}

// ImportState takes a user ID or a pending invitation ID.
func (r *UserResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	user, err := r.rest.GetUserREST(ctx, req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Failed to import user", err.Error())
		return
	}

	data := UserResourceModel{Name: types.StringValue(user.Name), UserType: types.StringNull()}
	// REST doesn't report the user type; GraphQL does for active users.
	if gqlUser, err := r.client.GetUser(ctx, user.ID); err == nil && gqlUser.UserType != "" {
		data.UserType = types.StringValue(gqlUser.UserType)
	}
	applyRESTUser(&data, user)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// applyRESTUser copies a REST user or invitation into the model. Status is
// lowercased to match the values GraphQL stored before. For invitations the
// API reports the email as the name, so the existing name is kept until the
// user is active.
func applyRESTUser(data *UserResourceModel, user *client.RESTUser) {
	status := strings.ToLower(user.Status)
	data.ID = types.StringValue(user.ID)
	// Arize lowercases emails; keep the configured case so plans stay clean.
	if !strings.EqualFold(data.Email.ValueString(), user.Email) {
		data.Email = types.StringValue(user.Email)
	}
	data.Status = types.StringValue(status)
	data.CreatedAt = types.StringValue(user.CreatedAt)
	if status == "active" || data.Name.IsNull() || data.Name.ValueString() == "" {
		data.Name = types.StringValue(user.Name)
	}
	if data.UserType.IsNull() || data.UserType.IsUnknown() {
		data.UserType = types.StringValue("human")
	}
}

// emailChanged reports whether an email change needs a new user. Emails are
// compared case-insensitively, as Arize does.
func emailChanged(state, plan string) bool {
	return !strings.EqualFold(state, plan)
}
