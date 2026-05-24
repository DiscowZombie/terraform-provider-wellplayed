// Copyright (c) Mathéo Cimbaro
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"github.com/hashicorp/terraform-plugin-framework-timetypes/timetypes"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// tournamentComputedFields selects the server-owned fields refreshed onto the
// model after a create/update mutation. Authored fields are kept from the plan
// to avoid "inconsistent result after apply" errors and timestamp permadiffs.
const tournamentComputedFields = `id organizationId createdById createdAt updatedAt tournamentSeriesId visibleAt`

// tournamentReadFields selects what Read refreshes from the API. configuration
// is deliberately omitted: the API auto-provisions a default configuration, so
// it is managed write-only from config rather than refreshed (see Read).
// teams/steps/createdBy/myTeam/teamScores are relational/caller-dependent
// sub-resolvers and are likewise excluded.
const tournamentReadFields = `
id
title
description
startAt
endAt
startRegistrationsAt
endRegistrationsAt
visibleAt
organizationId
tournamentSeriesId
createdById
createdAt
updatedAt`

// gqlTournament mirrors the readable scalar subset of the Tournament type.
type gqlTournament struct {
	ID                   string  `json:"id"`
	Title                string  `json:"title"`
	Description          string  `json:"description"`
	StartAt              *string `json:"startAt"`
	EndAt                *string `json:"endAt"`
	StartRegistrationsAt *string `json:"startRegistrationsAt"`
	EndRegistrationsAt   *string `json:"endRegistrationsAt"`
	VisibleAt            *string `json:"visibleAt"`
	OrganizationID       string  `json:"organizationId"`
	TournamentSeriesID   *string `json:"tournamentSeriesId"`
	CreatedByID          string  `json:"createdById"`
	CreatedAt            string  `json:"createdAt"`
	UpdatedAt            string  `json:"updatedAt"`
}

// --- Plan -> GraphQL input --------------------------------------------------

// buildTournamentInput maps the model to the create/update input. The two
// inputs share the same field set, so the same builder serves both.
func buildTournamentInput(m *tournamentModel) map[string]any {
	input := map[string]any{
		"title":       m.Title.ValueString(),
		"description": m.Description.ValueString(),
	}
	putTime(input, "startAt", m.StartAt)
	putTime(input, "endAt", m.EndAt)
	putTime(input, "startRegistrationsAt", m.StartRegistrationsAt)
	putTime(input, "endRegistrationsAt", m.EndRegistrationsAt)
	putTime(input, "visibleAt", m.VisibleAt)

	if m.Configuration != nil {
		// Wrapped in UpdateTournamentConfigurationOrImportFromIdInput: this
		// resource always supplies an inline configuration.
		input["configuration"] = map[string]any{"configuration": buildConfigurationInput(m.Configuration)}
	}
	return input
}

func buildConfigurationInput(c *configModel) map[string]any {
	out := map[string]any{}
	// type is Optional+Computed with a TOURNAMENT default, so it is always known.
	if !c.Type.IsNull() && !c.Type.IsUnknown() {
		out["type"] = c.Type.ValueString()
	}
	putFloat(out, "teamMinSize", c.TeamMinSize)
	putFloat(out, "teamMaxSize", c.TeamMaxSize)
	putFloat(out, "teamsCount", c.TeamsCount)
	putStr(out, "teamStatusAfterRegistration", c.TeamStatusAfterRegistration)

	if c.RegistrationConditions != nil {
		// Both lists are non-null in the schema: always send arrays, never omit.
		teamConds := make([]map[string]any, 0, len(c.RegistrationConditions.TeamConditions))
		for i := range c.RegistrationConditions.TeamConditions {
			teamConds = append(teamConds, buildTeamConditionInput(&c.RegistrationConditions.TeamConditions[i]))
		}
		memberConds := make([]map[string]any, 0, len(c.RegistrationConditions.MemberConditions))
		for i := range c.RegistrationConditions.MemberConditions {
			memberConds = append(memberConds, buildMemberConditionInput(&c.RegistrationConditions.MemberConditions[i]))
		}
		out["registrationConditions"] = map[string]any{
			"teamConditions":   teamConds,
			"memberConditions": memberConds,
		}
	}

	if c.CustomFields != nil {
		fields := make([]map[string]any, 0, len(c.CustomFields))
		for i := range c.CustomFields {
			fields = append(fields, buildPropertyInput(&c.CustomFields[i]))
		}
		out["customFields"] = fields
	}
	return out
}

func buildTeamConditionInput(t *teamConditionModel) map[string]any {
	out := map[string]any{
		"property":          t.Property.ValueString(),
		"propertyCondition": t.PropertyCondition.ValueString(),
	}
	putStr(out, "errorMessage", t.ErrorMessage)
	if t.StringCondition != nil {
		out["stringCondition"] = buildStringConditionInput(t.StringCondition)
	}
	if t.NumericCondition != nil {
		nc := map[string]any{
			"conditionType": t.NumericCondition.ConditionType.ValueString(),
			"value":         t.NumericCondition.Value.ValueFloat64(),
		}
		putStr(nc, "aggregationType", t.NumericCondition.AggregationType)
		putStr(nc, "propertySource", t.NumericCondition.PropertySource)
		putStr(nc, "propertySourceId", t.NumericCondition.PropertySourceID)
		out["numericCondition"] = nc
	}
	return out
}

func buildMemberConditionInput(m *memberConditionModel) map[string]any {
	out := map[string]any{
		"propertySource": m.PropertySource.ValueString(),
	}
	putStr(out, "propertySourceId", m.PropertySourceID)
	putStr(out, "errorMessage", m.ErrorMessage)
	putStr(out, "ruleDescription", m.RuleDescription)
	if m.Condition != nil {
		cond := map[string]any{
			"property":          m.Condition.Property.ValueString(),
			"propertyCondition": m.Condition.PropertyCondition.ValueString(),
		}
		if m.Condition.StringCondition != nil {
			cond["stringCondition"] = buildStringConditionInput(m.Condition.StringCondition)
		}
		if m.Condition.NumericCondition != nil {
			cond["numericCondition"] = map[string]any{
				"conditionType": m.Condition.NumericCondition.ConditionType.ValueString(),
				"value":         m.Condition.NumericCondition.Value.ValueFloat64(),
			}
		}
		out["condition"] = cond
	}
	return out
}

func buildStringConditionInput(s *stringConditionModel) map[string]any {
	return map[string]any{
		"conditionType": s.ConditionType.ValueString(),
		"value":         s.Value.ValueString(),
	}
}

func buildPropertyInput(p *propertyModel) map[string]any {
	out := map[string]any{
		"property": p.Property.ValueString(),
		"name":     p.Name.ValueString(),
		"type":     p.Type.ValueString(),
		"required": p.Required.ValueBool(),
		"order":    p.Order.ValueFloat64(),
		"unique":   p.Unique.ValueBool(),
	}
	putStr(out, "visibility", p.Visibility)
	putStr(out, "editability", p.Editability)
	return out
}

// putStr adds v to m under key only when it is a known, non-null value.
func putStr(m map[string]any, key string, v types.String) {
	if !v.IsNull() && !v.IsUnknown() {
		m[key] = v.ValueString()
	}
}

// putFloat adds v to m under key only when it is a known, non-null value.
func putFloat(m map[string]any, key string, v types.Float64) {
	if !v.IsNull() && !v.IsUnknown() {
		m[key] = v.ValueFloat64()
	}
}

// putTime adds the RFC 3339 string of v to m under key when it is a known,
// non-null value.
func putTime(m map[string]any, key string, v timetypes.RFC3339) {
	if !v.IsNull() && !v.IsUnknown() {
		m[key] = v.ValueString()
	}
}

// --- GraphQL response -> model ----------------------------------------------

// applyComputedToModel overlays the server-owned fields onto a plan-sourced
// model after create/update. Authored fields are left untouched so the applied
// state matches the plan; visible_at is filled only when the plan left it
// unknown (Optional+Computed).
func applyComputedToModel(g *gqlTournament, m *tournamentModel) {
	m.ID = types.StringValue(g.ID)
	m.OrganizationID = types.StringValue(g.OrganizationID)
	m.CreatedByID = types.StringValue(g.CreatedByID)
	m.CreatedAt = types.StringValue(g.CreatedAt)
	m.UpdatedAt = types.StringValue(g.UpdatedAt)
	m.TournamentSeriesID = strVal(g.TournamentSeriesID)
	if m.VisibleAt.IsUnknown() {
		m.VisibleAt = timeVal(g.VisibleAt)
	}
}

// modelFromGQL reconstructs the refreshable fields from a Read response.
// configuration is left nil: Read restores it from prior state.
func modelFromGQL(g *gqlTournament) *tournamentModel {
	return &tournamentModel{
		ID:                   types.StringValue(g.ID),
		Title:                types.StringValue(g.Title),
		Description:          types.StringValue(g.Description),
		StartAt:              timeVal(g.StartAt),
		EndAt:                timeVal(g.EndAt),
		StartRegistrationsAt: timeVal(g.StartRegistrationsAt),
		EndRegistrationsAt:   timeVal(g.EndRegistrationsAt),
		VisibleAt:            timeVal(g.VisibleAt),
		OrganizationID:       types.StringValue(g.OrganizationID),
		TournamentSeriesID:   strVal(g.TournamentSeriesID),
		CreatedByID:          types.StringValue(g.CreatedByID),
		CreatedAt:            types.StringValue(g.CreatedAt),
		UpdatedAt:            types.StringValue(g.UpdatedAt),
	}
}

// strVal converts an optional string pointer to a framework value.
func strVal(p *string) types.String {
	if p == nil {
		return types.StringNull()
	}
	return types.StringValue(*p)
}

// timeVal converts an optional RFC 3339 string pointer to a framework value.
// The API returns valid RFC 3339, so a non-nil value always parses.
func timeVal(p *string) timetypes.RFC3339 {
	if p == nil {
		return timetypes.NewRFC3339Null()
	}
	return timetypes.NewRFC3339ValueMust(*p)
}
