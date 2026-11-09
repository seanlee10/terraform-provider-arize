package provider

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/arize-ai/terraform-provider-arize/internal/client"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = (*SpaceResource)(nil)
	_ resource.ResourceWithModifyPlan  = (*SpaceResource)(nil)
	_ resource.ResourceWithImportState = (*SpaceResource)(nil)
)

func NewSpaceResource() resource.Resource {
	return &SpaceResource{}
}

type SpaceResource struct {
	client     *client.Client
	restClient *client.RESTClient
}

type SpaceResourceModel struct {
	ID               types.String `tfsdk:"id"`
	Name             types.String `tfsdk:"name"`
	UUID             types.String `tfsdk:"uuid"`
	Description      types.String `tfsdk:"description"`
	Private          types.Bool   `tfsdk:"private"`
	CreatedAt        types.String `tfsdk:"created_at"`
	OrganizationID   types.String `tfsdk:"organization_id"`
	OrganizationName types.String `tfsdk:"organization_name"`
	Organization     types.Object `tfsdk:"organization"`
}

func (r *SpaceResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_space"
}

func (r *SpaceResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages an Arize space.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The unique identifier for the space.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the space.",
				Required:            true,
			},
			"uuid": schema.StringAttribute{
				MarkdownDescription: "The UUID of the space.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "A description of the space.",
				Optional:            true,
			},
			"private": schema.BoolAttribute{
				MarkdownDescription: "Whether the space is private.",
				Required:            true,
			},
			"created_at": schema.StringAttribute{
				MarkdownDescription: "Timestamp when the space was created.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"organization_id": schema.StringAttribute{
				MarkdownDescription: "The ID of the organization to create the space in (listed by `ax organizations list`). " +
					"Set this or `organization_name`; with neither, the account's first organization is used. " +
					"With `organization_name`, the looked-up ID is stored here. Changing the organization forces a new space.",
				Optional: true,
				Computed: true,
				Validators: []validator.String{
					stringvalidator.ConflictsWith(path.MatchRoot("organization_name")),
					stringvalidator.LengthAtLeast(1),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplaceIf(organizationChanged,
						"A different organization replaces the space.",
						"A different organization replaces the space."),
				},
			},
			"organization_name": schema.StringAttribute{
				MarkdownDescription: "The name of the organization to create the space in, as shown in Arize. " +
					"Set this or `organization_id`. Changing it to a different organization forces a new space.",
				Optional:   true,
				Validators: []validator.String{stringvalidator.LengthAtLeast(1)},
			},
			"organization": schema.SingleNestedAttribute{
				MarkdownDescription: "The organization that owns the space.",
				Computed:            true,
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.UseStateForUnknown(),
				},
				Attributes: map[string]schema.Attribute{
					"id": schema.StringAttribute{
						Computed: true,
					},
					"name": schema.StringAttribute{
						Computed: true,
					},
				},
			},
		},
	}
}

func (r *SpaceResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
	r.restClient = client.NewREST(config.RESTEndpoint.ValueString(), config.APIKey.ValueString())
}

func (r *SpaceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data SpaceResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	orgID := data.OrganizationID.ValueString()
	source := "organization_id"
	var err error
	if !data.OrganizationName.IsNull() {
		source = "organization_name"
		if orgID, err = resolveOrganizationID(ctx, r.client, data.OrganizationName.ValueString()); err != nil {
			resp.Diagnostics.AddAttributeError(path.Root("organization_name"), "Invalid organization_name", err.Error())
			return
		}
		data.OrganizationID = types.StringValue(orgID)
	} else if data.OrganizationID.IsUnknown() || orgID == "" {
		source = ""
		// No organization configured: use the account's first organization
		orgID, err = r.getDefaultOrganizationID(ctx)
		if err != nil {
			resp.Diagnostics.AddError("Failed to get organization", err.Error())
			return
		}
		data.OrganizationID = types.StringValue(orgID)
	} else if orgID, err = resolveOrganizationID(ctx, r.client, orgID); err != nil {
		// A name is sent as its ID; state keeps the configured value.
		resp.Diagnostics.AddAttributeError(path.Root("organization_id"), "Invalid organization_id", err.Error())
		return
	}

	input := &client.CreateSpaceInput{
		Name:                  data.Name.ValueString(),
		AccountOrganizationID: orgID,
		Private:               data.Private.ValueBool(),
		Description:           data.Description.ValueString(),
	}

	space, err := r.client.CreateSpace(ctx, input)
	if err != nil {
		detail := err.Error()
		if client.IsUnauthorized(err) {
			// Arize says only "Not Authorized"; explain who tried, where.
			if access, aerr := r.client.GetOrganizationAccess(ctx); aerr == nil {
				detail = spaceCreateUnauthorizedDetail(orgID, source, access)
			}
		}
		resp.Diagnostics.AddError("Failed to create space", detail)
		return
	}

	data.ID = types.StringValue(space.ID)
	data.UUID = types.StringValue(space.UUID)
	data.CreatedAt = types.StringValue(space.CreatedAt)

	// Map organization if present
	if space.Organization != nil {
		data.OrganizationID = organizationIDState(data.OrganizationID, space.Organization)
		// Keep a configured name (any case) while it names the space's organization.
		if !data.OrganizationName.IsNull() && !strings.EqualFold(data.OrganizationName.ValueString(), space.Organization.Name) {
			data.OrganizationName = types.StringValue(space.Organization.Name)
		}
		orgMap := map[string]attr.Value{
			"id":   types.StringValue(space.Organization.ID),
			"name": types.StringValue(space.Organization.Name),
		}
		orgObj, _ := types.ObjectValue(map[string]attr.Type{
			"id":   types.StringType,
			"name": types.StringType,
		}, orgMap)
		data.Organization = orgObj
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *SpaceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data SpaceResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	space, err := r.client.GetSpace(ctx, data.ID.ValueString())
	if errors.Is(err, client.ErrSpaceNotFound) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Failed to read space", err.Error())
		return
	}

	data.Name = types.StringValue(space.Name)
	data.UUID = types.StringValue(space.UUID)
	// An unset description comes back as ""; keep it null so plans stay clean
	if space.Description != "" || !data.Description.IsNull() {
		data.Description = types.StringValue(space.Description)
	}
	data.Private = types.BoolValue(space.Private)
	data.CreatedAt = types.StringValue(space.CreatedAt)

	// Map organization if present
	if space.Organization != nil {
		data.OrganizationID = organizationIDState(data.OrganizationID, space.Organization)
		// Keep a configured name (any case) while it names the space's organization.
		if !data.OrganizationName.IsNull() && !strings.EqualFold(data.OrganizationName.ValueString(), space.Organization.Name) {
			data.OrganizationName = types.StringValue(space.Organization.Name)
		}
		orgMap := map[string]attr.Value{
			"id":   types.StringValue(space.Organization.ID),
			"name": types.StringValue(space.Organization.Name),
		}
		orgObj, _ := types.ObjectValue(map[string]attr.Type{
			"id":   types.StringType,
			"name": types.StringType,
		}, orgMap)
		data.Organization = orgObj
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *SpaceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state SpaceResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// REST, like DeleteSpace: the provider never used GraphQL updateSpace.
	// A null description is sent as null, which clears it.
	err := r.restClient.UpdateSpace(ctx, state.ID.ValueString(), &client.UpdateSpaceInput{
		Name:        plan.Name.ValueString(),
		Description: nonEmpty(plan.Description.ValueStringPointer()), // the API rejects ""
		IsPrivate:   plan.Private.ValueBool(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to update space", err.Error())
		return
	}

	plan.ID = state.ID
	plan.UUID = state.UUID
	plan.CreatedAt = state.CreatedAt
	// organization_id can only change in place between the ID and the name
	// of the same organization (see organizationChanged); keep what's planned.
	if plan.OrganizationID.IsUnknown() {
		plan.OrganizationID = state.OrganizationID
	}
	plan.Organization = state.Organization
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *SpaceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data SpaceResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.restClient.DeleteSpace(ctx, data.ID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Failed to delete space", err.Error())
		return
	}
}

// ImportState takes a space ID. Read fills in the rest; organization_id is
// stored as the organization's ID.
func (r *SpaceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// organizationIDState is the organization_id to store: the configured value
// when it names or identifies the space's organization (so a name in the
// configuration doesn't show a change), otherwise the organization's ID.
func organizationIDState(current types.String, org *client.Org) types.String {
	v := current.ValueString()
	if !current.IsNull() && !current.IsUnknown() && (v == org.ID || strings.EqualFold(v, org.Name)) {
		return current
	}
	return types.StringValue(org.ID)
}

// organizationChanged reports whether a new organization_id points to a
// different organization than the space's. Switching between the ID and the
// name of the same organization doesn't replace the space.
func organizationChanged(ctx context.Context, req planmodifier.StringRequest, resp *stringplanmodifier.RequiresReplaceIfFuncResponse) {
	var org struct {
		ID   types.String `tfsdk:"id"`
		Name types.String `tfsdk:"name"`
	}
	resp.RequiresReplace = true
	if diags := req.State.GetAttribute(ctx, path.Root("organization"), &org); diags.HasError() || org.ID.IsNull() {
		return
	}
	v := req.PlanValue.ValueString()
	if v == org.ID.ValueString() || strings.EqualFold(v, org.Name.ValueString()) {
		resp.RequiresReplace = false
	}
}

// ModifyPlan checks configured organization names at plan time. For
// organization_name it plans the organization's ID, and replaces the space
// when the name points to a different organization.
func (r *SpaceResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.Plan.Raw.IsNull() || r.client == nil {
		return
	}
	var org, name types.String
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("organization_id"), &org)...)
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("organization_name"), &name)...)
	if !name.IsNull() && !name.IsUnknown() {
		r.planOrganizationName(ctx, req, resp, name.ValueString())
		return
	}

	// A name in organization_id (accepted since v0.3.4).
	if org.IsNull() || org.IsUnknown() || org.ValueString() == "" || isArizeID(org.ValueString()) {
		return
	}
	if !req.State.Raw.IsNull() {
		var prior types.String
		resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("organization_id"), &prior)...)
		if prior.Equal(org) {
			return // unchanged; already resolved when the space was created
		}
	}
	if _, err := resolveOrganizationID(ctx, r.client, org.ValueString()); err != nil {
		resp.Diagnostics.AddAttributeError(path.Root("organization_id"), "Invalid organization_id", err.Error())
	}
}

func (r *SpaceResource) planOrganizationName(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse, name string) {
	var prior struct {
		ID   types.String `tfsdk:"id"`
		Name types.String `tfsdk:"name"`
	}
	hasPrior := false
	if !req.State.Raw.IsNull() {
		d := req.State.GetAttribute(ctx, path.Root("organization"), &prior)
		hasPrior = !d.HasError() && !prior.ID.IsNull()
	}
	if hasPrior && strings.EqualFold(name, prior.Name.ValueString()) {
		resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, path.Root("organization_id"), prior.ID)...)
		return
	}

	id, err := resolveOrganizationID(ctx, r.client, name)
	if err != nil {
		resp.Diagnostics.AddAttributeError(path.Root("organization_name"), "Invalid organization_name", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, path.Root("organization_id"), types.StringValue(id))...)
	if hasPrior && id != prior.ID.ValueString() {
		resp.RequiresReplace = append(resp.RequiresReplace, path.Root("organization_name"))
	}
}

func (r *SpaceResource) getDefaultOrganizationID(ctx context.Context) (string, error) {
	return r.client.GetAccountOrganization(ctx)
}
