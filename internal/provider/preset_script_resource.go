package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/DiscowZombie/terraform-provider-wellplayed/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure the resource satisfies the framework interfaces.
var (
	_ resource.Resource                = &presetScriptResource{}
	_ resource.ResourceWithConfigure   = &presetScriptResource{}
	_ resource.ResourceWithImportState = &presetScriptResource{}
)

// NewPresetScriptResource constructs the wellplayed_preset_script resource.
func NewPresetScriptResource() resource.Resource {
	return &presetScriptResource{}
}

type presetScriptResource struct {
	client *client.Client
}

// presetScriptModel maps the resource schema to Go values. All authored fields
// round-trip through Read.
type presetScriptModel struct {
	ID          types.String           `tfsdk:"id"`
	Name        types.String           `tfsdk:"name"`
	Description types.String           `tfsdk:"description"`
	Script      types.String           `tfsdk:"script"`
	Parameters  []presetParameterModel `tfsdk:"parameters"`
	CreatedAt   types.String           `tfsdk:"created_at"`
	UpdatedAt   types.String           `tfsdk:"updated_at"`
}

type presetParameterModel struct {
	Name         types.String `tfsdk:"name"`
	Type         types.String `tfsdk:"type"`
	Required     types.Bool   `tfsdk:"required"`
	DefaultValue types.String `tfsdk:"default_value"`
	Description  types.String `tfsdk:"description"`
}

func (r *presetScriptResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_preset_script"
}

func (r *presetScriptResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a WellPlayed preset script: a named Lua script that builds a tournament " +
			"step rule set, with declared parameters. Backed by the `createPresetScript`, " +
			"`updatePresetScript`, and `deletePresetScript` mutations.\n\n" +
			"Each update replaces the script and parameters in place — there is no separate " +
			"publish/release step. Applying the preset to a tournament step (and validating the " +
			"resulting rule set) is a runtime concern and is not handled by this resource.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Unique identifier of the preset script.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Preset name. Unique within the organization.",
				Required:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "Optional description of the preset.",
				Optional:            true,
			},
			"script": schema.StringAttribute{
				MarkdownDescription: "Lua source that builds a complete step rule set when the preset is applied.",
				Required:            true,
			},
			"parameters": schema.ListNestedAttribute{
				MarkdownDescription: "Parameters declared by the preset. Values are supplied at apply time.",
				Optional:            true,
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							MarkdownDescription: "Parameter name. Unique within the preset.",
							Required:            true,
						},
						"type": schema.StringAttribute{
							MarkdownDescription: "Primitive parameter type. One of `INT`, `FLOAT`, `STRING`, `BOOLEAN`.",
							Required:            true,
						},
						"required": schema.BoolAttribute{
							MarkdownDescription: "Whether a value must be supplied when applying the preset.",
							Required:            true,
						},
						"default_value": schema.StringAttribute{
							MarkdownDescription: "Default value used when the parameter is optional, serialized as a string.",
							Optional:            true,
						},
						"description": schema.StringAttribute{
							MarkdownDescription: "Optional human-readable description of the parameter.",
							Optional:            true,
						},
					},
				},
			},
			"created_at": schema.StringAttribute{
				MarkdownDescription: "When the preset was created (RFC 3339).",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"updated_at": schema.StringAttribute{
				MarkdownDescription: "When the preset was last updated (RFC 3339).",
				Computed:            true,
			},
		},
	}
}

func (r *presetScriptResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *client.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}
	r.client = c
}

func (r *presetScriptResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan presetScriptModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	input := buildPresetScriptInput(&plan)

	query := `mutation($input: CreatePresetScriptInput!) { createPresetScript(input: $input) { ` + presetScriptFields + ` } }`
	var out struct {
		CreatePresetScript gqlPresetScript `json:"createPresetScript"`
	}
	if err := r.client.Execute(ctx, query, map[string]any{"input": input}, &out); err != nil {
		resp.Diagnostics.AddError("Unable to create preset script", err.Error())
		return
	}

	applyPresetScriptToModel(&out.CreatePresetScript, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *presetScriptResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state presetScriptModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	query := `query($id: ID!) { presetScript(id: $id) { ` + presetScriptFields + ` } }`
	var out struct {
		PresetScript *gqlPresetScript `json:"presetScript"`
	}
	if err := r.client.Execute(ctx, query, map[string]any{"id": state.ID.ValueString()}, &out); err != nil {
		if isPresetScriptNotFoundErr(err) {
			tflog.Warn(ctx, "Preset script not found, removing from state", map[string]any{"id": state.ID.ValueString()})
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Unable to read preset script", err.Error())
		return
	}
	if out.PresetScript == nil {
		tflog.Warn(ctx, "Preset script not found, removing from state", map[string]any{"id": state.ID.ValueString()})
		resp.State.RemoveResource(ctx)
		return
	}

	applyPresetScriptToModel(out.PresetScript, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *presetScriptResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state presetScriptModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	input := buildPresetScriptInput(&plan)

	query := `mutation($id: ID!, $input: UpdatePresetScriptInput!) { updatePresetScript(id: $id, input: $input) { ` + presetScriptFields + ` } }`
	vars := map[string]any{"id": state.ID.ValueString(), "input": input}
	var out struct {
		UpdatePresetScript gqlPresetScript `json:"updatePresetScript"`
	}
	if err := r.client.Execute(ctx, query, vars, &out); err != nil {
		resp.Diagnostics.AddError("Unable to update preset script", err.Error())
		return
	}

	applyPresetScriptToModel(&out.UpdatePresetScript, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *presetScriptResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state presetScriptModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	query := `mutation($id: ID!) { deletePresetScript(id: $id) }`
	if err := r.client.Execute(ctx, query, map[string]any{"id": state.ID.ValueString()}, nil); err != nil {
		resp.Diagnostics.AddError("Unable to delete preset script", err.Error())
		return
	}
}

func (r *presetScriptResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// isPresetScriptNotFoundErr reports whether a GraphQL error indicates the
// preset is gone.
func isPresetScriptNotFoundErr(err error) bool {
	s := strings.ToLower(err.Error())
	return strings.Contains(s, "not found") ||
		strings.Contains(s, "does not exist") ||
		strings.Contains(s, "no preset")
}
