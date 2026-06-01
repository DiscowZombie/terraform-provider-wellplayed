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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure the resource satisfies the framework interfaces.
var (
	_ resource.Resource                = &iamGroupResource{}
	_ resource.ResourceWithConfigure   = &iamGroupResource{}
	_ resource.ResourceWithImportState = &iamGroupResource{}
)

// NewIAMGroupResource constructs the wellplayed_iam_group resource.
func NewIAMGroupResource() resource.Resource {
	return &iamGroupResource{}
}

type iamGroupResource struct {
	client *client.Client
}

// iamGroupModel maps the resource schema to Go values.
type iamGroupModel struct {
	ID             types.String `tfsdk:"id"`
	Name           types.String `tfsdk:"name"`
	Permissions    types.Set    `tfsdk:"permissions"`
	Type           types.String `tfsdk:"type"`
	OrganizationID types.String `tfsdk:"organization_id"`
	CreatedAt      types.String `tfsdk:"created_at"`
	UpdatedAt      types.String `tfsdk:"updated_at"`
}

func (r *iamGroupResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_iam_group"
}

func (r *iamGroupResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages an organization permission group: a named set of permissions that members " +
			"and apps can be assigned to (see `wellplayed_iam_member`'s `group_id`). Backed by the " +
			"`createOrganizationGroup`, `updateOrganizationGroup`, and `deleteOrganizationGroup` mutations.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Unique identifier of the group.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Display name of the group. Unique within the organization.",
				Required:            true,
			},
			"permissions": schema.SetNestedAttribute{
				MarkdownDescription: "Permissions granted to members of this group. Omit for a group with no " +
					"permissions. Unlike `wellplayed_iam_member`, group permissions are returned by the API, " +
					"so out-of-band changes are detected as drift.",
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.Set{setplanmodifier.UseStateForUnknown()},
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							MarkdownDescription: "The permission identifier.",
							Required:            true,
						},
						"resources": schema.SetAttribute{
							MarkdownDescription: "Resource identifiers this permission is scoped to.",
							Required:            true,
							ElementType:         types.StringType,
						},
					},
				},
			},
			"type": schema.StringAttribute{
				MarkdownDescription: "The type of group, one of `ANONYMOUS`, `CONNECTED`, `MEMBER`, `ADMIN`. " +
					"Assigned by the API.",
				Computed: true,
			},
			"organization_id": schema.StringAttribute{
				MarkdownDescription: "ID of the organization this group belongs to.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"created_at": schema.StringAttribute{
				MarkdownDescription: "When the group was created (RFC 3339).",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"updated_at": schema.StringAttribute{
				MarkdownDescription: "When the group was last updated (RFC 3339).",
				Computed:            true,
			},
		},
	}
}

func (r *iamGroupResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *iamGroupResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan iamGroupModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	input, diags := buildOrganizationGroupInput(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	query := `mutation($input: CreateOrganizationGroupInput!) { createOrganizationGroup(input: $input) { ` + organizationGroupFields + ` } }`
	var out struct {
		CreateOrganizationGroup gqlOrganizationGroup `json:"createOrganizationGroup"`
	}
	if err := r.client.Execute(ctx, query, map[string]any{"input": input}, &out); err != nil {
		resp.Diagnostics.AddError("Unable to create IAM group", err.Error())
		return
	}

	resp.Diagnostics.Append(applyGroupToModel(ctx, &out.CreateOrganizationGroup, &plan)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *iamGroupResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state iamGroupModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	query := `query($id: ID!) { organizationGroup(id: $id) { ` + organizationGroupFields + ` } }`
	var out struct {
		OrganizationGroup *gqlOrganizationGroup `json:"organizationGroup"`
	}
	if err := r.client.Execute(ctx, query, map[string]any{"id": state.ID.ValueString()}, &out); err != nil {
		if isGroupNotFoundErr(err) {
			tflog.Warn(ctx, "IAM group not found, removing from state", map[string]any{"id": state.ID.ValueString()})
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Unable to read IAM group", err.Error())
		return
	}
	if out.OrganizationGroup == nil {
		tflog.Warn(ctx, "IAM group not found, removing from state", map[string]any{"id": state.ID.ValueString()})
		resp.State.RemoveResource(ctx)
		return
	}

	resp.Diagnostics.Append(applyGroupToModel(ctx, out.OrganizationGroup, &state)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *iamGroupResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state iamGroupModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	input, diags := buildOrganizationGroupInput(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	query := `mutation($id: ID!, $input: CreateOrganizationGroupInput!) { updateOrganizationGroup(id: $id, input: $input) { ` + organizationGroupFields + ` } }`
	vars := map[string]any{"id": state.ID.ValueString(), "input": input}
	var out struct {
		UpdateOrganizationGroup gqlOrganizationGroup `json:"updateOrganizationGroup"`
	}
	if err := r.client.Execute(ctx, query, vars, &out); err != nil {
		resp.Diagnostics.AddError("Unable to update IAM group", err.Error())
		return
	}

	resp.Diagnostics.Append(applyGroupToModel(ctx, &out.UpdateOrganizationGroup, &plan)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *iamGroupResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state iamGroupModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	query := `mutation($id: ID!) { deleteOrganizationGroup(id: $id) }`
	if err := r.client.Execute(ctx, query, map[string]any{"id": state.ID.ValueString()}, nil); err != nil {
		resp.Diagnostics.AddError("Unable to delete IAM group", err.Error())
		return
	}
}

func (r *iamGroupResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// isGroupNotFoundErr reports whether a GraphQL error indicates the group is
// gone. organizationGroup returns a non-null type, so a missing group surfaces
// as an error rather than a null result.
func isGroupNotFoundErr(err error) bool {
	s := strings.ToLower(err.Error())
	return strings.Contains(s, "not found") ||
		strings.Contains(s, "does not exist") ||
		strings.Contains(s, "no group")
}
