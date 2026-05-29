package provider

import (
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

// stepRuleSetFields selects the readable scalar subset of StepRuleSetModel that
// this resource refreshes. The relational sub-resolvers (scoringRuleSet,
// advancementRules, crossStepRules, structureTemplate) describe the generated
// rule set itself and are not managed by this resource.
const stepRuleSetFields = `
id
version
presetName
teamCount
validated
validatedAt
createdAt
updatedAt`

// gqlStepRuleSet mirrors the readable scalar subset of StepRuleSetModel.
type gqlStepRuleSet struct {
	ID          string  `json:"id"`
	Version     int64   `json:"version"`
	PresetName  *string `json:"presetName"`
	TeamCount   *int64  `json:"teamCount"`
	Validated   bool    `json:"validated"`
	ValidatedAt *string `json:"validatedAt"`
	CreatedAt   string  `json:"createdAt"`
	UpdatedAt   string  `json:"updatedAt"`
}

// validationJobFields selects what this resource needs to track an async
// validation job to completion and surface a useful failure reason: the
// terminal status, the one-line errorSummary, and the structured per-error
// messages from the result.
const validationJobFields = `
id
status
errorSummary
result {
  success
  errors {
    code
    message
    hint
  }
}`

// gqlValidationJob mirrors the parts of ValidationJobModel this resource uses.
// Status is one of the ValidationJobStatus enum values: QUEUED, RUNNING,
// SUCCEEDED, FAILED, CANCELLED.
type gqlValidationJob struct {
	ID           string               `json:"id"`
	Status       string               `json:"status"`
	ErrorSummary *string              `json:"errorSummary"`
	Result       *gqlValidationResult `json:"result"`
}

type gqlValidationResult struct {
	Success bool                 `json:"success"`
	Errors  []gqlValidationError `json:"errors"`
}

type gqlValidationError struct {
	Code    string  `json:"code"`
	Message string  `json:"message"`
	Hint    *string `json:"hint"`
}

// failureDetail builds a human-readable explanation of why a validation job did
// not succeed, combining the job status, its error summary, and any structured
// per-error messages.
func (j *gqlValidationJob) failureDetail() string {
	var b strings.Builder
	b.WriteString("Validation job " + j.ID + " ended with status " + j.Status + ".")
	if j.ErrorSummary != nil && *j.ErrorSummary != "" {
		b.WriteString("\n\n" + *j.ErrorSummary)
	}
	if j.Result != nil {
		for _, e := range j.Result.Errors {
			b.WriteString("\n\n- ")
			if e.Code != "" {
				b.WriteString("[" + e.Code + "] ")
			}
			b.WriteString(e.Message)
			if e.Hint != nil && *e.Hint != "" {
				b.WriteString("\n  hint: " + *e.Hint)
			}
		}
	}
	return b.String()
}

// applyStepRuleSetToModel overlays the server-owned rule set fields onto the
// model after apply, read, and update. The authored input fields
// (tournament_step_id, preset_script_id, parameters, validate) are left
// untouched so the applied state matches the plan.
func applyStepRuleSetToModel(g *gqlStepRuleSet, m *tournamentStepPresetModel) {
	m.ID = types.StringValue(g.ID)
	m.Version = types.Int64Value(g.Version)
	m.PresetName = strVal(g.PresetName)
	m.TeamCount = int64PtrVal(g.TeamCount)
	m.Validated = types.BoolValue(g.Validated)
	m.ValidatedAt = strVal(g.ValidatedAt)
	m.CreatedAt = types.StringValue(g.CreatedAt)
	m.UpdatedAt = types.StringValue(g.UpdatedAt)
}

// int64PtrVal converts an optional int64 pointer to a framework value.
func int64PtrVal(p *int64) types.Int64 {
	if p == nil {
		return types.Int64Null()
	}
	return types.Int64Value(*p)
}
