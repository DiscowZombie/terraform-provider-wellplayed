package provider

import (
	"context"
	"fmt"

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
	_ resource.Resource                = &organizationAttachedAppResource{}
	_ resource.ResourceWithConfigure   = &organizationAttachedAppResource{}
	_ resource.ResourceWithImportState = &organizationAttachedAppResource{}
)

// NewOrganizationAttachedAppResource constructs the
// wellplayed_organization_attached_app resource.
func NewOrganizationAttachedAppResource() resource.Resource {
	return &organizationAttachedAppResource{}
}

type organizationAttachedAppResource struct {
	client *client.Client
}

// organizationAttachedAppModel maps the resource schema to Go values. app_id is
// the immutable identity; group_id and permissions are the mutable assignment.
type organizationAttachedAppModel struct {
	ID             types.String `tfsdk:"id"`
	AppID          types.String `tfsdk:"app_id"`
	GroupID        types.String `tfsdk:"group_id"`
	Permissions    types.Set    `tfsdk:"permissions"`
	OrganizationID types.String `tfsdk:"organization_id"`
	CreatedAt      types.String `tfsdk:"created_at"`
	UpdatedAt      types.String `tfsdk:"updated_at"`
}

func (r *organizationAttachedAppResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_organization_attached_app"
}

func (r *organizationAttachedAppResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Attaches an application to the organization and assigns it a permission group " +
			"(and optional member-specific permissions). This is the app-side counterpart of " +
			"`wellplayed_iam_member`: it places an app — rather than a user — into an IAM group. Backed by " +
			"the `setOrganizationAttachedApp` and `deleteOrganizationAttachedApp` mutations and the " +
			"`organizationAttachedApps` query.\n\n" +
			"Each resource owns exactly one `app_id`; that field is immutable and changing it forces " +
			"replacement.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Identifier of the attachment. Equal to `app_id`.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"app_id": schema.StringAttribute{
				MarkdownDescription: "Identifier of the application to attach. Changing it forces a new " +
					"attachment. This is the canonical identifier used for import.",
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"group_id": schema.StringAttribute{
				MarkdownDescription: "ID of the permission group assigned to this attached app (see " +
					"`wellplayed_iam_group`). Omit to attach the app without a group.",
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"permissions": schema.SetNestedAttribute{
				MarkdownDescription: "Permissions granted to this attached app beyond its group. Omit for " +
					"none. Note: the API does not return permissions in a readable form, so this value is " +
					"managed write-only from configuration and is not refreshed during Read; out-of-band " +
					"changes to permissions are not detected as drift.",
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
			"organization_id": schema.StringAttribute{
				MarkdownDescription: "ID of the organization this app is attached to.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"created_at": schema.StringAttribute{
				MarkdownDescription: "When the attachment was created (RFC 3339).",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"updated_at": schema.StringAttribute{
				MarkdownDescription: "When the attachment was last updated (RFC 3339).",
				Computed:            true,
			},
		},
	}
}

func (r *organizationAttachedAppResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *organizationAttachedAppResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan organizationAttachedAppModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	app, err := r.setAttachedApp(ctx, &plan)
	if err != nil {
		resp.Diagnostics.AddError("Unable to attach app", err.Error())
		return
	}

	applyAttachedAppToModel(app, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *organizationAttachedAppResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state organizationAttachedAppModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	app, err := r.findAttachedApp(ctx, state.AppID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Unable to read attached app", err.Error())
		return
	}
	if app == nil {
		tflog.Warn(ctx, "Attached app not found, removing from state", map[string]any{"app_id": state.AppID.ValueString()})
		resp.State.RemoveResource(ctx)
		return
	}

	applyAttachedAppToModel(app, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *organizationAttachedAppResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan organizationAttachedAppModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	app, err := r.setAttachedApp(ctx, &plan)
	if err != nil {
		resp.Diagnostics.AddError("Unable to update attached app", err.Error())
		return
	}

	applyAttachedAppToModel(app, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *organizationAttachedAppResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state organizationAttachedAppModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	query := `mutation($input: DeleteOrganizationAttachedAppInput!) { deleteOrganizationAttachedApp(input: $input) }`
	vars := map[string]any{"input": map[string]any{"appId": state.AppID.ValueString()}}
	if err := r.client.Execute(ctx, query, vars, nil); err != nil {
		resp.Diagnostics.AddError("Unable to detach app", err.Error())
		return
	}
}

func (r *organizationAttachedAppResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import using the app ID; Read reconciles the rest.
	resource.ImportStatePassthroughID(ctx, path.Root("app_id"), req, resp)
}

// setAttachedApp runs the upsert mutation and returns the resulting attachment.
func (r *organizationAttachedAppResource) setAttachedApp(ctx context.Context, m *organizationAttachedAppModel) (*gqlOrganizationAttachedApp, error) {
	input, diags := buildSetAttachedAppInput(ctx, m)
	if diags.HasError() {
		return nil, fmt.Errorf("invalid attached app input: %v", diags.Errors())
	}

	query := `mutation($input: SetOrganizationAttachedAppInput!) { setOrganizationAttachedApp(input: $input) { ` + organizationAttachedAppFields + ` } }`
	var out struct {
		SetOrganizationAttachedApp gqlOrganizationAttachedApp `json:"setOrganizationAttachedApp"`
	}
	if err := r.client.Execute(ctx, query, map[string]any{"input": input}, &out); err != nil {
		return nil, err
	}
	return &out.SetOrganizationAttachedApp, nil
}

// findAttachedApp pages through organizationAttachedApps looking for appID.
// Returns nil (no error) when the attachment is absent.
func (r *organizationAttachedAppResource) findAttachedApp(ctx context.Context, appID string) (*gqlOrganizationAttachedApp, error) {
	const pageSize = 100
	query := `query($page: PageInfo!) {
  organizationAttachedApps(page: $page) {
    nodes { ` + organizationAttachedAppFields + ` }
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
			OrganizationAttachedApps struct {
				Nodes    []gqlOrganizationAttachedApp `json:"nodes"`
				PageInfo struct {
					HasNextPage bool   `json:"hasNextPage"`
					EndCursor   string `json:"endCursor"`
				} `json:"pageInfo"`
			} `json:"organizationAttachedApps"`
		}
		if err := r.client.Execute(ctx, query, map[string]any{"page": page}, &out); err != nil {
			return nil, err
		}
		for i := range out.OrganizationAttachedApps.Nodes {
			if out.OrganizationAttachedApps.Nodes[i].AppID == appID {
				return &out.OrganizationAttachedApps.Nodes[i], nil
			}
		}
		if !out.OrganizationAttachedApps.PageInfo.HasNextPage || out.OrganizationAttachedApps.PageInfo.EndCursor == "" {
			return nil, nil
		}
		cursor := out.OrganizationAttachedApps.PageInfo.EndCursor
		after = &cursor
	}
}
