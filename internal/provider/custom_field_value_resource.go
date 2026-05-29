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
	_ resource.Resource                = &customFieldValueResource{}
	_ resource.ResourceWithConfigure   = &customFieldValueResource{}
	_ resource.ResourceWithImportState = &customFieldValueResource{}
)

// NewCustomFieldValueResource constructs the wellplayed_custom_field_value
// resource.
func NewCustomFieldValueResource() resource.Resource {
	return &customFieldValueResource{}
}

type customFieldValueResource struct {
	client *client.Client
}

// customFieldValueModel maps the resource schema to Go values. The triple
// (object_type, object_id, key) is the identity; value is the only mutable
// attribute.
type customFieldValueModel struct {
	ID         types.String `tfsdk:"id"`
	ObjectType types.String `tfsdk:"object_type"`
	ObjectID   types.String `tfsdk:"object_id"`
	Key        types.String `tfsdk:"key"`
	Value      types.String `tfsdk:"value"`
}

func (r *customFieldValueResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_custom_field_value"
}

func (r *customFieldValueResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Sets the value of a single custom field on an object (Tournament, Account, " +
			"Team, etc.) within the organization. Backed by the `setCustomFieldValues` mutation and the " +
			"`customFieldValues` query.\n\n" +
			"The field definition itself must already exist — manage it with " +
			"`wellplayed_custom_field_definition`. Each resource owns exactly one `(object_type, " +
			"object_id, key)` triple; those three fields are immutable and changing any of them forces " +
			"replacement.\n\n" +
			"WellPlayed has no API to unset a custom field value, so `terraform destroy` clears the value " +
			"by setting it to an empty string. For typed fields (`SELECT`, `NUMBER`, regex-validated, " +
			"etc.) an empty string may be rejected by server-side validation.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Synthetic identifier of the form `<object_type>:<object_id>:<key>`.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"object_type": schema.StringAttribute{
				MarkdownDescription: "Object type the value is attached to (e.g. `Tournament`, `Account`, " +
					"`Team`, `Organization`). See the `ObjectType` enum in the WellPlayed schema for the " +
					"full list. Immutable.",
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"object_id": schema.StringAttribute{
				MarkdownDescription: "Identifier of the specific object instance the value is set on. Immutable.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"key": schema.StringAttribute{
				MarkdownDescription: "Key of the custom field definition whose value is being set. The " +
					"definition must already exist for this object type. Immutable.",
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"value": schema.StringAttribute{
				MarkdownDescription: "Value to set, serialized as a string. Its interpretation depends on the " +
					"field definition's `type` (e.g. `\"true\"` for a `BOOLEAN`, the option value for a " +
					"`SELECT`).",
				Required: true,
			},
		},
	}
}

func (r *customFieldValueResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *customFieldValueResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan customFieldValueModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.setValue(ctx, &plan, plan.Value.ValueString()); err != nil {
		resp.Diagnostics.AddError("Unable to set custom field value", err.Error())
		return
	}

	plan.ID = types.StringValue(customFieldValueID(&plan))
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *customFieldValueResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state customFieldValueModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// customFieldValues returns one entry per definition for the object type,
	// each carrying its value (or null). Filter to the key held in state.
	query := `query($objectType: ObjectType!, $objectId: ID!) { customFieldValues(objectType: $objectType, objectId: $objectId) { ` + customFieldWithValueFields + ` } }`
	vars := map[string]any{
		"objectType": state.ObjectType.ValueString(),
		"objectId":   state.ObjectID.ValueString(),
	}
	var out struct {
		CustomFieldValues []gqlCustomFieldWithValue `json:"customFieldValues"`
	}
	if err := r.client.Execute(ctx, query, vars, &out); err != nil {
		resp.Diagnostics.AddError("Unable to read custom field values", err.Error())
		return
	}

	key := state.Key.ValueString()
	var value *string
	for i := range out.CustomFieldValues {
		if out.CustomFieldValues[i].Definition.Key == key {
			value = out.CustomFieldValues[i].Value
			break
		}
	}

	// A null value means the field is unset on the object (never set, or cleared
	// out of band) — treat it as gone so the next apply recreates it.
	if value == nil {
		tflog.Warn(ctx, "Custom field value not set, removing from state", map[string]any{
			"object_type": state.ObjectType.ValueString(),
			"object_id":   state.ObjectID.ValueString(),
			"key":         key,
		})
		resp.State.RemoveResource(ctx)
		return
	}

	state.Value = types.StringValue(*value)
	state.ID = types.StringValue(customFieldValueID(&state))
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *customFieldValueResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan customFieldValueModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.setValue(ctx, &plan, plan.Value.ValueString()); err != nil {
		resp.Diagnostics.AddError("Unable to update custom field value", err.Error())
		return
	}

	plan.ID = types.StringValue(customFieldValueID(&plan))
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *customFieldValueResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state customFieldValueModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// There is no unset/delete mutation, so clear the value by setting it empty.
	if err := r.setValue(ctx, &state, ""); err != nil {
		resp.Diagnostics.AddError("Unable to clear custom field value", err.Error())
		return
	}
}

// ImportState accepts a composite "<object_type>:<object_id>:<key>" because the
// value is identified by that triple and there is no standalone lookup by id.
func (r *customFieldValueResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, ":", 3)
	if len(parts) != 3 || parts[0] == "" || parts[1] == "" || parts[2] == "" {
		resp.Diagnostics.AddError(
			"Invalid import id",
			"Expected '<object_type>:<object_id>:<key>' (e.g. 'Tournament:trn_01h...:sponsor'). Got: "+req.ID,
		)
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("object_type"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("object_id"), parts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("key"), parts[2])...)
}

// setValue upserts the single (key, value) pair this resource owns onto the
// object via setCustomFieldValues.
func (r *customFieldValueResource) setValue(ctx context.Context, m *customFieldValueModel, value string) error {
	input := buildSetCustomFieldValuesInput(m, value)
	query := `mutation($input: SetCustomFieldValuesInput!) { setCustomFieldValues(input: $input) { ` + customFieldWithValueFields + ` } }`
	return r.client.Execute(ctx, query, map[string]any{"input": input}, nil)
}

// customFieldValueID builds the synthetic "<object_type>:<object_id>:<key>" id.
func customFieldValueID(m *customFieldValueModel) string {
	return m.ObjectType.ValueString() + ":" + m.ObjectID.ValueString() + ":" + m.Key.ValueString()
}
