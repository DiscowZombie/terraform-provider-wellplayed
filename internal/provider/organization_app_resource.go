// Copyright (c) Mathéo Cimbaro
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/DiscowZombie/terraform-provider-wellplayed/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure the resource satisfies the framework interfaces.
var (
	_ resource.Resource                = &organizationAppResource{}
	_ resource.ResourceWithConfigure   = &organizationAppResource{}
	_ resource.ResourceWithImportState = &organizationAppResource{}
)

// NewOrganizationAppResource constructs the wellplayed_organization_app resource.
func NewOrganizationAppResource() resource.Resource {
	return &organizationAppResource{}
}

type organizationAppResource struct {
	client *client.Client
}

// organizationAppModel maps the resource schema to Go values. All authored
// fields round-trip through Read via the app's `configuration`; only `secret`
// is write-only (returned solely on creation), so it is preserved from prior
// state rather than refreshed.
type organizationAppModel struct {
	ID                 types.String `tfsdk:"id"`
	Name               types.String `tfsdk:"name"`
	Description        types.String `tfsdk:"description"`
	Icon               types.String `tfsdk:"icon"`
	ShortDescription   types.String `tfsdk:"short_description"`
	Public             types.Bool   `tfsdk:"public"`
	RedirectURLs       types.List   `tfsdk:"redirect_urls"`
	LogoutRedirectURLs types.List   `tfsdk:"logout_redirect_urls"`
	LoginURL           types.String `tfsdk:"login_url"`
	ConsentURL         types.String `tfsdk:"consent_url"`
	RequiresConsent    types.Bool   `tfsdk:"requires_consent"`
	Secret             types.String `tfsdk:"secret"`
	CreatorID          types.String `tfsdk:"creator_id"`
	OrganizationID     types.String `tfsdk:"organization_id"`
	CreatedAt          types.String `tfsdk:"created_at"`
	UpdatedAt          types.String `tfsdk:"updated_at"`
}

func (r *organizationAppResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_organization_app"
}

func (r *organizationAppResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a WellPlayed organization OAuth2 application. Backed by the " +
			"`createOrganizationApp`, `updateOrganizationApp`, and `deleteOrganizationApp` mutations.\n\n" +
			"The `secret` is the OAuth2 client secret. The API returns it only when the app is created (or " +
			"when the secret is reset out of band), so it is stored once at create time and never refreshed " +
			"during Read; it is unavailable after import.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Unique identifier of the app, which is also the OAuth2 client ID.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Display name of the app.",
				Required:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "Description of the app.",
				Required:            true,
			},
			"icon": schema.StringAttribute{
				MarkdownDescription: "App icon URL.",
				Optional:            true,
			},
			"short_description": schema.StringAttribute{
				MarkdownDescription: "Short description of the app.",
				Optional:            true,
			},
			"public": schema.BoolAttribute{
				MarkdownDescription: "Whether this is a public OAuth2 client (no client secret required). " +
					"Defaults to the server's choice when omitted. Changing it forces a new app.",
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.Bool{boolplanmodifier.RequiresReplace()},
			},
			"redirect_urls": schema.ListAttribute{
				MarkdownDescription: "Allowed OAuth2 redirect URLs after authorization.",
				Required:            true,
				ElementType:         types.StringType,
			},
			"logout_redirect_urls": schema.ListAttribute{
				MarkdownDescription: "Allowed redirect URLs after logout.",
				Required:            true,
				ElementType:         types.StringType,
			},
			"login_url": schema.StringAttribute{
				MarkdownDescription: "URL where users are redirected to log in.",
				Required:            true,
			},
			"consent_url": schema.StringAttribute{
				MarkdownDescription: "URL where users are redirected to grant consent.",
				Required:            true,
			},
			"requires_consent": schema.BoolAttribute{
				MarkdownDescription: "Whether the app requires explicit user consent during authorization.",
				Required:            true,
			},
			"secret": schema.StringAttribute{
				MarkdownDescription: "OAuth2 client secret. Returned only when the app is created; not " +
					"refreshed during Read and unavailable after import.",
				Computed:      true,
				Sensitive:     true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"creator_id": schema.StringAttribute{
				MarkdownDescription: "ID of the account that created this app.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"organization_id": schema.StringAttribute{
				MarkdownDescription: "ID of the organization that owns this app.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"created_at": schema.StringAttribute{
				MarkdownDescription: "When the app was created (RFC 3339).",
				Computed:            true,
			},
			"updated_at": schema.StringAttribute{
				MarkdownDescription: "When the app was last updated (RFC 3339).",
				Computed:            true,
			},
		},
	}
}

func (r *organizationAppResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *organizationAppResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan organizationAppModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	input, diags := buildOrganizationAppInput(ctx, &plan, true)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	query := `mutation($input: CreateOrganizationAppInput!) { createOrganizationApp(input: $input) { ` + organizationAppFields + ` } }`
	var out struct {
		CreateOrganizationApp gqlOrganizationApp `json:"createOrganizationApp"`
	}
	if err := r.client.Execute(ctx, query, map[string]any{"input": input}, &out); err != nil {
		resp.Diagnostics.AddError("Unable to create organization app", err.Error())
		return
	}

	applyOrganizationAppComputed(&out.CreateOrganizationApp, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *organizationAppResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state organizationAppModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	query := `query($id: String!) { organizationApp(id: $id) { ` + organizationAppFields + ` } }`
	var out struct {
		OrganizationApp *gqlOrganizationApp `json:"organizationApp"`
	}
	if err := r.client.Execute(ctx, query, map[string]any{"id": state.ID.ValueString()}, &out); err != nil {
		if isOrganizationAppNotFoundErr(err) {
			tflog.Warn(ctx, "Organization app not found, removing from state", map[string]any{"id": state.ID.ValueString()})
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Unable to read organization app", err.Error())
		return
	}
	if out.OrganizationApp == nil {
		tflog.Warn(ctx, "Organization app not found, removing from state", map[string]any{"id": state.ID.ValueString()})
		resp.State.RemoveResource(ctx)
		return
	}

	// secret is returned only on creation; leave the prior value in place.
	resp.Diagnostics.Append(applyOrganizationAppRead(out.OrganizationApp, &state)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *organizationAppResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state organizationAppModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// public is create-only (RequiresReplace) and absent from the update input.
	input, diags := buildOrganizationAppInput(ctx, &plan, false)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	query := `mutation($id: String!, $input: UpdateOrganizationAppInput!) { updateOrganizationApp(id: $id, input: $input) { ` + organizationAppFields + ` } }`
	vars := map[string]any{"id": state.ID.ValueString(), "input": input}
	var out struct {
		UpdateOrganizationApp gqlOrganizationApp `json:"updateOrganizationApp"`
	}
	if err := r.client.Execute(ctx, query, vars, &out); err != nil {
		resp.Diagnostics.AddError("Unable to update organization app", err.Error())
		return
	}

	applyOrganizationAppComputed(&out.UpdateOrganizationApp, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *organizationAppResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state organizationAppModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	query := `mutation($id: String!) { deleteOrganizationApp(id: $id) }`
	if err := r.client.Execute(ctx, query, map[string]any{"id": state.ID.ValueString()}, nil); err != nil {
		resp.Diagnostics.AddError("Unable to delete organization app", err.Error())
		return
	}
}

func (r *organizationAppResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// isOrganizationAppNotFoundErr reports whether a GraphQL error indicates the
// app is gone.
func isOrganizationAppNotFoundErr(err error) bool {
	s := strings.ToLower(err.Error())
	return strings.Contains(s, "not found") ||
		strings.Contains(s, "does not exist") ||
		strings.Contains(s, "no organization app")
}
