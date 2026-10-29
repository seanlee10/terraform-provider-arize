package provider

import (
	"context"
	"fmt"

	"github.com/arize-ai/terraform-provider-arize/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = (*SpaceResource)(nil)

func NewSpaceResource() resource.Resource {
	return &SpaceResource{}
}

type SpaceResource struct {
	client     *client.Client
	restClient *client.RESTClient
}

type SpaceResourceModel struct {
	ID             types.String `tfsdk:"id"`
	Name           types.String `tfsdk:"name"`
	UUID           types.String `tfsdk:"uuid"`
	Description    types.String `tfsdk:"description"`
	Private        types.Bool   `tfsdk:"private"`
	CreatedAt      types.String `tfsdk:"created_at"`
	OrganizationID types.String `tfsdk:"organization_id"`
	Organization   types.Object `tfsdk:"organization"`
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
				MarkdownDescription: "The ID of the organization to create the space in. Defaults to the account's first organization. Changing this forces a new space.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplace(),
				},
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
	if data.OrganizationID.IsUnknown() || orgID == "" {
		// No organization configured: use the account's first organization
		var err error
		orgID, err = r.getDefaultOrganizationID(ctx)
		if err != nil {
			resp.Diagnostics.AddError("Failed to get organization", err.Error())
			return
		}
	}
	data.OrganizationID = types.StringValue(orgID)

	input := &client.CreateSpaceInput{
		Name:                  data.Name.ValueString(),
		AccountOrganizationID: orgID,
		Private:               data.Private.ValueBool(),
		Description:           data.Description.ValueString(),
	}

	space, err := r.client.CreateSpace(ctx, input)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create space", err.Error())
		return
	}

	data.ID = types.StringValue(space.ID)
	data.UUID = types.StringValue(space.UUID)
	data.CreatedAt = types.StringValue(space.CreatedAt)

	// Map organization if present
	if space.Organization != nil {
		data.OrganizationID = types.StringValue(space.Organization.ID)
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
		data.OrganizationID = types.StringValue(space.Organization.ID)
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
	resp.Diagnostics.AddError(
		"Space updates not yet supported",
		"Space updates will be implemented in a future version.",
	)
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

func (r *SpaceResource) getDefaultOrganizationID(ctx context.Context) (string, error) {
	return r.client.GetAccountOrganization(ctx)
}
