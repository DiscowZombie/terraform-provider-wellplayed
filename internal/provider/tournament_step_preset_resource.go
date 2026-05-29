package provider

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/DiscowZombie/terraform-provider-wellplayed/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure the resource satisfies the framework interfaces.
var (
	_ resource.Resource                = &tournamentStepPresetResource{}
	_ resource.ResourceWithConfigure   = &tournamentStepPresetResource{}
	_ resource.ResourceWithImportState = &tournamentStepPresetResource{}
)

// validationPollInterval is how often the resource polls a running validation
// job. The overall wait is bounded by the operation's context deadline.
const validationPollInterval = 2 * time.Second

// NewTournamentStepPresetResource constructs the
// wellplayed_tournament_step_preset resource.
func NewTournamentStepPresetResource() resource.Resource {
	return &tournamentStepPresetResource{}
}

type tournamentStepPresetResource struct {
	client *client.Client
}

// tournamentStepPresetModel maps the resource schema to Go values. The authored
// inputs (tournament_step_id, preset_script_id, parameters, validate) round-trip
// from configuration; the remaining fields describe the resulting rule set and
// are computed from the server.
type tournamentStepPresetModel struct {
	ID               types.String `tfsdk:"id"`
	TournamentStepID types.String `tfsdk:"tournament_step_id"`
	PresetScriptID   types.String `tfsdk:"preset_script_id"`
	Parameters       types.String `tfsdk:"parameters"`
	Validate         types.Bool   `tfsdk:"validate"`
	Version          types.Int64  `tfsdk:"version"`
	PresetName       types.String `tfsdk:"preset_name"`
	TeamCount        types.Int64  `tfsdk:"team_count"`
	Validated        types.Bool   `tfsdk:"validated"`
	ValidatedAt      types.String `tfsdk:"validated_at"`
	CreatedAt        types.String `tfsdk:"created_at"`
	UpdatedAt        types.String `tfsdk:"updated_at"`
}

func (r *tournamentStepPresetResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_tournament_step_preset"
}

func (r *tournamentStepPresetResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Applies a preset script to a tournament step, producing the step's rule set, and " +
			"optionally validates the result. Backed by the `applyPresetScript` mutation, the " +
			"`stepRuleSet` query, and the `validateStepRuleSet` validation job.\n\n" +
			"The tournament step must already exist — create it with `wellplayed_tournament_step` and " +
			"reference its `id`. Each apply re-runs the preset and replaces the step's rule set in " +
			"place, bumping its `version`. When `validate` is `true`, the apply blocks until the " +
			"WellPlayed validation job finishes and fails if the resulting rule set is invalid.\n\n" +
			"There is no API to remove a rule set from a step, so destroying this resource only drops it " +
			"from Terraform state — the last-applied rule set remains on the step until another preset " +
			"is applied.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Identifier of the step rule set produced by applying the preset.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"tournament_step_id": schema.StringAttribute{
				MarkdownDescription: "ID of the tournament step the preset is applied to (see " +
					"`wellplayed_tournament_step`). Immutable; changing it applies the preset to a different " +
					"step and forces replacement.",
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"preset_script_id": schema.StringAttribute{
				MarkdownDescription: "ID of the preset script to apply (see `wellplayed_preset_script`). " +
					"Changing it re-applies the new preset to the step.",
				Required: true,
			},
			"parameters": schema.StringAttribute{
				MarkdownDescription: "Parameter values passed to the preset script, encoded as a JSON object " +
					"string (e.g. `jsonencode({ team_count = 8 })`). Must satisfy the parameters declared by " +
					"the preset script.",
				Optional: true,
			},
			"validate": schema.BoolAttribute{
				MarkdownDescription: "Whether to validate the resulting rule set after applying the preset. " +
					"When `true`, the apply blocks until the validation job completes and fails the apply if " +
					"the rule set is invalid. Defaults to `false`.",
				Optional: true,
				Computed: true,
				Default:  booldefault.StaticBool(false),
			},
			"version": schema.Int64Attribute{
				MarkdownDescription: "Version of the step rule set. Incremented each time the preset is applied.",
				Computed:            true,
			},
			"preset_name": schema.StringAttribute{
				MarkdownDescription: "Name of the preset that produced the rule set, as recorded by the server.",
				Computed:            true,
			},
			"team_count": schema.Int64Attribute{
				MarkdownDescription: "Number of teams the rule set was built for, if determined by the preset.",
				Computed:            true,
			},
			"validated": schema.BoolAttribute{
				MarkdownDescription: "Whether the current rule set has passed validation.",
				Computed:            true,
			},
			"validated_at": schema.StringAttribute{
				MarkdownDescription: "When the rule set was last validated (RFC 3339), if ever.",
				Computed:            true,
			},
			"created_at": schema.StringAttribute{
				MarkdownDescription: "When the rule set was created (RFC 3339).",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"updated_at": schema.StringAttribute{
				MarkdownDescription: "When the rule set was last updated (RFC 3339).",
				Computed:            true,
			},
		},
	}
}

func (r *tournamentStepPresetResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *tournamentStepPresetResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan tournamentStepPresetModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ruleSet, err := r.applyPreset(ctx, &plan)
	if err != nil {
		resp.Diagnostics.AddError("Unable to apply preset to tournament step", err.Error())
		return
	}
	applyStepRuleSetToModel(ruleSet, &plan)

	if plan.Validate.ValueBool() {
		if refreshed := r.validateRuleSet(ctx, plan.TournamentStepID.ValueString(), ruleSet.ID, &resp.Diagnostics); refreshed != nil {
			applyStepRuleSetToModel(refreshed, &plan)
		}
		if resp.Diagnostics.HasError() {
			return
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *tournamentStepPresetResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state tournamentStepPresetModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ruleSet, err := r.readRuleSet(ctx, state.TournamentStepID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Unable to read step rule set", err.Error())
		return
	}
	if ruleSet == nil {
		tflog.Warn(ctx, "Step rule set not found, removing from state", map[string]any{
			"tournament_step_id": state.TournamentStepID.ValueString(),
		})
		resp.State.RemoveResource(ctx)
		return
	}

	applyStepRuleSetToModel(ruleSet, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *tournamentStepPresetResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state tournamentStepPresetModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Re-apply the preset only when the preset or its parameters changed. A bare
	// validate toggle re-validates the existing rule set without re-applying.
	presetChanged := !plan.PresetScriptID.Equal(state.PresetScriptID) || !plan.Parameters.Equal(state.Parameters)

	var ruleSet *gqlStepRuleSet
	var err error
	if presetChanged {
		ruleSet, err = r.applyPreset(ctx, &plan)
		if err != nil {
			resp.Diagnostics.AddError("Unable to apply preset to tournament step", err.Error())
			return
		}
	} else {
		ruleSet, err = r.readRuleSet(ctx, plan.TournamentStepID.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Unable to read step rule set", err.Error())
			return
		}
		if ruleSet == nil {
			resp.Diagnostics.AddError(
				"Step rule set not found",
				"The rule set for tournament step "+plan.TournamentStepID.ValueString()+" no longer exists. Re-apply the preset.",
			)
			return
		}
	}
	applyStepRuleSetToModel(ruleSet, &plan)

	if plan.Validate.ValueBool() {
		if refreshed := r.validateRuleSet(ctx, plan.TournamentStepID.ValueString(), ruleSet.ID, &resp.Diagnostics); refreshed != nil {
			applyStepRuleSetToModel(refreshed, &plan)
		}
		if resp.Diagnostics.HasError() {
			return
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *tournamentStepPresetResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// The schema exposes no mutation to remove a rule set from a step, so there
	// is nothing to call: dropping the resource from state leaves the
	// last-applied rule set on the step.
	tflog.Warn(ctx, "wellplayed_tournament_step_preset has no server-side delete; the applied rule set remains on the step", nil)
}

// ImportState recovers the rule set for a step from its tournament step id.
// The preset id and parameters are not echoed back by the API (only the
// resulting rule set is), so they are not populated on import — set
// preset_script_id (and parameters) in configuration to match.
func (r *tournamentStepPresetResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("tournament_step_id"), req, resp)
}

// applyPreset runs the applyPresetScript mutation and returns the resulting
// rule set.
func (r *tournamentStepPresetResource) applyPreset(ctx context.Context, m *tournamentStepPresetModel) (*gqlStepRuleSet, error) {
	vars := map[string]any{
		"presetScriptId":   m.PresetScriptID.ValueString(),
		"tournamentStepId": m.TournamentStepID.ValueString(),
	}
	putStr(vars, "parameters", m.Parameters)

	query := `mutation($presetScriptId: ID!, $tournamentStepId: ID!, $parameters: String) { ` +
		`applyPresetScript(presetScriptId: $presetScriptId, tournamentStepId: $tournamentStepId, parameters: $parameters) { ` +
		stepRuleSetFields + ` } }`
	var out struct {
		ApplyPresetScript gqlStepRuleSet `json:"applyPresetScript"`
	}
	if err := r.client.Execute(ctx, query, vars, &out); err != nil {
		return nil, err
	}
	return &out.ApplyPresetScript, nil
}

// readRuleSet fetches the current rule set for a step, returning nil when the
// step has none.
func (r *tournamentStepPresetResource) readRuleSet(ctx context.Context, tournamentStepID string) (*gqlStepRuleSet, error) {
	query := `query($tournamentStepId: ID!) { stepRuleSet(tournamentStepId: $tournamentStepId) { ` + stepRuleSetFields + ` } }`
	var out struct {
		StepRuleSet *gqlStepRuleSet `json:"stepRuleSet"`
	}
	if err := r.client.Execute(ctx, query, map[string]any{"tournamentStepId": tournamentStepID}, &out); err != nil {
		if isStepRuleSetNotFoundErr(err) {
			return nil, nil
		}
		return nil, err
	}
	return out.StepRuleSet, nil
}

// validateRuleSet starts a validation job for the rule set, polls it to
// completion, and fails (via diags) if the rule set is invalid. On success it
// returns the refreshed rule set (validated/validatedAt updated), or nil if the
// refresh could not be performed.
func (r *tournamentStepPresetResource) validateRuleSet(ctx context.Context, tournamentStepID, stepRuleSetID string, diags *diag.Diagnostics) *gqlStepRuleSet {
	startQuery := `mutation($stepRuleSetId: ID!) { validateStepRuleSet(stepRuleSetId: $stepRuleSetId) { ` + validationJobFields + ` } }`
	var started struct {
		ValidateStepRuleSet gqlValidationJob `json:"validateStepRuleSet"`
	}
	if err := r.client.Execute(ctx, startQuery, map[string]any{"stepRuleSetId": stepRuleSetID}, &started); err != nil {
		diags.AddError("Unable to start rule set validation", err.Error())
		return nil
	}

	job := started.ValidateStepRuleSet
	pollQuery := `query($jobId: ID!) { stepRuleSetValidationJob(jobId: $jobId) { ` + validationJobFields + ` } }`
	for !isTerminalValidationStatus(job.Status) {
		select {
		case <-ctx.Done():
			diags.AddError("Timed out waiting for rule set validation", ctx.Err().Error())
			return nil
		case <-time.After(validationPollInterval):
		}

		var polled struct {
			StepRuleSetValidationJob *gqlValidationJob `json:"stepRuleSetValidationJob"`
		}
		if err := r.client.Execute(ctx, pollQuery, map[string]any{"jobId": job.ID}, &polled); err != nil {
			diags.AddError("Unable to poll rule set validation job", err.Error())
			return nil
		}
		if polled.StepRuleSetValidationJob == nil {
			diags.AddError("Rule set validation job disappeared", "Validation job "+job.ID+" could no longer be found.")
			return nil
		}
		job = *polled.StepRuleSetValidationJob
	}

	if job.Status != "SUCCEEDED" {
		diags.AddError("Tournament step rule set is invalid", job.failureDetail())
		return nil
	}

	// Validation succeeded; refresh validated/validatedAt from the rule set.
	refreshed, err := r.readRuleSet(ctx, tournamentStepID)
	if err != nil {
		tflog.Warn(ctx, "Validation succeeded but refreshing the rule set failed", map[string]any{"error": err.Error()})
		return nil
	}
	return refreshed
}

// isTerminalValidationStatus reports whether a ValidationJobStatus is final.
func isTerminalValidationStatus(status string) bool {
	switch status {
	case "SUCCEEDED", "FAILED", "CANCELLED":
		return true
	default:
		return false
	}
}

// isStepRuleSetNotFoundErr reports whether a GraphQL error indicates the step or
// its rule set is gone.
func isStepRuleSetNotFoundErr(err error) bool {
	s := strings.ToLower(err.Error())
	return strings.Contains(s, "not found") ||
		strings.Contains(s, "does not exist") ||
		strings.Contains(s, "no rule set") ||
		strings.Contains(s, "no step")
}
