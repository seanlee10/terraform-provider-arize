package provider

import (
	"context"
	"errors"
	"fmt"
	"sort"

	"github.com/arize-ai/terraform-provider-arize/internal/client"
	"github.com/hashicorp/terraform-plugin-framework-validators/mapvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = (*DatasetResource)(nil)
var _ resource.ResourceWithImportState = (*DatasetResource)(nil)
var _ resource.ResourceWithValidateConfig = (*DatasetResource)(nil)

func NewDatasetResource() resource.Resource {
	return &DatasetResource{}
}

type DatasetResource struct {
	client *client.RESTClient
}

type DatasetResourceModel struct {
	ID        types.String `tfsdk:"id"`
	SpaceID   types.String `tfsdk:"space_id"`
	Name      types.String `tfsdk:"name"`
	KeyColumn types.String `tfsdk:"key_column"`
	Examples  types.Map    `tfsdk:"examples"`
	CreatedAt types.String `tfsdk:"created_at"`
}

var exampleType = types.MapType{ElemType: types.StringType}

func (r *DatasetResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dataset"
}

func (r *DatasetResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages an Arize dataset and its examples. Rows are matched to existing examples by `key_column`, " +
			"so each apply updates changed rows, adds new ones, and deletes missing ones in the dataset's latest version, in place. " +
			"Import with `<dataset_id>:<key_column>`.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The dataset ID.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"space_id": schema.StringAttribute{
				MarkdownDescription: "ID of the space the dataset belongs to. Changing it replaces the dataset.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Dataset name, unique within the space.",
				Required:            true,
				Validators:          []validator.String{stringvalidator.LengthAtLeast(1)},
			},
			"key_column": schema.StringAttribute{
				MarkdownDescription: "Column whose value identifies a row, e.g. `question`. Every row needs a unique, non-empty value. " +
					"Changing a row's key counts as deleting that example and adding a new one.",
				Required:   true,
				Validators: []validator.String{stringvalidator.LengthAtLeast(1)},
			},
			"examples": schema.MapAttribute{
				ElementType: exampleType,
				MarkdownDescription: "Rows of the dataset keyed by their `key_column` value, each a map of column to string value " +
					"that includes the key column. From a CSV: `{ for row in csvdecode(file(\"data.csv\")) : row.question => row }`. " +
					"The columns `id`, `created_at`, `updated_at`, and `annotations` are reserved.",
				Required:   true,
				Validators: []validator.Map{mapvalidator.SizeAtLeast(1)},
			},
			"created_at": schema.StringAttribute{
				MarkdownDescription: "Timestamp when the dataset was created.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
		},
	}
}

func (r *DatasetResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
	r.client = client.NewREST(config.RESTEndpoint.ValueString(), config.APIKey.ValueString())
}

func (r *DatasetResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var data DatasetResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() || data.KeyColumn.IsUnknown() || data.Examples.IsUnknown() || data.Examples.IsNull() {
		return
	}
	for _, el := range data.Examples.Elements() { // rows may be unknown until apply
		if el.IsUnknown() {
			return
		}
	}
	rows, diags := exampleRows(ctx, data.Examples)
	resp.Diagnostics.Append(diags...)
	if diags.HasError() {
		return
	}
	if err := validateExampleRows(data.KeyColumn.ValueString(), rows); err != nil {
		resp.Diagnostics.AddAttributeError(path.Root("examples"), "Invalid dataset examples", err.Error())
	}
}

func (r *DatasetResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data DatasetResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	rows, diags := exampleRows(ctx, data.Examples)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// The create request takes at most one batch; the rest are appended.
	ordered := sortedRows(rows)
	first, rest := ordered, []map[string]string(nil)
	if len(ordered) > 1000 {
		first, rest = ordered[:1000], ordered[1000:]
	}
	ds, err := r.client.CreateDataset(ctx, data.SpaceID.ValueString(), data.Name.ValueString(), first)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create dataset", err.Error())
		return
	}
	// Save the ID now so a failure below leaves the dataset tracked, not orphaned.
	data.ID = types.StringValue(ds.ID)
	data.CreatedAt = types.StringValue(ds.CreatedAt)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)

	if len(rest) > 0 {
		if err := r.client.InsertDatasetExamples(ctx, ds.ID, rest); err != nil {
			resp.Diagnostics.AddError("Failed to add dataset examples", err.Error())
			return
		}
	}
}

func (r *DatasetResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data DatasetResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ds, err := r.client.GetDataset(ctx, data.ID.ValueString())
	if errors.Is(err, client.ErrDatasetNotFound) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Failed to read dataset", err.Error())
		return
	}
	fetched, err := r.client.ListDatasetExamples(ctx, ds.ID)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read dataset examples", err.Error())
		return
	}

	byKey, err := examplesByKey(data.KeyColumn.ValueString(), fetched)
	if err != nil {
		resp.Diagnostics.AddError("Dataset examples can't be keyed", err.Error())
		return
	}
	examples, diags := types.MapValueFrom(ctx, exampleType, byKey)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.ID = types.StringValue(ds.ID)
	data.SpaceID = types.StringValue(ds.SpaceID)
	data.Name = types.StringValue(ds.Name)
	data.CreatedAt = types.StringValue(ds.CreatedAt)
	data.Examples = examples
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *DatasetResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state DatasetResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	id := state.ID.ValueString()

	if !plan.Name.Equal(state.Name) {
		if err := r.client.RenameDataset(ctx, id, plan.Name.ValueString()); err != nil {
			resp.Diagnostics.AddError("Failed to rename dataset", err.Error())
			return
		}
	}

	rows, diags := exampleRows(ctx, plan.Examples)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	existing, err := r.client.ListDatasetExamples(ctx, id)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read dataset examples", err.Error())
		return
	}
	d := diffExamples(plan.KeyColumn.ValueString(), existing, rows)

	if len(d.Update) > 0 {
		if err := r.client.UpdateDatasetExamples(ctx, id, d.Update); err != nil {
			resp.Diagnostics.AddError("Failed to update dataset examples", err.Error())
			return
		}
	}
	if len(d.Insert) > 0 {
		if err := r.client.InsertDatasetExamples(ctx, id, d.Insert); err != nil {
			resp.Diagnostics.AddError("Failed to add dataset examples", err.Error())
			return
		}
	}
	if len(d.Delete) > 0 {
		ds, err := r.client.GetDataset(ctx, id)
		if err != nil {
			resp.Diagnostics.AddError("Failed to read dataset", err.Error())
			return
		}
		if err := r.client.DeleteDatasetExamples(ctx, id, ds.LatestVersionID(), d.Delete); err != nil {
			resp.Diagnostics.AddError("Failed to delete dataset examples", err.Error())
			return
		}
	}

	plan.ID = state.ID
	plan.CreatedAt = state.CreatedAt
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *DatasetResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data DatasetResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DeleteDataset(ctx, data.ID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Failed to delete dataset", err.Error())
	}
}

// ImportState takes "<dataset_id>:<key_column>": Read needs the key column to
// key the examples.
func (r *DatasetResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	datasetID, keyColumn, err := parseDatasetImportID(req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Invalid import ID", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), datasetID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("key_column"), keyColumn)...)
}

func exampleRows(ctx context.Context, m types.Map) (map[string]map[string]string, diag.Diagnostics) {
	if m.IsNull() || m.IsUnknown() {
		return nil, nil
	}
	var rows map[string]map[string]string
	diags := m.ElementsAs(ctx, &rows, false)
	return rows, diags
}

// sortedRows returns the rows ordered by key, so examples are created in a
// stable order.
func sortedRows(rows map[string]map[string]string) []map[string]string {
	keys := make([]string, 0, len(rows))
	for k := range rows {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := make([]map[string]string, len(keys))
	for i, k := range keys {
		out[i] = rows[k]
	}
	return out
}
