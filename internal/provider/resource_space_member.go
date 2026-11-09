package provider

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/arize-ai/terraform-provider-arize/internal/client"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = (*SpaceMemberResource)(nil)
	_ resource.ResourceWithImportState = (*SpaceMemberResource)(nil)
)

func NewSpaceMemberResource() resource.Resource {
	return &SpaceMemberResource{}
}

type SpaceMemberResource struct {
	client *client.Client
}

type SpaceMemberResourceModel struct {
	ID      types.String `tfsdk:"id"`
	SpaceID types.String `tfsdk:"space_id"`
	UserID  types.String `tfsdk:"user_id"`
	Role    types.String `tfsdk:"role"`
	Email   types.String `tfsdk:"email"`
	Name    types.String `tfsdk:"name"`
}

func (r *SpaceMemberResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_space_member"
}

func (r *SpaceMemberResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a user's membership in an Arize space.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The unique identifier for the space membership.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"space_id": schema.StringAttribute{
				MarkdownDescription: "The ID of the space.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"user_id": schema.StringAttribute{
				MarkdownDescription: "The ID of the user.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"role": schema.StringAttribute{
				MarkdownDescription: "The role of the user in the space (admin, member, readOnly, annotator). " +
					"Users whose account role is Annotator can only be annotators, and other users can't be.",
				Required: true,
				Validators: []validator.String{
					stringvalidator.OneOf("admin", "member", "readOnly", "annotator"),
				},
			},
			"email": schema.StringAttribute{
				MarkdownDescription: "The email of the user (computed).",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the user (computed).",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *SpaceMemberResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *SpaceMemberResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data SpaceMemberResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if typ, _ := client.GlobalIDType(data.UserID.ValueString()); typ == client.GlobalIDTypeInvitation {
		resp.Diagnostics.AddAttributeError(path.Root("user_id"), "User hasn't accepted the invitation yet",
			"user_id is a pending invitation. Arize can only add users to a space after they accept the "+
				"invitation and sign up. Run terraform apply again once they have; arize_user then reads "+
				"their new user ID.")
		return
	}

	input := &client.AssignSpaceMembershipInput{
		SpaceMemberships: []client.SpaceMemberInput{
			{
				SpaceID: data.SpaceID.ValueString(),
				UserID:  data.UserID.ValueString(),
				Role:    data.Role.ValueString(),
			},
		},
	}

	members, err := r.client.AssignSpaceMembership(ctx, input)
	if err != nil {
		resp.Diagnostics.AddError("Failed to assign space membership", spaceRoleError(err))
		return
	}

	data.ID = types.StringNull()
	data.Email = types.StringNull()
	data.Name = types.StringNull()
	for _, m := range members {
		if m.User == nil || m.User.ID != data.UserID.ValueString() {
			continue
		}
		data.ID = types.StringValue(m.ID)
		data.Email = types.StringValue(m.User.Email)
		data.Name = types.StringValue(m.User.Name)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *SpaceMemberResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data SpaceMemberResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	su, err := r.client.GetSpaceUser(ctx, data.SpaceID.ValueString(), data.UserID.ValueString())
	if errors.Is(err, client.ErrSpaceMemberNotFound) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Failed to read space membership", err.Error())
		return
	}

	applySpaceUser(&data, su)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *SpaceMemberResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data SpaceMemberResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Re-assign with new role
	if typ, _ := client.GlobalIDType(data.UserID.ValueString()); typ == client.GlobalIDTypeInvitation {
		resp.Diagnostics.AddAttributeError(path.Root("user_id"), "User hasn't accepted the invitation yet",
			"user_id is a pending invitation. Arize can only add users to a space after they accept the "+
				"invitation and sign up. Run terraform apply again once they have; arize_user then reads "+
				"their new user ID.")
		return
	}

	input := &client.AssignSpaceMembershipInput{
		SpaceMemberships: []client.SpaceMemberInput{
			{
				SpaceID: data.SpaceID.ValueString(),
				UserID:  data.UserID.ValueString(),
				Role:    data.Role.ValueString(),
			},
		},
	}

	_, err := r.client.AssignSpaceMembership(ctx, input)
	if err != nil {
		resp.Diagnostics.AddError("Failed to update space membership", spaceRoleError(err))
		return
	}

	// Only the role changes in place; keep the computed values from state.
	var state SpaceMemberResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.ID = state.ID
	data.Email = state.Email
	data.Name = state.Name
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *SpaceMemberResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data SpaceMemberResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.RemoveSpaceMember(ctx, data.SpaceID.ValueString(), data.UserID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Failed to remove space member", err.Error())
	}
}

// applySpaceUser refreshes the model from the user's entry in the space. An
// account admin's entry hides any explicit membership (no role), so the state
// is kept as it is for those.
func applySpaceUser(data *SpaceMemberResourceModel, su *client.SpaceUser) {
	data.Email = types.StringValue(su.User.Email)
	data.Name = types.StringValue(su.User.Name)
	// Account admins are listed with no role; keep the configured one.
	if su.IsExplicit() && su.Role != nil {
		data.Role = types.StringValue(*su.Role)
	}
}

// spaceRoleError explains Arize's annotator rule, which its errors ("User's
// space role cannot be changed to annotator") don't.
func spaceRoleError(err error) string {
	msg := err.Error()
	if strings.Contains(msg, "annotator") && strings.Contains(msg, "cannot be changed") {
		msg += "\n\nArize ties the annotator space role to the user's account role: users with the " +
			"Annotator account role can only be annotators in spaces, and other users can't be annotators. " +
			"Change the role here, or the user's account role in Arize."
	}
	return msg
}

// ImportState takes "<space_id>:<user_id>" (IDs never contain ":").
func (r *SpaceMemberResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	spaceID, userID, ok := strings.Cut(req.ID, ":")
	if !ok || spaceID == "" || userID == "" {
		resp.Diagnostics.AddError("Invalid import ID",
			fmt.Sprintf("Expected \"<space_id>:<user_id>\", got %q.", req.ID))
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("space_id"), spaceID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("user_id"), userID)...)
}
