package provider

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// tournamentStepFields selects the readable scalar subset of TournamentStep.
// The TournamentStep type has no `properties` field, so properties is managed
// write-only (sent on create/update, restored from prior state during Read).
// Relational/sub-resolver fields (configuration, seedingOrder, teamScores,
// tournament, manualPins) are out of scope for this resource.
const tournamentStepFields = `
id
tournamentId
name
description
order
type
status
createdAt
updatedAt`

// gqlTournamentStep mirrors the readable scalar subset of the TournamentStep type.
type gqlTournamentStep struct {
	ID           string  `json:"id"`
	TournamentID string  `json:"tournamentId"`
	Name         string  `json:"name"`
	Description  string  `json:"description"`
	Order        float64 `json:"order"`
	Type         string  `json:"type"`
	Status       string  `json:"status"`
	CreatedAt    string  `json:"createdAt"`
	UpdatedAt    string  `json:"updatedAt"`
}

// --- Plan -> GraphQL input --------------------------------------------------

// buildTournamentStepInput maps the model to the CreateTournamentStepInput used
// by both the create and update mutations. The optional bracket `configuration`
// argument is out of scope and never sent. `properties` is a nullable list, so
// an absent block is omitted rather than sent as `[]`.
func buildTournamentStepInput(m *tournamentStepModel) map[string]any {
	step := map[string]any{
		"name":        m.Name.ValueString(),
		"description": m.Description.ValueString(),
		"order":       m.Order.ValueFloat64(),
		"type":        m.Type.ValueString(),
	}

	if m.Properties != nil {
		props := make([]map[string]any, 0, len(m.Properties))
		for i := range m.Properties {
			props = append(props, map[string]any{
				"property": m.Properties[i].Property.ValueString(),
				"value":    m.Properties[i].Value.ValueString(),
			})
		}
		step["properties"] = props
	}

	return step
}

// --- GraphQL response -> model ----------------------------------------------

// applyComputedToStepModel overlays the server-owned fields onto a plan-sourced
// model after create/update. Authored fields (tournament_id, name, description,
// order, type, properties) are left untouched so the applied state matches the
// plan.
func applyComputedToStepModel(g *gqlTournamentStep, m *tournamentStepModel) {
	m.ID = types.StringValue(g.ID)
	m.Status = types.StringValue(g.Status)
	m.CreatedAt = types.StringValue(g.CreatedAt)
	m.UpdatedAt = types.StringValue(g.UpdatedAt)
}

// modelFromStepGQL reconstructs the refreshable fields from a Read response.
// properties is left nil: Read restores it from prior state because the API
// does not expose it on TournamentStep.
func modelFromStepGQL(g *gqlTournamentStep) *tournamentStepModel {
	return &tournamentStepModel{
		ID:           types.StringValue(g.ID),
		TournamentID: types.StringValue(g.TournamentID),
		Name:         types.StringValue(g.Name),
		Description:  types.StringValue(g.Description),
		Order:        types.Float64Value(g.Order),
		Type:         types.StringValue(g.Type),
		Status:       types.StringValue(g.Status),
		CreatedAt:    types.StringValue(g.CreatedAt),
		UpdatedAt:    types.StringValue(g.UpdatedAt),
	}
}
