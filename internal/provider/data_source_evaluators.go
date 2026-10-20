package provider

import (
	"context"
	"fmt"

	"github.com/arize-ai/terraform-provider-arize/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = (*EvaluatorsDataSource)(nil)

func NewEvaluatorsDataSource() datasource.DataSource {
	return &EvaluatorsDataSource{}
}

type EvaluatorsDataSource struct {
	client *client.Client
}

type EvaluatorsDataSourceModel struct {
	SpaceID    types.String               `tfsdk:"space_id"`
	Evaluators []EvaluatorDataSourceModel `tfsdk:"evaluators"`
}

type EvaluatorDataSourceModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	TaskType    types.String `tfsdk:"task_type"`
	CreatedAt   types.String `tfsdk:"created_at"`
	UpdatedAt   types.String `tfsdk:"updated_at"`
}

func (d *EvaluatorsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_evaluators"
}

func (d *EvaluatorsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Lists the evaluators in an Arize space, including ones created in the Arize UI.",
		Attributes: map[string]schema.Attribute{
			"space_id": schema.StringAttribute{
				MarkdownDescription: "ID of the space to list evaluators from.",
				Required:            true,
			},
			"evaluators": schema.ListNestedAttribute{
				MarkdownDescription: "Evaluators in the space.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id":          schema.StringAttribute{Computed: true, MarkdownDescription: "Evaluator ID."},
						"name":        schema.StringAttribute{Computed: true, MarkdownDescription: "Evaluator name."},
						"description": schema.StringAttribute{Computed: true, MarkdownDescription: "Evaluator description."},
						"task_type":   schema.StringAttribute{Computed: true, MarkdownDescription: "Task type, e.g. `template_evaluation` or `code_evaluation`."},
						"created_at":  schema.StringAttribute{Computed: true, MarkdownDescription: "Creation timestamp."},
						"updated_at":  schema.StringAttribute{Computed: true, MarkdownDescription: "Last update timestamp."},
					},
				},
			},
		},
	}
}

func (d *EvaluatorsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	config, ok := req.ProviderData.(*ArizeProviderModel)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected DataSource Configure Type",
			fmt.Sprintf("Expected *ArizeProviderModel, got: %T", req.ProviderData),
		)
		return
	}
	d.client = client.New(config.Endpoint.ValueString(), config.APIKey.ValueString())
}

func (d *EvaluatorsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data EvaluatorsDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	evaluators, err := d.client.ListEvaluators(ctx, data.SpaceID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to list evaluators", err.Error())
		return
	}

	data.Evaluators = make([]EvaluatorDataSourceModel, 0, len(evaluators))
	for _, e := range evaluators {
		item := EvaluatorDataSourceModel{
			ID:          types.StringValue(e.ID),
			Name:        types.StringValue(e.Name),
			Description: types.StringNull(),
			TaskType:    types.StringValue(e.TaskType),
			CreatedAt:   types.StringValue(e.CreatedAt),
			UpdatedAt:   types.StringValue(e.UpdatedAt),
		}
		if e.Description != nil {
			item.Description = types.StringValue(*e.Description)
		}
		data.Evaluators = append(data.Evaluators, item)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
