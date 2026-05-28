package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/DiscowZombie/terraform-provider-wellplayed/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure the resource satisfies the framework interfaces.
var (
	_ resource.Resource                = &customFieldDefinitionResource{}
	_ resource.ResourceWithConfigure   = &customFieldDefinitionResource{}
	_ resource.ResourceWithImportState = &customFieldDefinitionResource{}
)

// NewCustomFieldDefinitionResource constructs the
// wellplayed_custom_field_definition resource.
func NewCustomFieldDefinitionResource() resource.Resource {
	return &customFieldDefinitionResource{}
}

type customFieldDefinitionResource struct {
	client *client.Client
}

// customFieldDefinitionModel maps the resource schema to Go values. All
// authored fields round-trip through Read.
type customFieldDefinitionModel struct {
	ID              types.String             `tfsdk:"id"`
	ObjectType      types.String             `tfsdk:"object_type"`
	Key             types.String             `tfsdk:"key"`
	Name            types.String             `tfsdk:"name"`
	Description     types.String             `tfsdk:"description"`
	Type            types.String             `tfsdk:"type"`
	Required        types.Bool               `tfsdk:"required"`
	Unique          types.Bool               `tfsdk:"unique"`
	Order           types.Int64              `tfsdk:"order"`
	Visibility      types.String             `tfsdk:"visibility"`
	Editability     types.String             `tfsdk:"editability"`
	Options         []customFieldOptionModel `tfsdk:"options"`
	DefaultValue    types.String             `tfsdk:"default_value"`
	ValidationRegex types.String             `tfsdk:"validation_regex"`
	CreatedAt       types.String             `tfsdk:"created_at"`
	UpdatedAt       types.String             `tfsdk:"updated_at"`
}

type customFieldOptionModel struct {
	Label types.String `tfsdk:"label"`
	Value types.String `tfsdk:"value"`
}

func (r *customFieldDefinitionResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_custom_field_definition"
}

func (r *customFieldDefinitionResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a WellPlayed custom field definition attached to an object type " +
			"(Tournament, Account, Team, etc.) within the organization. Backed by the " +
			"`createCustomFieldDefinition`, `updateCustomFieldDefinition`, and " +
			"`deleteCustomFieldDefinition` mutations.\n\n" +
			"Custom field *values* (per-object data filled in by users) are not managed by this " +
			"resource — only the definition itself. The identity fields `object_type`, `key`, and " +
			"`type` are immutable; changing any of them forces replacement.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Unique identifier of the custom field definition.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"object_type": schema.StringAttribute{
				MarkdownDescription: "Object type this field is attached to (e.g. `Tournament`, `Account`, " +
					"`Team`, `Organization`). See the `ObjectType` enum in the WellPlayed schema for the " +
					"full list. Immutable.",
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"key": schema.StringAttribute{
				MarkdownDescription: "Unique key identifier for this field within the object type. Used " +
					"as the stable lookup name when reading or writing values. Immutable.",
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Human-readable display name of the field.",
				Required:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "Optional description explaining the purpose of this field.",
				Optional:            true,
			},
			"type": schema.StringAttribute{
				MarkdownDescription: "Data type of the field value. One of `STRING`, `NUMBER`, `BOOLEAN`, " +
					"`DATE`, `EMAIL`, `PHONE`, `URL`, `COUNTRY`, `SELECT`, `MULTI_SELECT`, `IMAGE`. " +
					"Immutable.",
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"required": schema.BoolAttribute{
				MarkdownDescription: "Whether this field must be filled in when creating/updating the " +
					"parent object. Defaults to `false` on the server when omitted.",
				Optional: true,
				Computed: true,
			},
			"unique": schema.BoolAttribute{
				MarkdownDescription: "Whether the value must be unique across all objects of this type. " +
					"Defaults to `false` on the server when omitted.",
				Optional: true,
				Computed: true,
			},
			"order": schema.Int64Attribute{
				MarkdownDescription: "Display order position of this field. Assigned by the server when " +
					"omitted.",
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
			},
			"visibility": schema.StringAttribute{
				MarkdownDescription: "Visibility level controlling who can view this field value. One of " +
					"`PUBLIC`, `OWNER`, `OWNER_OR_PERMISSION`, `WITH_PERMISSION`.",
				Required: true,
			},
			"editability": schema.StringAttribute{
				MarkdownDescription: "Editability rule controlling when and by whom this field can be " +
					"modified. One of `ALWAYS`, `ONE_TIME`, `WITH_PERMISSION`.",
				Required: true,
			},
			"options": schema.ListNestedAttribute{
				MarkdownDescription: "Available options for `SELECT` or `MULTI_SELECT` fields. Ignored " +
					"for other types.",
				Optional: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"label": schema.StringAttribute{
							MarkdownDescription: "Human-readable label shown to the user.",
							Required:            true,
						},
						"value": schema.StringAttribute{
							MarkdownDescription: "Stable value persisted when the option is selected.",
							Required:            true,
						},
					},
				},
			},
			"default_value": schema.StringAttribute{
				MarkdownDescription: "Default value applied when no value is provided, serialized as a " +
					"string.",
				Optional: true,
			},
			"validation_regex": schema.StringAttribute{
				MarkdownDescription: "Optional regex pattern the value must match.",
				Optional:            true,
			},
			"created_at": schema.StringAttribute{
				MarkdownDescription: "When the definition was created (RFC 3339).",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"updated_at": schema.StringAttribute{
				MarkdownDescription: "When the definition was last updated (RFC 3339).",
				Computed:            true,
			},
		},
	}
}

func (r *customFieldDefinitionResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *customFieldDefinitionResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan customFieldDefinitionModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	input := buildCreateCustomFieldDefinitionInput(&plan)

	query := `mutation($input: CreateCustomFieldDefinitionInput!) { createCustomFieldDefinition(input: $input) { ` + customFieldDefinitionFields + ` } }`
	var out struct {
		CreateCustomFieldDefinition gqlCustomFieldDefinition `json:"createCustomFieldDefinition"`
	}
	if err := r.client.Execute(ctx, query, map[string]any{"input": input}, &out); err != nil {
		resp.Diagnostics.AddError("Unable to create custom field definition", err.Error())
		return
	}

	applyCustomFieldDefinitionToModel(&out.CreateCustomFieldDefinition, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *customFieldDefinitionResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state customFieldDefinitionModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// The schema exposes no single-record lookup by id, so list-and-filter on
	// the (objectType, id) pair held in state.
	query := `query($objectType: ObjectType!) { customFieldDefinitions(objectType: $objectType) { ` + customFieldDefinitionFields + ` } }`
	var out struct {
		CustomFieldDefinitions []gqlCustomFieldDefinition `json:"customFieldDefinitions"`
	}
	vars := map[string]any{"objectType": state.ObjectType.ValueString()}
	if err := r.client.Execute(ctx, query, vars, &out); err != nil {
		resp.Diagnostics.AddError("Unable to read custom field definitions", err.Error())
		return
	}

	id := state.ID.ValueString()
	var found *gqlCustomFieldDefinition
	for i := range out.CustomFieldDefinitions {
		if out.CustomFieldDefinitions[i].ID == id {
			found = &out.CustomFieldDefinitions[i]
			break
		}
	}
	if found == nil {
		tflog.Warn(ctx, "Custom field definition not found, removing from state", map[string]any{
			"id":          id,
			"object_type": state.ObjectType.ValueString(),
		})
		resp.State.RemoveResource(ctx)
		return
	}

	applyCustomFieldDefinitionToModel(found, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *customFieldDefinitionResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state customFieldDefinitionModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	input := buildUpdateCustomFieldDefinitionInput(&plan)

	query := `mutation($id: ID!, $input: UpdateCustomFieldDefinitionInput!) { updateCustomFieldDefinition(id: $id, input: $input) { ` + customFieldDefinitionFields + ` } }`
	vars := map[string]any{"id": state.ID.ValueString(), "input": input}
	var out struct {
		UpdateCustomFieldDefinition gqlCustomFieldDefinition `json:"updateCustomFieldDefinition"`
	}
	if err := r.client.Execute(ctx, query, vars, &out); err != nil {
		resp.Diagnostics.AddError("Unable to update custom field definition", err.Error())
		return
	}

	applyCustomFieldDefinitionToModel(&out.UpdateCustomFieldDefinition, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *customFieldDefinitionResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state customFieldDefinitionModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	query := `mutation($id: ID!) { deleteCustomFieldDefinition(id: $id) }`
	if err := r.client.Execute(ctx, query, map[string]any{"id": state.ID.ValueString()}, nil); err != nil {
		resp.Diagnostics.AddError("Unable to delete custom field definition", err.Error())
		return
	}
}

// ImportState accepts a composite "<object_type>:<id>" because Read needs the
// object type to query the schema's list-only endpoint.
func (r *customFieldDefinitionResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, ":", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		resp.Diagnostics.AddError(
			"Invalid import id",
			"Expected '<object_type>:<id>' (e.g. 'Tournament:cfd_01h...'). Got: "+req.ID,
		)
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("object_type"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[1])...)
}
