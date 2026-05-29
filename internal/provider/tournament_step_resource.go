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
	_ resource.Resource                = &tournamentStepResource{}
	_ resource.ResourceWithConfigure   = &tournamentStepResource{}
	_ resource.ResourceWithImportState = &tournamentStepResource{}
)

// NewTournamentStepResource constructs the wellplayed_tournament_step resource.
func NewTournamentStepResource() resource.Resource {
	return &tournamentStepResource{}
}

type tournamentStepResource struct {
	client *client.Client
}

// tournamentStepModel maps the resource schema to Go values.
type tournamentStepModel struct {
	ID           types.String        `tfsdk:"id"`
	TournamentID types.String        `tfsdk:"tournament_id"`
	Name         types.String        `tfsdk:"name"`
	Description  types.String        `tfsdk:"description"`
	Order        types.Float64       `tfsdk:"order"`
	Type         types.String        `tfsdk:"type"`
	Properties   []stepPropertyModel `tfsdk:"properties"`
	Status       types.String        `tfsdk:"status"`
	CreatedAt    types.String        `tfsdk:"created_at"`
	UpdatedAt    types.String        `tfsdk:"updated_at"`
}

type stepPropertyModel struct {
	Property types.String `tfsdk:"property"`
	Value    types.String `tfsdk:"value"`
}

func (r *tournamentStepResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_tournament_step"
}

func (r *tournamentStepResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a step (phase) within a WellPlayed tournament: a bracket or competition " +
			"stage such as a group, round-robin, or elimination bracket. Backed by the " +
			"`createTournamentStep`, `updateTournamentStep`, and `deleteTournamentStep` mutations.\n\n" +
			"This resource manages the step's core fields only. The bracket `configuration` " +
			"(groups, rounds, and games) is not managed here — it is typically built by the " +
			"`generate`/`seed` step mutations or a preset script at runtime.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Unique identifier of the tournament step.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"tournament_id": schema.StringAttribute{
				MarkdownDescription: "ID of the tournament this step belongs to. Changing this forces a new step " +
					"to be created, since a step cannot be moved between tournaments.",
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Name of the tournament step.",
				Required:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "Description of the tournament step.",
				Required:            true,
			},
			"order": schema.Float64Attribute{
				MarkdownDescription: "Display order of this step within the tournament.",
				Required:            true,
			},
			"type": schema.StringAttribute{
				MarkdownDescription: "Type of bracket or competition format. One of `SCORE`, `ROUND_ROBIN`, " +
					"`SINGLE_ELIM`, `DOUBLE_ELIM`, `CUSTOM`.",
				Required: true,
			},
			"properties": schema.ListNestedAttribute{
				MarkdownDescription: "Custom key/value properties for this step. Managed write-only: these are " +
					"sent on create/update but not returned by the API, so they are not refreshed during " +
					"Read and out-of-band changes are not detected as drift.",
				Optional: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"property": schema.StringAttribute{
							MarkdownDescription: "Key identifier of the property.",
							Required:            true,
						},
						"value": schema.StringAttribute{
							MarkdownDescription: "Value assigned to the property.",
							Required:            true,
						},
					},
				},
			},
			"status": schema.StringAttribute{
				MarkdownDescription: "Current lifecycle status of the step. One of `CONFIGURED`, `GENERATING`, " +
					"`GENERATED`, `SEEDING`, `SEEDED`, `STARTED`, `ENDED`.",
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"created_at": schema.StringAttribute{
				MarkdownDescription: "When the step was created (RFC 3339).",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"updated_at": schema.StringAttribute{
				MarkdownDescription: "When the step was last updated (RFC 3339).",
				Computed:            true,
			},
		},
	}
}

func (r *tournamentStepResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *tournamentStepResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan tournamentStepModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	step := buildTournamentStepInput(&plan)

	query := `mutation($tournamentId: ID!, $step: CreateTournamentStepInput!) { createTournamentStep(tournamentId: $tournamentId, step: $step) { ` + tournamentStepFields + ` } }`
	vars := map[string]any{"tournamentId": plan.TournamentID.ValueString(), "step": step}
	var out struct {
		CreateTournamentStep gqlTournamentStep `json:"createTournamentStep"`
	}
	if err := r.client.Execute(ctx, query, vars, &out); err != nil {
		resp.Diagnostics.AddError("Unable to create tournament step", err.Error())
		return
	}

	applyComputedToStepModel(&out.CreateTournamentStep, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *tournamentStepResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state tournamentStepModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	query := `query($id: ID!) { tournamentStep(id: $id) { ` + tournamentStepFields + ` } }`
	var out struct {
		TournamentStep *gqlTournamentStep `json:"tournamentStep"`
	}
	if err := r.client.Execute(ctx, query, map[string]any{"id": state.ID.ValueString()}, &out); err != nil {
		if isTournamentStepNotFoundErr(err) {
			tflog.Warn(ctx, "Tournament step not found, removing from state", map[string]any{"id": state.ID.ValueString()})
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Unable to read tournament step", err.Error())
		return
	}
	if out.TournamentStep == nil {
		tflog.Warn(ctx, "Tournament step not found, removing from state", map[string]any{"id": state.ID.ValueString()})
		resp.State.RemoveResource(ctx)
		return
	}

	model := modelFromStepGQL(out.TournamentStep)
	// properties is write-only managed (the API does not expose it on
	// TournamentStep), so keep whatever the prior state held.
	model.Properties = state.Properties
	resp.Diagnostics.Append(resp.State.Set(ctx, model)...)
}

func (r *tournamentStepResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state tournamentStepModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	step := buildTournamentStepInput(&plan)

	query := `mutation($stepId: ID!, $step: CreateTournamentStepInput!) { updateTournamentStep(stepId: $stepId, step: $step) { ` + tournamentStepFields + ` } }`
	vars := map[string]any{"stepId": state.ID.ValueString(), "step": step}
	var out struct {
		UpdateTournamentStep gqlTournamentStep `json:"updateTournamentStep"`
	}
	if err := r.client.Execute(ctx, query, vars, &out); err != nil {
		resp.Diagnostics.AddError("Unable to update tournament step", err.Error())
		return
	}

	applyComputedToStepModel(&out.UpdateTournamentStep, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *tournamentStepResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state tournamentStepModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	query := `mutation($stepId: ID!) { deleteTournamentStep(stepId: $stepId) }`
	if err := r.client.Execute(ctx, query, map[string]any{"stepId": state.ID.ValueString()}, nil); err != nil {
		resp.Diagnostics.AddError("Unable to delete tournament step", err.Error())
		return
	}
}

func (r *tournamentStepResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// isTournamentStepNotFoundErr reports whether a GraphQL error indicates the
// step is gone.
func isTournamentStepNotFoundErr(err error) bool {
	s := strings.ToLower(err.Error())
	return strings.Contains(s, "not found") ||
		strings.Contains(s, "does not exist") ||
		strings.Contains(s, "no step") ||
		strings.Contains(s, "no tournament step")
}
