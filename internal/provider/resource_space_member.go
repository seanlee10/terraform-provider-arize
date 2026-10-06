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

var _ resource.Resource = (*SpaceMemberResource)(nil)

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
				MarkdownDescription: "The role of the user in the space (admin, member, readOnly, annotator).",
				Required:            true,
			},
			"email": schema.StringAttribute{
				MarkdownDescription: "The email of the user (computed).",
				Computed:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the user (computed).",
				Computed:            true,
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
		resp.Diagnostics.AddError("Failed to assign space membership", err.Error())
		return
	}

	if len(members) > 0 {
		data.ID = types.StringValue(members[0].ID)
		if members[0].User != nil {
			data.Email = types.StringValue(members[0].User.Email)
			data.Name = types.StringValue(members[0].User.Name)
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *SpaceMemberResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data SpaceMemberResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// For now, we just return the state. In a real implementation,
	// we'd query the API to verify the membership still exists.
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *SpaceMemberResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data SpaceMemberResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Re-assign with new role
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
		resp.Diagnostics.AddError("Failed to update space membership", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *SpaceMemberResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data SpaceMemberResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// For now, we don't have a removeSpaceMember mutation implemented.
	// In a real implementation, we'd call the API to remove the user from the space.
	// For testing purposes, we'll just remove from state.
}
