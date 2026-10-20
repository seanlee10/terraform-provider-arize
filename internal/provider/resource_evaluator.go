package provider

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/arize-ai/terraform-provider-arize/internal/client"
	"github.com/hashicorp/terraform-plugin-framework-validators/resourcevalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                     = (*EvaluatorResource)(nil)
	_ resource.ResourceWithImportState      = (*EvaluatorResource)(nil)
	_ resource.ResourceWithConfigValidators = (*EvaluatorResource)(nil)
	_ resource.ResourceWithModifyPlan       = (*EvaluatorResource)(nil)
)

const (
	defaultCreateCommitMessage = "Created by Terraform"
	defaultUpdateCommitMessage = "Updated by Terraform"
)

func NewEvaluatorResource() resource.Resource {
	return &EvaluatorResource{}
}

type EvaluatorResource struct {
	client *client.Client
}

type EvaluatorResourceModel struct {
	ID              types.String            `tfsdk:"id"`
	SpaceID         types.String            `tfsdk:"space_id"`
	Name            types.String            `tfsdk:"name"`
	Description     types.String            `tfsdk:"description"`
	CommitMessage   types.String            `tfsdk:"commit_message"`
	DataGranularity types.String            `tfsdk:"data_granularity"`
	TaskType        types.String            `tfsdk:"task_type"`
	VersionNumber   types.Int64             `tfsdk:"version_number"`
	Template        *TemplateEvaluatorModel `tfsdk:"template"`
	Code            *CodeEvaluatorModel     `tfsdk:"code"`
}

type TemplateEvaluatorModel struct {
	ColumnName            types.String       `tfsdk:"column_name"`
	Template              types.String       `tfsdk:"template"`
	ClassificationChoices types.Map          `tfsdk:"classification_choices"`
	Direction             types.String       `tfsdk:"direction"`
	IncludeExplanations   types.Bool         `tfsdk:"include_explanations"`
	UseFunctionCalling    types.Bool         `tfsdk:"use_function_calling"`
	UseStructuredOutput   types.Bool         `tfsdk:"use_structured_output"`
	LLM                   *EvaluatorLLMModel `tfsdk:"llm"`
}

type EvaluatorLLMModel struct {
	IntegrationID        types.String `tfsdk:"integration_id"`
	ModelName            types.String `tfsdk:"model_name"`
	InvocationParameters types.String `tfsdk:"invocation_parameters"`
	ProviderParameters   types.String `tfsdk:"provider_parameters"`
}

type CodeEvaluatorModel struct {
	ColumnName     types.String `tfsdk:"column_name"`
	ClassName      types.String `tfsdk:"class_name"`
	Code           types.String `tfsdk:"code"`
	Imports        types.String `tfsdk:"imports"`
	SpanAttributes types.List   `tfsdk:"span_attributes"`
}

func (r *EvaluatorResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_evaluator"
}

// Switching between template and code evaluators cannot be done with a new version.
var replaceOnEvaluatorKindChange = objectplanmodifier.RequiresReplaceIf(
	func(ctx context.Context, req planmodifier.ObjectRequest, resp *objectplanmodifier.RequiresReplaceIfFuncResponse) {
		resp.RequiresReplace = req.StateValue.IsNull() != req.PlanValue.IsNull()
	},
	"Switching between template and code evaluators requires replacement.",
	"Switching between `template` and `code` evaluators requires replacement.",
)

func (r *EvaluatorResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages an Arize evaluator: an LLM-as-judge (`template`) or Python (`code`) evaluator in a space. " +
			"Arize versions evaluators: changing `template` or `code` publishes a new evaluator version using `commit_message`, " +
			"while `name` and `description` are edited in place.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The evaluator ID.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"space_id": schema.StringAttribute{
				MarkdownDescription: "ID of the space that owns the evaluator. Changing it forces a new evaluator.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Evaluator name. Must be unique within the space.",
				Required:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "Evaluator description. The Arize API cannot clear a description, so removing it from configuration leaves the current description in place.",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"commit_message": schema.StringAttribute{
				MarkdownDescription: "Commit message for the evaluator version published by this apply. " +
					"Only used when the evaluator is created or its `template`/`code` changes. " +
					"Defaults to `\"" + defaultCreateCommitMessage + "\"` / `\"" + defaultUpdateCommitMessage + "\"`.",
				Optional: true,
				Computed: true,
			},
			"data_granularity": schema.StringAttribute{
				MarkdownDescription: "What the evaluator scores: `span`, `trace`, or `session`. Arize does not allow changing it, so changing it forces a new evaluator. Defaults to `span`.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("span"),
				Validators:          []validator.String{stringvalidator.OneOf("span", "trace", "session")},
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"task_type": schema.StringAttribute{
				MarkdownDescription: "Arize task type: `template_evaluation` or `code_evaluation`.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"version_number": schema.Int64Attribute{
				MarkdownDescription: "Number of the evaluator's current version.",
				Computed:            true,
			},
			"template": schema.SingleNestedAttribute{
				MarkdownDescription: "LLM-as-judge evaluator configuration. Exactly one of `template` or `code` must be set.",
				Optional:            true,
				PlanModifiers:       []planmodifier.Object{replaceOnEvaluatorKindChange},
				Attributes: map[string]schema.Attribute{
					"column_name": schema.StringAttribute{
						MarkdownDescription: "Name of the result column the evaluator writes (e.g. `correctness`). Must be unique within the space.",
						Required:            true,
					},
					"template": schema.StringAttribute{
						MarkdownDescription: "Prompt template sent to the judge LLM. Reference span data with `{variable}` placeholders such as `{input}` and `{output}`.",
						Required:            true,
					},
					"classification_choices": schema.MapAttribute{
						MarkdownDescription: "Map of output label to score, e.g. `{ correct = 1, incorrect = 0 }`.",
						ElementType:         types.Float64Type,
						Optional:            true,
					},
					"direction": schema.StringAttribute{
						MarkdownDescription: "Whether higher scores are better: `maximize`, `minimize`, or `none`.",
						Optional:            true,
						Computed:            true,
						Validators:          []validator.String{stringvalidator.OneOf("maximize", "minimize", "none")},
						PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
					},
					"include_explanations": schema.BoolAttribute{
						MarkdownDescription: "Ask the judge to explain its label. Defaults to `true`.",
						Optional:            true,
						Computed:            true,
						Default:             booldefault.StaticBool(true),
					},
					"use_function_calling": schema.BoolAttribute{
						MarkdownDescription: "Use function calling when the model supports it. Defaults to `true`.",
						Optional:            true,
						Computed:            true,
						Default:             booldefault.StaticBool(true),
					},
					"use_structured_output": schema.BoolAttribute{
						MarkdownDescription: "Use structured output when the model supports it. Defaults to `true`.",
						Optional:            true,
						Computed:            true,
						Default:             booldefault.StaticBool(true),
					},
					"llm": schema.SingleNestedAttribute{
						MarkdownDescription: "Judge model. When omitted, Arize uses the space's default LLM integration.",
						Optional:            true,
						Attributes: map[string]schema.Attribute{
							"integration_id": schema.StringAttribute{
								MarkdownDescription: "ID of an LLM integration available to the space.",
								Required:            true,
							},
							"model_name": schema.StringAttribute{
								MarkdownDescription: "Model name, e.g. `gpt-4o` or `claude-sonnet-5-5`.",
								Required:            true,
							},
							"invocation_parameters": schema.StringAttribute{
								MarkdownDescription: "JSON object of model invocation parameters, e.g. `jsonencode({ max_tokens = 256 })`. " +
									"Arize normalizes these per provider, so changes made outside Terraform are not detected.",
								Optional: true,
							},
							"provider_parameters": schema.StringAttribute{
								MarkdownDescription: "JSON object of provider-specific parameters. Changes made outside Terraform are not detected.",
								Optional:            true,
							},
						},
					},
				},
			},
			"code": schema.SingleNestedAttribute{
				MarkdownDescription: "Python code evaluator configuration. Exactly one of `template` or `code` must be set.",
				Optional:            true,
				PlanModifiers:       []planmodifier.Object{replaceOnEvaluatorKindChange},
				Attributes: map[string]schema.Attribute{
					"column_name": schema.StringAttribute{
						MarkdownDescription: "Name of the result column the evaluator writes. Must be unique within the space.",
						Required:            true,
					},
					"class_name": schema.StringAttribute{
						MarkdownDescription: "Name of the `CodeEvaluator` subclass defined in `code`.",
						Required:            true,
					},
					"code": schema.StringAttribute{
						MarkdownDescription: "Python source defining the evaluator class. Arize validates it on apply.",
						Required:            true,
					},
					"imports": schema.StringAttribute{
						MarkdownDescription: "Python import statements the evaluator code needs.",
						Optional:            true,
					},
					"span_attributes": schema.ListAttribute{
						MarkdownDescription: "Span attributes the evaluator reads, derived by Arize from the `evaluate()` signature.",
						ElementType:         types.StringType,
						Computed:            true,
					},
				},
			},
		},
	}
}

func (r *EvaluatorResource) ConfigValidators(ctx context.Context) []resource.ConfigValidator {
	return []resource.ConfigValidator{
		resourcevalidator.ExactlyOneOf(path.MatchRoot("template"), path.MatchRoot("code")),
	}
}

func (r *EvaluatorResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.State.Raw.IsNull() || req.Plan.Raw.IsNull() {
		return
	}
	var configured, prior types.String
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("description"), &configured)...)
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("description"), &prior)...)
	if configured.IsNull() && !prior.IsNull() && prior.ValueString() != "" {
		resp.Diagnostics.AddAttributeWarning(path.Root("description"),
			"Evaluator description will not be removed",
			"The Arize API cannot clear an evaluator description, so the existing description is kept. Set a new description to change it.")
	}
}

func (r *EvaluatorResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *EvaluatorResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan EvaluatorResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	config, diags := buildEvaluatorConfig(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if plan.CommitMessage.IsUnknown() || plan.CommitMessage.IsNull() {
		plan.CommitMessage = types.StringValue(defaultCreateCommitMessage)
	}

	id, err := r.client.CreateEvaluator(ctx, plan.SpaceID.ValueString(), plan.Name.ValueString(),
		plan.Description.ValueString(), plan.CommitMessage.ValueString(), config)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create evaluator", err.Error())
		return
	}
	plan.ID = types.StringValue(id)
	// Persist the ID before reading back so a failed read doesn't orphan the evaluator.
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), id)...)

	evaluator, err := r.client.GetEvaluator(ctx, plan.SpaceID.ValueString(), id)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read evaluator after create", err.Error())
		return
	}
	resp.Diagnostics.Append(applyEvaluator(ctx, &plan, evaluator, false)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *EvaluatorResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state EvaluatorResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	evaluator, err := r.client.GetEvaluator(ctx, state.SpaceID.ValueString(), state.ID.ValueString())
	if errors.Is(err, client.ErrEvaluatorNotFound) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Failed to read evaluator", err.Error())
		return
	}

	resp.Diagnostics.Append(applyEvaluator(ctx, &state, evaluator, true)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *EvaluatorResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state EvaluatorResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	planConfig, diags := buildEvaluatorConfig(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	stateConfig, diags := buildEvaluatorConfig(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	id := state.ID.ValueString()

	if !plan.Name.Equal(state.Name) || !plan.Description.Equal(state.Description) {
		if err := r.client.EditEvaluator(ctx, id, plan.Name.ValueString(), plan.Description.ValueString()); err != nil {
			resp.Diagnostics.AddError("Failed to update evaluator", err.Error())
			return
		}
	}

	configChanged, err := evaluatorConfigsDiffer(planConfig, stateConfig)
	if err != nil {
		resp.Diagnostics.AddError("Failed to compare evaluator configuration", err.Error())
		return
	}
	if plan.CommitMessage.IsUnknown() || plan.CommitMessage.IsNull() {
		if configChanged {
			plan.CommitMessage = types.StringValue(defaultUpdateCommitMessage)
		} else {
			plan.CommitMessage = state.CommitMessage
		}
	}
	if configChanged {
		if err := r.client.CreateEvaluatorVersion(ctx, id, plan.CommitMessage.ValueString(), planConfig); err != nil {
			resp.Diagnostics.AddError("Failed to publish evaluator version", err.Error())
			return
		}
	}

	evaluator, err := r.client.GetEvaluator(ctx, plan.SpaceID.ValueString(), id)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read evaluator after update", err.Error())
		return
	}
	resp.Diagnostics.Append(applyEvaluator(ctx, &plan, evaluator, false)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *EvaluatorResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state EvaluatorResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DeleteEvaluator(ctx, state.ID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Failed to delete evaluator", err.Error())
	}
}

func (r *EvaluatorResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	spaceID, evaluatorID, ok := strings.Cut(req.ID, "/")
	if !ok || spaceID == "" || evaluatorID == "" {
		resp.Diagnostics.AddError(
			"Invalid import ID",
			fmt.Sprintf("Expected \"<space_id>/<evaluator_id>\", got %q.", req.ID),
		)
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("space_id"), spaceID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), evaluatorID)...)
}

func buildEvaluatorConfig(ctx context.Context, m *EvaluatorResourceModel) (client.EvaluatorConfigInput, diag.Diagnostics) {
	var diags diag.Diagnostics
	var config client.EvaluatorConfigInput
	granularity := m.DataGranularity.ValueString()

	if t := m.Template; t != nil {
		input := &client.TemplateEvaluatorInput{
			Name:                           t.ColumnName.ValueString(),
			Template:                       t.Template.ValueString(),
			Position:                       0,
			Direction:                      t.Direction.ValueString(),
			IncludeExplanations:            t.IncludeExplanations.ValueBool(),
			UseFunctionCallingIfAvailable:  t.UseFunctionCalling.ValueBool(),
			UseStructuredOutputIfAvailable: t.UseStructuredOutput.ValueBool(),
			DataGranularityType:            granularity,
		}
		if !t.ClassificationChoices.IsNull() && !t.ClassificationChoices.IsUnknown() {
			diags.Append(t.ClassificationChoices.ElementsAs(ctx, &input.ClassificationChoices, false)...)
		}
		if t.LLM != nil {
			input.LlmConfig = &client.LlmConfigInput{
				IntegrationID:        t.LLM.IntegrationID.ValueString(),
				ModelName:            t.LLM.ModelName.ValueString(),
				InvocationParameters: parseJSONObject(t.LLM.InvocationParameters, "template.llm.invocation_parameters", &diags),
				ProviderParameters:   parseJSONObject(t.LLM.ProviderParameters, "template.llm.provider_parameters", &diags),
			}
		}
		config.Template = input
	}

	if c := m.Code; c != nil {
		config.Code = &client.CodeEvaluatorInput{
			Name:                c.ColumnName.ValueString(),
			EvaluationClass:     c.ClassName.ValueString(),
			EvalClassCodeBlock:  c.Code.ValueString(),
			PackageImports:      c.Imports.ValueString(),
			Position:            0,
			DataGranularityType: granularity,
		}
	}

	return config, diags
}

func parseJSONObject(v types.String, attr string, diags *diag.Diagnostics) map[string]interface{} {
	out := map[string]interface{}{}
	if v.IsNull() || v.IsUnknown() || v.ValueString() == "" {
		return out
	}
	if err := json.Unmarshal([]byte(v.ValueString()), &out); err != nil {
		diags.AddAttributeError(path.Root(attr), "Invalid JSON object", fmt.Sprintf("%s must be a JSON object: %s", attr, err))
	}
	return out
}

func evaluatorConfigsDiffer(a, b client.EvaluatorConfigInput) (bool, error) {
	aj, err := json.Marshal([]interface{}{a.Template, a.Code})
	if err != nil {
		return false, err
	}
	bj, err := json.Marshal([]interface{}{b.Template, b.Code})
	if err != nil {
		return false, err
	}
	return string(aj) != string(bj), nil
}

// applyEvaluator copies API values into the model. Computed fields are always set;
// user inputs are only overwritten when refreshing (Read), so Create/Update keep planned values.
func applyEvaluator(ctx context.Context, m *EvaluatorResourceModel, e *client.Evaluator, refresh bool) diag.Diagnostics {
	var diags diag.Diagnostics

	m.ID = types.StringValue(e.ID)
	m.TaskType = types.StringValue(e.TaskType)
	m.VersionNumber = types.Int64Null()
	if m.CommitMessage.IsNull() || m.CommitMessage.IsUnknown() {
		m.CommitMessage = types.StringValue(e.CommitMessage)
	}
	if refresh {
		m.Name = types.StringValue(e.Name)
	}
	if refresh || m.Description.IsUnknown() {
		m.Description = types.StringNull()
		if e.Description != nil && *e.Description != "" {
			m.Description = types.StringValue(*e.Description)
		}
	}

	cfg := e.Config
	if cfg == nil {
		diags.AddError("Evaluator has no configuration", fmt.Sprintf("Arize returned no configuration for evaluator %s.", e.ID))
		return diags
	}
	if cfg.EvaluatorVersion != nil {
		m.VersionNumber = types.Int64Value(cfg.EvaluatorVersion.VersionNumber)
	}
	if refresh || m.DataGranularity.IsUnknown() {
		m.DataGranularity = types.StringValue(cfg.DataGranularityType)
	}

	switch cfg.Typename {
	case client.EvaluatorConfigTemplate:
		prior := m.Template
		if prior == nil || refresh {
			t := &TemplateEvaluatorModel{
				ColumnName:          types.StringValue(cfg.Name),
				Template:            types.StringValue(cfg.Template),
				IncludeExplanations: types.BoolValue(cfg.IncludeExplanations),
				UseFunctionCalling:  types.BoolValue(cfg.UseFunctionCallingIfAvailable),
				UseStructuredOutput: types.BoolValue(cfg.UseStructuredOutputIfAvailable),
				Direction:           types.StringValue(cfg.Direction),
			}
			t.ClassificationChoices = types.MapNull(types.Float64Type)
			if len(cfg.ClassificationChoices) > 0 {
				choices, d := types.MapValueFrom(ctx, types.Float64Type, cfg.ClassificationChoices)
				diags.Append(d...)
				t.ClassificationChoices = choices
			}
			if cfg.LlmConfig != nil {
				t.LLM = &EvaluatorLLMModel{
					IntegrationID:        types.StringValue(cfg.LlmConfig.IntegrationID),
					ModelName:            types.StringValue(cfg.LlmConfig.ModelName),
					InvocationParameters: types.StringNull(),
					ProviderParameters:   types.StringNull(),
				}
				// Arize rewrites these per provider; keep what the user configured.
				if prior != nil && prior.LLM != nil {
					t.LLM.InvocationParameters = prior.LLM.InvocationParameters
					t.LLM.ProviderParameters = prior.LLM.ProviderParameters
				}
			}
			m.Template = t
		} else if prior.Direction.IsUnknown() {
			prior.Direction = types.StringValue(cfg.Direction)
		}
		m.Code = nil

	case client.EvaluatorConfigCode:
		spanAttrs, d := types.ListValueFrom(ctx, types.StringType, cfg.SpanAttributes)
		diags.Append(d...)
		if m.Code == nil || refresh {
			m.Code = &CodeEvaluatorModel{
				ColumnName: types.StringValue(cfg.Name),
				ClassName:  types.StringValue(cfg.EvaluationClass),
				Code:       types.StringValue(cfg.EvalClassCodeBlock),
				Imports:    types.StringNull(),
			}
			if cfg.PackageImports != "" {
				m.Code.Imports = types.StringValue(cfg.PackageImports)
			}
		}
		m.Code.SpanAttributes = spanAttrs
		m.Template = nil

	default:
		diags.AddError(
			"Unsupported evaluator type",
			fmt.Sprintf("Evaluator %s has configuration type %q, which this provider does not manage yet.", e.ID, cfg.Typename),
		)
	}

	return diags
}
