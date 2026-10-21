package provider

import (
	"context"
	"errors"
	"fmt"
	"sort"

	"github.com/arize-ai/terraform-provider-arize/internal/client"
	"github.com/hashicorp/terraform-plugin-framework-validators/mapvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/resourcevalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = (*SAMLIdPResource)(nil)
var _ resource.ResourceWithImportState = (*SAMLIdPResource)(nil)
var _ resource.ResourceWithConfigValidators = (*SAMLIdPResource)(nil)

// Built-in role names accepted for org/space defaults and role mappings.
// Custom roles (arize_role) are only accepted per space or project.
var builtinRoles = []string{"admin", "member", "readOnly", "annotator"}

func NewSAMLIdPResource() resource.Resource {
	return &SAMLIdPResource{}
}

type SAMLIdPResource struct {
	client *client.Client
}

type SAMLIdPResourceModel struct {
	ID                     types.String `tfsdk:"id"`
	MetadataURL            types.String `tfsdk:"metadata_url"`
	MetadataXML            types.String `tfsdk:"metadata_xml"`
	EnforceSAML            types.Bool   `tfsdk:"enforce_saml"`
	SyncUserRoles          types.Bool   `tfsdk:"sync_user_roles"`
	SignAuthn              types.Bool   `tfsdk:"sign_authn"`
	DefaultOrgID           types.String `tfsdk:"default_org_id"`
	DefaultOrgRoleID       types.String `tfsdk:"default_org_role_id"`
	DefaultSpaceID         types.String `tfsdk:"default_space_id"`
	DefaultSpaceRoleID     types.String `tfsdk:"default_space_role_id"`
	EmailDomainsList       types.Set    `tfsdk:"email_domains_list"`
	AllowLoginWithDefaults types.Bool   `tfsdk:"allow_login_with_defaults"`
	RoleMappings           types.List   `tfsdk:"role_mappings"`
	CreatedAt              types.String `tfsdk:"created_at"`
	UpdatedAt              types.String `tfsdk:"updated_at"`
}

type RoleMappingModel struct {
	Attributes       types.Map    `tfsdk:"attributes"`
	IsAccountAdmin   types.Bool   `tfsdk:"is_account_admin"`
	AccountRole      types.String `tfsdk:"account_role"`
	OrgID            types.String `tfsdk:"org_id"`
	OrgRole          types.String `tfsdk:"org_role"`
	SpaceRoles       types.Map    `tfsdk:"space_roles"`
	SpaceCustomRoles types.Map    `tfsdk:"space_custom_roles"`
	ProjectRoles     types.Map    `tfsdk:"project_roles"`
}

var roleMappingAttrTypes = map[string]attr.Type{
	"attributes":         types.MapType{ElemType: types.StringType},
	"is_account_admin":   types.BoolType,
	"account_role":       types.StringType,
	"org_id":             types.StringType,
	"org_role":           types.StringType,
	"space_roles":        types.MapType{ElemType: types.StringType},
	"space_custom_roles": types.MapType{ElemType: types.StringType},
	"project_roles":      types.MapType{ElemType: types.StringType},
}

func (r *SAMLIdPResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_saml_idp"
}

func (r *SAMLIdPResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	// Every apply sends the whole IdP (updateSAMLIdP replaces it), so removing
	// an optional setting from the config clears it in Arize.
	optionalBool := func(desc string) schema.BoolAttribute {
		return schema.BoolAttribute{
			MarkdownDescription: desc + " Defaults to `false`.",
			Optional:            true,
			Computed:            true,
			Default:             booldefault.StaticBool(false),
		}
	}
	builtinRole := []validator.String{stringvalidator.OneOf(builtinRoles...)}
	idMap := func(desc string, values ...validator.String) schema.MapAttribute {
		return schema.MapAttribute{
			MarkdownDescription: desc,
			ElementType:         types.StringType,
			Optional:            true,
			Validators:          []validator.Map{mapvalidator.SizeAtLeast(1), mapvalidator.ValueStringsAre(values...)},
		}
	}

	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a SAML Identity Provider for Arize SSO. The Arize API cannot delete an IdP: `terraform destroy` removes it from state only, and it must be deleted in the Arize UI.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The unique identifier for the SAML IdP.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"metadata_url": schema.StringAttribute{
				MarkdownDescription: "URL to the SAML metadata. Exactly one of `metadata_url` or `metadata_xml` is required.",
				Optional:            true,
			},
			"metadata_xml": schema.StringAttribute{
				MarkdownDescription: "Raw SAML metadata XML. Exactly one of `metadata_url` or `metadata_xml` is required.",
				Optional:            true,
				Sensitive:           true,
			},
			"email_domains_list": schema.SetAttribute{
				MarkdownDescription: "Email domains handled by this IdP.",
				ElementType:         types.StringType,
				Required:            true,
				Validators:          []validator.Set{setvalidator.SizeAtLeast(1)},
			},
			"enforce_saml":    optionalBool("Whether to enforce SAML login for the account."),
			"sync_user_roles": optionalBool("Whether to re-apply role mappings on every login."),
			"sign_authn":      optionalBool("Whether to sign SAML authentication requests."),
			"allow_login_with_defaults": optionalBool(
				"Whether users matching no role mapping can log in with the `default_*` roles. " +
					"When `true`, all four `default_*` attributes are required; when `false`, at least one `role_mappings` entry is required.",
			),
			"default_org_id": schema.StringAttribute{
				MarkdownDescription: "Organization that users matching no role mapping join.",
				Optional:            true,
			},
			"default_org_role_id": schema.StringAttribute{
				MarkdownDescription: "Built-in organization role for `default_org_id`: `admin`, `member`, `readOnly`, or `annotator`. Custom roles are not accepted.",
				Optional:            true,
				Validators:          builtinRole,
			},
			"default_space_id": schema.StringAttribute{
				MarkdownDescription: "Space that users matching no role mapping join.",
				Optional:            true,
			},
			"default_space_role_id": schema.StringAttribute{
				MarkdownDescription: "Built-in space role for `default_space_id`: `admin`, `member`, `readOnly`, or `annotator`. Custom roles are not accepted.",
				Optional:            true,
				Validators:          builtinRole,
			},
			"role_mappings": schema.ListNestedAttribute{
				MarkdownDescription: "Roles granted to users whose SAML assertion attributes match.",
				Optional:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"attributes": schema.MapAttribute{
							MarkdownDescription: "SAML attribute names and the values a user must have, e.g. `{ groups = \"data-science\" }`.",
							ElementType:         types.StringType,
							Required:            true,
							Validators:          []validator.Map{mapvalidator.SizeAtLeast(1)},
						},
						"is_account_admin": schema.BoolAttribute{
							MarkdownDescription: "Make matching users account admins. Defaults to `false`.",
							Optional:            true,
							Computed:            true,
							Default:             booldefault.StaticBool(false),
						},
						"account_role": schema.StringAttribute{
							MarkdownDescription: "Account role: `admin`, `member`, `member_ui_only`, or `annotator`. Arize treats an unset value as `member`.",
							Optional:            true,
							Validators:          []validator.String{stringvalidator.OneOf("admin", "member", "member_ui_only", "annotator")},
						},
						"org_id": schema.StringAttribute{
							MarkdownDescription: "Organization that matching users join. Required unless `is_account_admin` is `true`.",
							Optional:            true,
							Validators:          []validator.String{stringvalidator.AlsoRequires(path.MatchRelative().AtParent().AtName("org_role"))},
						},
						"org_role": schema.StringAttribute{
							MarkdownDescription: "Built-in role in `org_id`: `admin`, `member`, `readOnly`, or `annotator`.",
							Optional:            true,
							Validators: append([]validator.String{
								stringvalidator.AlsoRequires(path.MatchRelative().AtParent().AtName("org_id")),
							}, builtinRole...),
						},
						"space_roles":        idMap("Built-in roles by space ID, e.g. `{ (arize_space.ml.id) = \"member\" }`.", builtinRole...),
						"space_custom_roles": idMap("Custom roles by space ID: `{ (arize_space.ml.id) = arize_role.data_scientist.id }`. Use instead of `space_roles` for a given mapping."),
						"project_roles":      idMap("Custom or predefined project roles by project ID. Requires `org_role`."),
					},
				},
			},
			"created_at": schema.StringAttribute{
				MarkdownDescription: "Timestamp when the SAML IdP was created.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"updated_at": schema.StringAttribute{
				MarkdownDescription: "Timestamp when the SAML IdP was last updated.",
				Computed:            true,
			},
		},
	}
}

func (r *SAMLIdPResource) ConfigValidators(ctx context.Context) []resource.ConfigValidator {
	return []resource.ConfigValidator{
		resourcevalidator.ExactlyOneOf(path.MatchRoot("metadata_url"), path.MatchRoot("metadata_xml")),
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

	input, diags := samlIdPInput(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	idp, err := r.client.CreateSAMLIdP(ctx, input)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create SAML IdP", err.Error())
		return
	}

	resp.Diagnostics.Append(applySAMLIdP(ctx, &data, idp)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *SAMLIdPResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data SAMLIdPResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	idp, err := r.client.GetSAMLIdP(ctx, data.ID.ValueString())
	if errors.Is(err, client.ErrSAMLIdPNotFound) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Failed to read SAML IdP", err.Error())
		return
	}

	resp.Diagnostics.Append(applySAMLIdP(ctx, &data, idp)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *SAMLIdPResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data SAMLIdPResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	input, diags := samlIdPInput(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	input.ID = data.ID.ValueString()

	idp, err := r.client.UpdateSAMLIdP(ctx, input)
	if err != nil {
		resp.Diagnostics.AddError("Failed to update SAML IdP", err.Error())
		return
	}

	resp.Diagnostics.Append(applySAMLIdP(ctx, &data, idp)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *SAMLIdPResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	resp.Diagnostics.AddWarning(
		"SAML IdP removed from state only",
		"The Arize API cannot delete SAML identity providers. The IdP still exists in Arize; delete it in the Arize UI.",
	)
}

func (r *SAMLIdPResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func samlIdPInput(ctx context.Context, data *SAMLIdPResourceModel) (*client.SAMLIdPInput, diag.Diagnostics) {
	var diags diag.Diagnostics

	var domains []string
	diags.Append(data.EmailDomainsList.ElementsAs(ctx, &domains, false)...)

	input := &client.SAMLIdPInput{
		MetadataURL:            data.MetadataURL.ValueStringPointer(),
		MetadataXML:            data.MetadataXML.ValueStringPointer(),
		EnforceSAML:            data.EnforceSAML.ValueBoolPointer(),
		SyncUserRoles:          data.SyncUserRoles.ValueBoolPointer(),
		SignAuthn:              data.SignAuthn.ValueBoolPointer(),
		DefaultOrgID:           data.DefaultOrgID.ValueStringPointer(),
		DefaultOrgRoleID:       data.DefaultOrgRoleID.ValueStringPointer(),
		DefaultSpaceID:         data.DefaultSpaceID.ValueStringPointer(),
		DefaultSpaceRoleID:     data.DefaultSpaceRoleID.ValueStringPointer(),
		AllowLoginWithDefaults: data.AllowLoginWithDefaults.ValueBoolPointer(),
		RoleMappings:           &client.RoleMappingsInput{MappingsList: []client.RoleMapping{}},
	}
	for _, d := range domains {
		input.EmailDomainsList = append(input.EmailDomainsList, client.EmailDomain{Domain: d})
	}

	var mappings []RoleMappingModel
	diags.Append(data.RoleMappings.ElementsAs(ctx, &mappings, false)...)
	for _, m := range mappings {
		mapping := client.RoleMapping{
			IsAccountAdmin:      m.IsAccountAdmin.ValueBoolPointer(),
			AccountRole:         m.AccountRole.ValueStringPointer(),
			AttributesMap:       mapToTuples(ctx, m.Attributes, &diags),
			SpaceRolesMap:       mapToTuples(ctx, m.SpaceRoles, &diags),
			SpaceRbacRolesMap:   mapToTuples(ctx, m.SpaceCustomRoles, &diags),
			ProjectRolesMap:     mapToTuples(ctx, m.ProjectRoles, &diags),
			ProjectRolesManaged: !m.ProjectRoles.IsNull(),
		}
		if !m.OrgID.IsNull() {
			mapping.OrgRole = &client.OrgRole{OrgID: m.OrgID.ValueString(), RoleID: m.OrgRole.ValueString()}
		}
		input.RoleMappings.MappingsList = append(input.RoleMappings.MappingsList, mapping)
	}

	return input, diags
}

func applySAMLIdP(ctx context.Context, data *SAMLIdPResourceModel, idp *client.SAMLIdP) diag.Diagnostics {
	var diags diag.Diagnostics

	domains, d := types.SetValueFrom(ctx, types.StringType, idp.Domains())
	diags.Append(d...)

	data.ID = types.StringValue(idp.ID)
	// Arize keeps both metadata fields; track the one the config uses, or the
	// URL when importing.
	useURL, useXML := !data.MetadataURL.IsNull(), !data.MetadataXML.IsNull()
	if !useURL && !useXML {
		useURL = nonEmpty(idp.MetadataURL) != nil
		useXML = !useURL
	}
	if useURL {
		data.MetadataURL = types.StringPointerValue(nonEmpty(idp.MetadataURL))
	}
	if useXML {
		data.MetadataXML = types.StringPointerValue(nonEmpty(idp.MetadataXML))
	}
	data.EnforceSAML = types.BoolValue(idp.EnforceSAML)
	data.SyncUserRoles = types.BoolValue(idp.SyncUserRoles)
	data.SignAuthn = types.BoolValue(idp.SignAuthn)
	data.DefaultOrgID = types.StringPointerValue(nonEmpty(idp.DefaultOrgID))
	data.DefaultOrgRoleID = types.StringPointerValue(nonEmpty(idp.DefaultOrgRoleID))
	data.DefaultSpaceID = types.StringPointerValue(nonEmpty(idp.DefaultSpaceID))
	data.DefaultSpaceRoleID = types.StringPointerValue(nonEmpty(idp.DefaultSpaceRoleID))
	data.EmailDomainsList = domains
	data.AllowLoginWithDefaults = types.BoolValue(idp.AllowLoginWithDefaults)
	data.CreatedAt = types.StringValue(idp.CreatedAt)
	data.UpdatedAt = types.StringValue(idp.UpdatedAt)

	elemType := types.ObjectType{AttrTypes: roleMappingAttrTypes}
	if len(idp.RoleMappings) == 0 {
		data.RoleMappings = types.ListNull(elemType)
		return diags
	}
	mappings := make([]RoleMappingModel, 0, len(idp.RoleMappings))
	for _, m := range idp.RoleMappings {
		model := RoleMappingModel{
			Attributes:       tuplesToMap(m.AttributesMap, &diags),
			IsAccountAdmin:   types.BoolValue(m.IsAccountAdmin != nil && *m.IsAccountAdmin),
			AccountRole:      types.StringPointerValue(m.AccountRole),
			OrgID:            types.StringNull(),
			OrgRole:          types.StringNull(),
			SpaceRoles:       tuplesToMap(m.SpaceRolesMap, &diags),
			SpaceCustomRoles: tuplesToMap(m.SpaceRbacRolesMap, &diags),
			ProjectRoles:     tuplesToMap(m.ProjectRolesMap, &diags),
		}
		if m.OrgRole != nil {
			model.OrgID = types.StringValue(m.OrgRole.OrgID)
			model.OrgRole = types.StringValue(m.OrgRole.RoleID)
		}
		mappings = append(mappings, model)
	}
	list, d := types.ListValueFrom(ctx, elemType, mappings)
	diags.Append(d...)
	data.RoleMappings = list
	return diags
}

// mapToTuples converts a Terraform map to the API's [key, value] tuple list,
// sorted by key to match what Arize returns.
func mapToTuples(ctx context.Context, m types.Map, diags *diag.Diagnostics) [][]string {
	tuples := [][]string{}
	if m.IsNull() || m.IsUnknown() {
		return tuples
	}
	var values map[string]string
	diags.Append(m.ElementsAs(ctx, &values, false)...)
	for k, v := range values {
		tuples = append(tuples, []string{k, v})
	}
	sort.Slice(tuples, func(i, j int) bool { return tuples[i][0] < tuples[j][0] })
	return tuples
}

func tuplesToMap(tuples [][]string, diags *diag.Diagnostics) types.Map {
	if len(tuples) == 0 {
		return types.MapNull(types.StringType)
	}
	values := make(map[string]attr.Value, len(tuples))
	for _, t := range tuples {
		if len(t) != 2 {
			diags.AddError("Unexpected role mapping entry", fmt.Sprintf("expected a [key, value] pair, got %v", t))
			continue
		}
		values[t[0]] = types.StringValue(t[1])
	}
	m, d := types.MapValue(types.StringType, values)
	diags.Append(d...)
	return m
}

func nonEmpty(s *string) *string {
	if s == nil || *s == "" {
		return nil
	}
	return s
}
