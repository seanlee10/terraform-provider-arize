package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/arize-ai/terraform-provider-arize/internal/client"
)

var _ resource.Resource = (*SAMLIdPResource)(nil)

func NewSAMLIdPResource() resource.Resource {
	return &SAMLIdPResource{}
}

type SAMLIdPResource struct {
	client *client.Client
}

type SAMLIdPResourceModel struct {
	ID                      types.String `tfsdk:"id"`
	MetadataURL             types.String `tfsdk:"metadata_url"`
	MetadataXML             types.String `tfsdk:"metadata_xml"`
	EnforceSAML             types.Bool   `tfsdk:"enforce_saml"`
	SyncUserRoles           types.Bool   `tfsdk:"sync_user_roles"`
	DefaultOrgID            types.String `tfsdk:"default_org_id"`
	DefaultOrgRoleID        types.String `tfsdk:"default_org_role_id"`
	DefaultSpaceID          types.String `tfsdk:"default_space_id"`
	DefaultSpaceRoleID      types.String `tfsdk:"default_space_role_id"`
	EmailDomainsList        types.List   `tfsdk:"email_domains_list"`
	AllowLoginWithDefaults  types.Bool   `tfsdk:"allow_login_with_defaults"`
	CreatedAt               types.String `tfsdk:"created_at"`
	UpdatedAt               types.String `tfsdk:"updated_at"`
}

func (r *SAMLIdPResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_saml_idp"
}

func (r *SAMLIdPResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a SAML Identity Provider for Arize SSO.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The unique identifier for the SAML IdP.",
				Computed:            true,
			},
			"metadata_url": schema.StringAttribute{
				MarkdownDescription: "URL to the SAML metadata.",
				Optional:            true,
			},
			"metadata_xml": schema.StringAttribute{
				MarkdownDescription: "Raw SAML metadata XML.",
				Optional:            true,
				Sensitive:           true,
			},
			"enforce_saml": schema.BoolAttribute{
				MarkdownDescription: "Whether to enforce SAML login.",
				Optional:            true,
			},
			"sync_user_roles": schema.BoolAttribute{
				MarkdownDescription: "Whether to sync user roles from SAML attributes.",
				Optional:            true,
			},
			"default_org_id": schema.StringAttribute{
				MarkdownDescription: "Default organization ID for SAML users.",
				Optional:            true,
			},
			"default_org_role_id": schema.StringAttribute{
				MarkdownDescription: "Default organization role ID for SAML users.",
				Optional:            true,
			},
			"default_space_id": schema.StringAttribute{
				MarkdownDescription: "Default space ID for SAML users.",
				Optional:            true,
			},
			"default_space_role_id": schema.StringAttribute{
				MarkdownDescription: "Default space role ID for SAML users.",
				Optional:            true,
			},
			"email_domains_list": schema.ListAttribute{
				MarkdownDescription: "List of email domains allowed for this SAML IdP.",
				ElementType:         types.StringType,
				Required:            true,
			},
			"allow_login_with_defaults": schema.BoolAttribute{
				MarkdownDescription: "Whether to allow login with default roles if no mapping exists.",
				Optional:            true,
			},
			"created_at": schema.StringAttribute{
				MarkdownDescription: "Timestamp when the SAML IdP was created.",
				Computed:            true,
			},
			"updated_at": schema.StringAttribute{
				MarkdownDescription: "Timestamp when the SAML IdP was last updated.",
				Computed:            true,
			},
		},
	}
}

func (r *SAMLIdPResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *SAMLIdPResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data SAMLIdPResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var emailDomains []string
	resp.Diagnostics.Append(data.EmailDomainsList.ElementsAs(ctx, &emailDomains, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	input := &client.CreateSAMLIdPInput{
		EmailDomainsList: emailDomains,
	}

	if !data.MetadataURL.IsNull() {
		url := data.MetadataURL.ValueString()
		input.MetadataURL = &url
	}
	if !data.MetadataXML.IsNull() {
		xml := data.MetadataXML.ValueString()
		input.MetadataXML = &xml
	}
	if !data.EnforceSAML.IsNull() {
		enforce := data.EnforceSAML.ValueBool()
		input.EnforceSAML = &enforce
	}
	if !data.SyncUserRoles.IsNull() {
		sync := data.SyncUserRoles.ValueBool()
		input.SyncUserRoles = &sync
	}
	if !data.DefaultOrgID.IsNull() {
		orgID := data.DefaultOrgID.ValueString()
		input.DefaultOrgID = &orgID
	}
	if !data.DefaultOrgRoleID.IsNull() {
		roleID := data.DefaultOrgRoleID.ValueString()
		input.DefaultOrgRoleID = &roleID
	}
	if !data.DefaultSpaceID.IsNull() {
		spaceID := data.DefaultSpaceID.ValueString()
		input.DefaultSpaceID = &spaceID
	}
	if !data.DefaultSpaceRoleID.IsNull() {
		roleID := data.DefaultSpaceRoleID.ValueString()
		input.DefaultSpaceRoleID = &roleID
	}
	if !data.AllowLoginWithDefaults.IsNull() {
		allow := data.AllowLoginWithDefaults.ValueBool()
		input.AllowLoginWithDefaults = &allow
	}

	samlIdP, err := r.client.CreateSAMLIdP(ctx, input)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create SAML IdP", err.Error())
		return
	}

	data.ID = types.StringValue(samlIdP.ID)
	data.CreatedAt = types.StringValue(samlIdP.CreatedAt)
	data.UpdatedAt = types.StringValue(samlIdP.UpdatedAt)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *SAMLIdPResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data SAMLIdPResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	samlIdP, err := r.client.GetSAMLIdP(ctx, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read SAML IdP", err.Error())
		return
	}

	data.MetadataURL = types.StringValue(samlIdP.MetadataURL)
	data.EnforceSAML = types.BoolValue(samlIdP.EnforceSAML)
	data.SyncUserRoles = types.BoolValue(samlIdP.SyncUserRoles)
	data.DefaultOrgID = types.StringValue(samlIdP.DefaultOrgID)
	data.DefaultOrgRoleID = types.StringValue(samlIdP.DefaultOrgRoleID)
	data.DefaultSpaceID = types.StringValue(samlIdP.DefaultSpaceID)
	data.DefaultSpaceRoleID = types.StringValue(samlIdP.DefaultSpaceRoleID)
	data.AllowLoginWithDefaults = types.BoolValue(samlIdP.AllowLoginWithDefaults)
	data.CreatedAt = types.StringValue(samlIdP.CreatedAt)
	data.UpdatedAt = types.StringValue(samlIdP.UpdatedAt)

	emailDomainsList, _ := types.ListValueFrom(ctx, types.StringType, samlIdP.EmailDomainsList)
	data.EmailDomainsList = emailDomainsList

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *SAMLIdPResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data SAMLIdPResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var emailDomains []string
	resp.Diagnostics.Append(data.EmailDomainsList.ElementsAs(ctx, &emailDomains, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	input := &client.UpdateSAMLIdPInput{
		ID:               data.ID.ValueString(),
		EmailDomainsList: emailDomains,
	}

	if !data.MetadataURL.IsNull() {
		url := data.MetadataURL.ValueString()
		input.MetadataURL = &url
	}
	if !data.MetadataXML.IsNull() {
		xml := data.MetadataXML.ValueString()
		input.MetadataXML = &xml
	}
	if !data.EnforceSAML.IsNull() {
		enforce := data.EnforceSAML.ValueBool()
		input.EnforceSAML = &enforce
	}
	if !data.SyncUserRoles.IsNull() {
		sync := data.SyncUserRoles.ValueBool()
		input.SyncUserRoles = &sync
	}
	if !data.DefaultOrgID.IsNull() {
		orgID := data.DefaultOrgID.ValueString()
		input.DefaultOrgID = &orgID
	}
	if !data.DefaultOrgRoleID.IsNull() {
		roleID := data.DefaultOrgRoleID.ValueString()
		input.DefaultOrgRoleID = &roleID
	}
	if !data.DefaultSpaceID.IsNull() {
		spaceID := data.DefaultSpaceID.ValueString()
		input.DefaultSpaceID = &spaceID
	}
	if !data.DefaultSpaceRoleID.IsNull() {
		roleID := data.DefaultSpaceRoleID.ValueString()
		input.DefaultSpaceRoleID = &roleID
	}
	if !data.AllowLoginWithDefaults.IsNull() {
		allow := data.AllowLoginWithDefaults.ValueBool()
		input.AllowLoginWithDefaults = &allow
	}

	samlIdP, err := r.client.UpdateSAMLIdP(ctx, input)
	if err != nil {
		resp.Diagnostics.AddError("Failed to update SAML IdP", err.Error())
		return
	}

	data.CreatedAt = types.StringValue(samlIdP.CreatedAt)
	data.UpdatedAt = types.StringValue(samlIdP.UpdatedAt)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *SAMLIdPResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	resp.Diagnostics.AddError(
		"SAML IdP deletion not supported",
		"Arize does not allow deleting SAML identity providers. You can disable it by setting enforce_saml=false.",
	)
}

func (r *SAMLIdPResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	samlIdP, err := r.client.GetSAMLIdP(ctx, req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Failed to import SAML IdP", err.Error())
		return
	}

	var data SAMLIdPResourceModel
	data.ID = types.StringValue(samlIdP.ID)
	data.MetadataURL = types.StringValue(samlIdP.MetadataURL)
	data.EnforceSAML = types.BoolValue(samlIdP.EnforceSAML)
	data.SyncUserRoles = types.BoolValue(samlIdP.SyncUserRoles)
	data.DefaultOrgID = types.StringValue(samlIdP.DefaultOrgID)
	data.DefaultOrgRoleID = types.StringValue(samlIdP.DefaultOrgRoleID)
	data.DefaultSpaceID = types.StringValue(samlIdP.DefaultSpaceID)
	data.DefaultSpaceRoleID = types.StringValue(samlIdP.DefaultSpaceRoleID)
	data.AllowLoginWithDefaults = types.BoolValue(samlIdP.AllowLoginWithDefaults)
	data.CreatedAt = types.StringValue(samlIdP.CreatedAt)
	data.UpdatedAt = types.StringValue(samlIdP.UpdatedAt)

	emailDomainsList, _ := types.ListValueFrom(ctx, types.StringType, samlIdP.EmailDomainsList)
	data.EmailDomainsList = emailDomainsList

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
