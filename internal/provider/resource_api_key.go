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

var _ resource.Resource = (*APIKeyResource)(nil)
var _ resource.ResourceWithImportState = (*APIKeyResource)(nil)

func NewAPIKeyResource() resource.Resource {
	return &APIKeyResource{}
}

type APIKeyResource struct {
	client *client.Client
}

type APIKeyResourceModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Key         types.String `tfsdk:"key"`
	Permissions types.List   `tfsdk:"permissions"`
	ExpiresAt   types.String `tfsdk:"expires_at"`
	CreatedAt   types.String `tfsdk:"created_at"`
	LastUsedAt  types.String `tfsdk:"last_used_at"`
}

func (r *APIKeyResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_api_key"
}

func (r *APIKeyResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages an Arize API key.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The unique identifier for the API key.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the API key.",
				Required:            true,
			},
			"key": schema.StringAttribute{
				MarkdownDescription: "The actual API key value. Only populated when the key is first created.",
				Computed:            true,
				Sensitive:           true,
			},
			"permissions": schema.ListAttribute{
				ElementType:         types.StringType,
				MarkdownDescription: "List of permissions for this API key.",
				Optional:            true,
			},
			"expires_at": schema.StringAttribute{
				MarkdownDescription: "Optional expiration timestamp for the API key.",
				Optional:            true,
			},
			"created_at": schema.StringAttribute{
				MarkdownDescription: "Timestamp when the API key was created.",
				Computed:            true,
			},
			"last_used_at": schema.StringAttribute{
				MarkdownDescription: "Timestamp of last use.",
				Computed:            true,
			},
		},
	}
}

func (r *APIKeyResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *APIKeyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data APIKeyResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var permissions []string
	resp.Diagnostics.Append(data.Permissions.ElementsAs(ctx, &permissions, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	input := &client.CreateAPIKeyInput{
		Name:        data.Name.ValueString(),
		Permissions: permissions,
		ExpiresAt:   data.ExpiresAt.ValueString(),
	}

	apiKey, err := r.client.CreateAPIKey(ctx, input)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create API key", err.Error())
		return
	}

	data.ID = types.StringValue(apiKey.ID)
	data.Key = types.StringValue(apiKey.Key)
	data.CreatedAt = types.StringValue(apiKey.CreatedAt)
	if apiKey.LastUsedAt != "" {
		data.LastUsedAt = types.StringValue(apiKey.LastUsedAt)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *APIKeyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data APIKeyResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiKey, err := r.client.GetAPIKey(ctx, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read API key", err.Error())
		return
	}

	data.Name = types.StringValue(apiKey.Name)
	permsValue, _ := types.ListValueFrom(ctx, types.StringType, apiKey.Permissions)
	data.Permissions = permsValue
	if apiKey.ExpiresAt != "" {
		data.ExpiresAt = types.StringValue(apiKey.ExpiresAt)
	}
	data.CreatedAt = types.StringValue(apiKey.CreatedAt)
	if apiKey.LastUsedAt != "" {
		data.LastUsedAt = types.StringValue(apiKey.LastUsedAt)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *APIKeyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError(
		"API Keys cannot be updated",
		"To change an API key's name, delete it and create a new one.",
	)
}

func (r *APIKeyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data APIKeyResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.DeleteAPIKey(ctx, data.ID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Failed to delete API key", err.Error())
		return
	}
}

func (r *APIKeyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	keyID := req.ID
	apiKey, err := r.client.GetAPIKey(ctx, keyID)
	if err != nil {
		resp.Diagnostics.AddError("Failed to import API key", err.Error())
		return
	}

	permsValue, _ := types.ListValueFrom(ctx, types.StringType, apiKey.Permissions)
	data := APIKeyResourceModel{
		ID:          types.StringValue(apiKey.ID),
		Name:        types.StringValue(apiKey.Name),
		Key:         types.StringNull(),
		Permissions: permsValue,
		ExpiresAt:   types.StringValue(apiKey.ExpiresAt),
		CreatedAt:   types.StringValue(apiKey.CreatedAt),
		LastUsedAt:  types.StringValue(apiKey.LastUsedAt),
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
