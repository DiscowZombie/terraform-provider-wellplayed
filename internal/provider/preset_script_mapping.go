package provider

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// presetScriptFields selects the full readable shape of PresetScriptModel.
// Every authored field round-trips through Read, so nothing is omitted.
const presetScriptFields = `
id
name
description
script
parameters {
  name
  type
  required
  defaultValue
  description
}
createdAt
updatedAt`

// gqlPresetScript mirrors the readable shape of PresetScriptModel.
type gqlPresetScript struct {
	ID          string           `json:"id"`
	Name        string           `json:"name"`
	Description *string          `json:"description"`
	Script      string           `json:"script"`
	Parameters  []gqlPresetParam `json:"parameters"`
	CreatedAt   string           `json:"createdAt"`
	UpdatedAt   string           `json:"updatedAt"`
}

type gqlPresetParam struct {
	Name         string  `json:"name"`
	Type         string  `json:"type"`
	Required     bool    `json:"required"`
	DefaultValue *string `json:"defaultValue"`
	Description  *string `json:"description"`
}

// --- Plan -> GraphQL input --------------------------------------------------

// buildPresetScriptInput maps the model to the create/update input. The two
// inputs share the same field set, so the same builder serves both. The
// `parameters` field on CreatePresetScriptInput is non-null, so an absent
// block sends `[]` rather than being omitted.
func buildPresetScriptInput(m *presetScriptModel) map[string]any {
	input := map[string]any{
		"name":   m.Name.ValueString(),
		"script": m.Script.ValueString(),
	}
	putStr(input, "description", m.Description)

	params := make([]map[string]any, 0, len(m.Parameters))
	for i := range m.Parameters {
		p := &m.Parameters[i]
		entry := map[string]any{
			"name":     p.Name.ValueString(),
			"type":     p.Type.ValueString(),
			"required": p.Required.ValueBool(),
		}
		putStr(entry, "defaultValue", p.DefaultValue)
		putStr(entry, "description", p.Description)
		params = append(params, entry)
	}
	input["parameters"] = params

	return input
}

// --- GraphQL response -> model ----------------------------------------------

// applyPresetScriptToModel overwrites the model with the server's view of the
// preset. Used after create, update, and read.
func applyPresetScriptToModel(g *gqlPresetScript, m *presetScriptModel) {
	m.ID = types.StringValue(g.ID)
	m.Name = types.StringValue(g.Name)
	m.Description = strVal(g.Description)
	m.Script = types.StringValue(g.Script)
	m.CreatedAt = types.StringValue(g.CreatedAt)
	m.UpdatedAt = types.StringValue(g.UpdatedAt)

	params := make([]presetParameterModel, 0, len(g.Parameters))
	for _, p := range g.Parameters {
		params = append(params, presetParameterModel{
			Name:         types.StringValue(p.Name),
			Type:         types.StringValue(p.Type),
			Required:     types.BoolValue(p.Required),
			DefaultValue: strVal(p.DefaultValue),
			Description:  strVal(p.Description),
		})
	}
	m.Parameters = params
}
