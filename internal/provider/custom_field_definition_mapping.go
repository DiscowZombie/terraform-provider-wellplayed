package provider

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// customFieldDefinitionFields selects the full readable shape of
// CustomFieldDefinitionModel. Every authored field round-trips through Read,
// so nothing is omitted.
const customFieldDefinitionFields = `
id
objectType
key
name
description
type
required
unique
order
visibility
editability
options {
  label
  value
}
defaultValue
validationRegex
createdAt
updatedAt`

// gqlCustomFieldDefinition mirrors the readable shape of
// CustomFieldDefinitionModel.
type gqlCustomFieldDefinition struct {
	ID              string                 `json:"id"`
	ObjectType      string                 `json:"objectType"`
	Key             string                 `json:"key"`
	Name            string                 `json:"name"`
	Description     *string                `json:"description"`
	Type            string                 `json:"type"`
	Required        bool                   `json:"required"`
	Unique          bool                   `json:"unique"`
	Order           int64                  `json:"order"`
	Visibility      string                 `json:"visibility"`
	Editability     string                 `json:"editability"`
	Options         []gqlCustomFieldOption `json:"options"`
	DefaultValue    *string                `json:"defaultValue"`
	ValidationRegex *string                `json:"validationRegex"`
	CreatedAt       string                 `json:"createdAt"`
	UpdatedAt       string                 `json:"updatedAt"`
}

type gqlCustomFieldOption struct {
	Label string `json:"label"`
	Value string `json:"value"`
}

// --- Plan -> GraphQL input --------------------------------------------------

// buildCreateCustomFieldDefinitionInput maps the model to the create input.
// The full set of fields is sent — including immutable identity fields
// (objectType, key, type) — because they only ever flow into Create.
func buildCreateCustomFieldDefinitionInput(m *customFieldDefinitionModel) map[string]any {
	input := map[string]any{
		"objectType":  m.ObjectType.ValueString(),
		"key":         m.Key.ValueString(),
		"name":        m.Name.ValueString(),
		"type":        m.Type.ValueString(),
		"visibility":  m.Visibility.ValueString(),
		"editability": m.Editability.ValueString(),
	}
	putStr(input, "description", m.Description)
	putBool(input, "required", m.Required)
	putBool(input, "unique", m.Unique)
	putInt(input, "order", m.Order)
	putStr(input, "defaultValue", m.DefaultValue)
	putStr(input, "validationRegex", m.ValidationRegex)
	if opts, ok := buildCustomFieldOptions(m.Options); ok {
		input["options"] = opts
	}
	return input
}

// buildUpdateCustomFieldDefinitionInput maps the model to the update input.
// objectType, key, and type are not in UpdateCustomFieldDefinitionInput — the
// schema marks them RequiresReplace, so the resource is recreated when they
// change.
func buildUpdateCustomFieldDefinitionInput(m *customFieldDefinitionModel) map[string]any {
	input := map[string]any{
		"name":        m.Name.ValueString(),
		"visibility":  m.Visibility.ValueString(),
		"editability": m.Editability.ValueString(),
	}
	putStr(input, "description", m.Description)
	putBool(input, "required", m.Required)
	putBool(input, "unique", m.Unique)
	putInt(input, "order", m.Order)
	putStr(input, "defaultValue", m.DefaultValue)
	putStr(input, "validationRegex", m.ValidationRegex)
	if opts, ok := buildCustomFieldOptions(m.Options); ok {
		input["options"] = opts
	}
	return input
}

// buildCustomFieldOptions converts the nested options block to a list of
// GraphQL inputs. Returns ok=false when the block was not authored so the
// caller can omit the key entirely (the API treats null and absent the same).
func buildCustomFieldOptions(opts []customFieldOptionModel) ([]map[string]any, bool) {
	if opts == nil {
		return nil, false
	}
	out := make([]map[string]any, 0, len(opts))
	for i := range opts {
		out = append(out, map[string]any{
			"label": opts[i].Label.ValueString(),
			"value": opts[i].Value.ValueString(),
		})
	}
	return out, true
}

// --- GraphQL response -> model ----------------------------------------------

// applyCustomFieldDefinitionToModel overwrites the model with the server's
// view of the definition. Used after create, update, and read.
func applyCustomFieldDefinitionToModel(g *gqlCustomFieldDefinition, m *customFieldDefinitionModel) {
	m.ID = types.StringValue(g.ID)
	m.ObjectType = types.StringValue(g.ObjectType)
	m.Key = types.StringValue(g.Key)
	m.Name = types.StringValue(g.Name)
	m.Description = strVal(g.Description)
	m.Type = types.StringValue(g.Type)
	m.Required = types.BoolValue(g.Required)
	m.Unique = types.BoolValue(g.Unique)
	m.Order = types.Int64Value(g.Order)
	m.Visibility = types.StringValue(g.Visibility)
	m.Editability = types.StringValue(g.Editability)
	m.DefaultValue = strVal(g.DefaultValue)
	m.ValidationRegex = strVal(g.ValidationRegex)
	m.CreatedAt = types.StringValue(g.CreatedAt)
	m.UpdatedAt = types.StringValue(g.UpdatedAt)

	if g.Options == nil {
		m.Options = nil
		return
	}
	opts := make([]customFieldOptionModel, 0, len(g.Options))
	for _, o := range g.Options {
		opts = append(opts, customFieldOptionModel{
			Label: types.StringValue(o.Label),
			Value: types.StringValue(o.Value),
		})
	}
	m.Options = opts
}

// putBool adds v to m under key only when it is a known, non-null value.
func putBool(m map[string]any, key string, v types.Bool) {
	if !v.IsNull() && !v.IsUnknown() {
		m[key] = v.ValueBool()
	}
}

// putInt adds v to m under key only when it is a known, non-null value.
func putInt(m map[string]any, key string, v types.Int64) {
	if !v.IsNull() && !v.IsUnknown() {
		m[key] = v.ValueInt64()
	}
}
