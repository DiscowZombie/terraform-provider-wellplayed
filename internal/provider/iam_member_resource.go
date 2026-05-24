package provider

import (
	"context"
	"fmt"

	"github.com/DiscowZombie/terraform-provider-wellplayed/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/diag"
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
	_ resource.Resource                   = &iamMemberResource{}
	_ resource.ResourceWithConfigure      = &iamMemberResource{}
	_ resource.ResourceWithImportState    = &iamMemberResource{}
	_ resource.ResourceWithValidateConfig = &iamMemberResource{}
)

// NewIAMMemberResource constructs the wellplayed_iam_member resource.
func NewIAMMemberResource() resource.Resource {
	return &iamMemberResource{}
}

type iamMemberResource struct {
	client *client.Client
}

// iamMemberModel maps the resource schema to Go values.
type iamMemberModel struct {
	ID             types.String `tfsdk:"id"`
	MemberID       types.String `tfsdk:"member_id"`
	OrganizationID types.String `tfsdk:"organization_id"`
	UserID         types.String `tfsdk:"user_id"`
	Email          types.String `tfsdk:"email"`
	GroupID        types.String `tfsdk:"group_id"`
	Permissions    types.Set    `tfsdk:"permissions"`
	CreatedAt      types.String `tfsdk:"created_at"`
	UpdatedAt      types.String `tfsdk:"updated_at"`
}

func (r *iamMemberResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_iam_member"
}

func (r *iamMemberResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages an organization (IAM) membership: assigns an account to a permission group " +
			"and grants member-specific permissions. Backed by the `setOrganizationMembership` and " +
			"`deleteOrganizationMembership` mutations.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Unique identifier of the membership.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"member_id": schema.StringAttribute{
				MarkdownDescription: "Account ID of the member. This is the canonical identifier used for import.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"organization_id": schema.StringAttribute{
				MarkdownDescription: "ID of the organization this membership belongs to.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"user_id": schema.StringAttribute{
				MarkdownDescription: "Account ID of the user to add. Mutually exclusive with `email`; " +
					"exactly one must be set. Changing it forces a new membership.",
				Optional:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"email": schema.StringAttribute{
				MarkdownDescription: "Email address of the user to add. Mutually exclusive with `user_id`; " +
					"exactly one must be set. Changing it forces a new membership.",
				Optional:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"group_id": schema.StringAttribute{
				MarkdownDescription: "ID of the permission group assigned to this member.",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"permissions": schema.SetNestedAttribute{
				MarkdownDescription: "Additional permissions granted to this member beyond the group. " +
					"Note: the API does not return permissions in a readable form, so this value is " +
					"managed write-only from configuration and is not refreshed during Read; " +
					"out-of-band changes to permissions are not detected as drift.",
				Optional: true,
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
			"created_at": schema.StringAttribute{
				MarkdownDescription: "When the membership was created (RFC 3339).",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"updated_at": schema.StringAttribute{
				MarkdownDescription: "When the membership was last updated (RFC 3339).",
				Computed:            true,
			},
		},
	}
}

func (r *iamMemberResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var data iamMemberModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	hasUserID := !data.UserID.IsNull() && !data.UserID.IsUnknown()
	hasEmail := !data.Email.IsNull() && !data.Email.IsUnknown()

	// Unknown values can't be validated yet; defer to apply time.
	if data.UserID.IsUnknown() || data.Email.IsUnknown() {
		return
	}

	switch {
	case hasUserID && hasEmail:
		resp.Diagnostics.AddError(
			"Conflicting member identity",
			"Only one of `user_id` or `email` may be set.",
		)
	case !hasUserID && !hasEmail:
		resp.Diagnostics.AddAttributeError(
			path.Root("user_id"),
			"Missing member identity",
			"Exactly one of `user_id` or `email` must be set.",
		)
	}
}

func (r *iamMemberResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// orgMember mirrors the readable subset of the OrganizationMember GraphQL
// type. `permissions` is deliberately excluded; see orgMemberFields.
type orgMember struct {
	ID             string `json:"id"`
	OrganizationID string `json:"organizationId"`
	GroupID        string `json:"groupId"`
	MemberID       string `json:"memberId"`
	CreatedAt      string `json:"createdAt"`
	UpdatedAt      string `json:"updatedAt"`
}

// orgMemberFields intentionally omits `permissions`: the API's resolver for
// OrganizationMember.permissions returns a non-iterable and errors when
// selected. Permissions are managed write-only from configuration instead.
const orgMemberFields = `id organizationId groupId memberId createdAt updatedAt`

func (r *iamMemberResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan iamMemberModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	input := map[string]any{}
	if !plan.UserID.IsNull() {
		input["userId"] = plan.UserID.ValueString()
	}
	if !plan.Email.IsNull() {
		input["email"] = plan.Email.ValueString()
	}
	if !plan.GroupID.IsNull() && !plan.GroupID.IsUnknown() {
		input["groupId"] = plan.GroupID.ValueString()
	}
	perms, diags := permissionsToInput(ctx, plan.Permissions)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	input["permissions"] = perms

	member, err := r.setMembership(ctx, input)
	if err != nil {
		resp.Diagnostics.AddError("Unable to create IAM member", err.Error())
		return
	}

	applyMemberToModel(member, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *iamMemberResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state iamMemberModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	member, err := r.findMember(ctx, state.MemberID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Unable to read IAM member", err.Error())
		return
	}
	if member == nil {
		tflog.Warn(ctx, "IAM member not found, removing from state", map[string]any{"member_id": state.MemberID.ValueString()})
		resp.State.RemoveResource(ctx)
		return
	}

	applyMemberToModel(member, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *iamMemberResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state iamMemberModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Identity is immutable (RequiresReplace), so address the member by its
	// account ID for the upsert.
	input := map[string]any{"userId": state.MemberID.ValueString()}
	if !plan.GroupID.IsNull() && !plan.GroupID.IsUnknown() {
		input["groupId"] = plan.GroupID.ValueString()
	}
	perms, diags := permissionsToInput(ctx, plan.Permissions)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	input["permissions"] = perms

	member, err := r.setMembership(ctx, input)
	if err != nil {
		resp.Diagnostics.AddError("Unable to update IAM member", err.Error())
		return
	}

	applyMemberToModel(member, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *iamMemberResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state iamMemberModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	query := `mutation($input: DeleteOrganizationMembershipInput!) { deleteOrganizationMembership(input: $input) }`
	vars := map[string]any{"input": map[string]any{"userId": state.MemberID.ValueString()}}
	if err := r.client.Execute(ctx, query, vars, nil); err != nil {
		resp.Diagnostics.AddError("Unable to delete IAM member", err.Error())
		return
	}
}

func (r *iamMemberResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import using the member (account) ID; Read reconciles the rest.
	resource.ImportStatePassthroughID(ctx, path.Root("member_id"), req, resp)
}

// setMembership runs the upsert mutation and returns the resulting member.
func (r *iamMemberResource) setMembership(ctx context.Context, input map[string]any) (*orgMember, error) {
	query := `mutation($input: SetOrganizationMembershipInput!) { setOrganizationMembership(input: $input) { ` + orgMemberFields + ` } }`
	var out struct {
		SetOrganizationMembership orgMember `json:"setOrganizationMembership"`
	}
	if err := r.client.Execute(ctx, query, map[string]any{"input": input}, &out); err != nil {
		return nil, err
	}
	return &out.SetOrganizationMembership, nil
}

// findMember pages through organizationMembers looking for memberID.
// Returns nil (no error) when the member is absent.
func (r *iamMemberResource) findMember(ctx context.Context, memberID string) (*orgMember, error) {
	const pageSize = 100
	query := `query($page: PageInfo!) {
  organizationMembers(page: $page) {
    nodes { ` + orgMemberFields + ` }
    pageInfo { hasNextPage endCursor }
  }
}`

	var after *string
	for {
		page := map[string]any{"first": pageSize}
		if after != nil {
			page["after"] = *after
		}
		var out struct {
			OrganizationMembers struct {
				Nodes    []orgMember `json:"nodes"`
				PageInfo struct {
					HasNextPage bool   `json:"hasNextPage"`
					EndCursor   string `json:"endCursor"`
				} `json:"pageInfo"`
			} `json:"organizationMembers"`
		}
		if err := r.client.Execute(ctx, query, map[string]any{"page": page}, &out); err != nil {
			return nil, err
		}
		for i := range out.OrganizationMembers.Nodes {
			if out.OrganizationMembers.Nodes[i].MemberID == memberID {
				return &out.OrganizationMembers.Nodes[i], nil
			}
		}
		if !out.OrganizationMembers.PageInfo.HasNextPage || out.OrganizationMembers.PageInfo.EndCursor == "" {
			return nil, nil
		}
		cursor := out.OrganizationMembers.PageInfo.EndCursor
		after = &cursor
	}
}

// permissionsToInput converts the permissions set to GraphQL input. It always
// returns a non-nil slice (empty when the set is null/unknown): the API calls
// .reduce() on permissions unconditionally, so the field must always be sent.
func permissionsToInput(ctx context.Context, set types.Set) ([]map[string]any, diag.Diagnostics) {
	var diags diag.Diagnostics
	if set.IsNull() || set.IsUnknown() {
		return []map[string]any{}, diags
	}

	var elems []permissionElem
	diags = append(diags, set.ElementsAs(ctx, &elems, false)...)
	if diags.HasError() {
		return nil, diags
	}

	out := make([]map[string]any, 0, len(elems))
	for _, e := range elems {
		var resources []string
		diags = append(diags, e.Resources.ElementsAs(ctx, &resources, false)...)
		out = append(out, map[string]any{"id": e.ID.ValueString(), "resources": resources})
	}
	return out, diags
}

// permissionElem is the Go shape of a single permission for ElementsAs.
type permissionElem struct {
	ID        types.String `tfsdk:"id"`
	Resources types.Set    `tfsdk:"resources"`
}

// applyMemberToModel copies a GraphQL member onto the Terraform model. It
// deliberately leaves identity (user_id / email) and permissions untouched:
// those are managed from configuration and not returned readably by the API.
func applyMemberToModel(m *orgMember, model *iamMemberModel) {
	model.ID = types.StringValue(m.ID)
	model.MemberID = types.StringValue(m.MemberID)
	model.OrganizationID = types.StringValue(m.OrganizationID)
	model.GroupID = types.StringValue(m.GroupID)
	model.CreatedAt = types.StringValue(m.CreatedAt)
	model.UpdatedAt = types.StringValue(m.UpdatedAt)
}
