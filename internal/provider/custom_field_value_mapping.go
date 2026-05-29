package provider

// customFieldWithValueFields selects the minimal shape needed to refresh a
// single custom field value: the owning definition's key (the stable lookup
// name) and the serialized value.
const customFieldWithValueFields = `
definition {
  key
}
value`

// gqlCustomFieldWithValue mirrors the readable shape selected by
// customFieldWithValueFields. value is nullable: the API returns one entry per
// definition for the object type, with a null value when nothing is set.
type gqlCustomFieldWithValue struct {
	Definition struct {
		Key string `json:"key"`
	} `json:"definition"`
	Value *string `json:"value"`
}

// buildSetCustomFieldValuesInput maps the model to the setCustomFieldValues
// input, sending the single (key, value) pair this resource owns. The mutation
// is an upsert over the `fields` list, so other keys on the object are left
// untouched.
func buildSetCustomFieldValuesInput(m *customFieldValueModel, value string) map[string]any {
	return map[string]any{
		"objectType": m.ObjectType.ValueString(),
		"objectId":   m.ObjectID.ValueString(),
		"fields": []map[string]any{
			{
				"key":   m.Key.ValueString(),
				"value": value,
			},
		},
	}
}
