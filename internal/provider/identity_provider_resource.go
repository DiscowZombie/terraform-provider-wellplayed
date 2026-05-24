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
	_ resource.Resource                   = &identityProviderResource{}
	_ resource.ResourceWithConfigure      = &identityProviderResource{}
	_ resource.ResourceWithImportState    = &identityProviderResource{}
	_ resource.ResourceWithValidateConfig = &identityProviderResource{}
)

// NewIdentityProviderResource constructs the wellplayed_identity_provider resource.
func NewIdentityProviderResource() resource.Resource {
	return &identityProviderResource{}
}

type identityProviderResource struct {
	client *client.Client
}

// identityProviderModel maps the resource schema to Go values. The OAuth2 and
// OpenID configuration blocks are managed write-only (see Read).
type identityProviderModel struct {
	ID                          types.String      `tfsdk:"id"`
	Name                        types.String      `tfsdk:"name"`
	Description                 types.String      `tfsdk:"description"`
	Icon                        types.String      `tfsdk:"icon"`
	Enabled                     types.Bool        `tfsdk:"enabled"`
	RequiredForPlayerValidation types.Bool        `tfsdk:"required_for_player_validation"`
	AllowLogin                  types.Bool        `tfsdk:"allow_login"`
	IdentityProviderID          types.String      `tfsdk:"identity_provider_id"`
	OrganizationID              types.String      `tfsdk:"organization_id"`
	CreatedAt                   types.String      `tfsdk:"created_at"`
	UpdatedAt                   types.String      `tfsdk:"updated_at"`
	OAuth2Configuration         *oauthConfigModel `tfsdk:"oauth2_configuration"`
	OpenIDConfiguration         *oauthConfigModel `tfsdk:"openid_configuration"`
}

// oauthConfigModel covers both oauth2_configuration and openid_configuration.
// The two GraphQL input types share the same field set; OpenID simply ignores
// the OAuth2-only fields (token_endpoint, authorization_url, link_redirect_url,
// token_endpoint_auth_method), which the builder skips when they are null.
type oauthConfigModel struct {
	ProviderType            types.String         `tfsdk:"provider_type"`
	ClientID                types.String         `tfsdk:"client_id"`
	ClientSecret            types.String         `tfsdk:"client_secret"`
	RedirectURL             types.String         `tfsdk:"redirect_url"`
	Issuer                  types.String         `tfsdk:"issuer"`
	AuthorizationEndpoint   types.String         `tfsdk:"authorization_endpoint"`
	TokenEndpoint           types.String         `tfsdk:"token_endpoint"`
	AuthorizationURL        types.String         `tfsdk:"authorization_url"`
	LinkRedirectURL         types.String         `tfsdk:"link_redirect_url"`
	TokenEndpointAuthMethod types.String         `tfsdk:"token_endpoint_auth_method"`
	DataRetrievers          []dataRetrieverModel `tfsdk:"data_retrievers"`
}

type dataRetrieverModel struct {
	URL      types.String   `tfsdk:"url"`
	Headers  []headerModel  `tfsdk:"headers"`
	Mappings []mappingModel `tfsdk:"mappings"`
}

type headerModel struct {
	Name  types.String `tfsdk:"name"`
	Value types.String `tfsdk:"value"`
}

type mappingModel struct {
	Path     types.String `tfsdk:"path"`
	MappedTo types.String `tfsdk:"mapped_to"`
	Private  types.Bool   `tfsdk:"private"`
}

func (r *identityProviderResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_identity_provider"
}

func (r *identityProviderResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a WellPlayed organization identity provider (IdP) for OAuth2 or OpenID Connect " +
			"authentication. Backed by the `createIdentityProvider`, `updateIdentityProvider`, and " +
			"`deleteIdentityProvider` mutations.\n\n" +
			"Provide at most one of `oauth2_configuration` or `openid_configuration`. A provider can also be " +
			"derived from a root identity provider via `identity_provider_id` instead of an inline configuration.\n\n" +
			"The configuration blocks carry the `client_secret` and are returned by the API only as an opaque " +
			"union, so they are managed write-only from configuration: sent on create/update but not refreshed " +
			"during Read, and out-of-band changes are not detected as drift.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Unique identifier of the identity provider.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Display name of the identity provider.",
				Required:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "Human-readable description of the identity provider.",
				Required:            true,
			},
			"icon": schema.StringAttribute{
				MarkdownDescription: "URL or identifier for the provider icon.",
				Optional:            true,
			},
			"enabled": schema.BoolAttribute{
				MarkdownDescription: "Whether this identity provider is currently active.",
				Required:            true,
			},
			"required_for_player_validation": schema.BoolAttribute{
				MarkdownDescription: "Whether player accounts must be validated through this provider.",
				Required:            true,
			},
			"allow_login": schema.BoolAttribute{
				MarkdownDescription: "Whether users can log in using this identity provider.",
				Required:            true,
			},
			"identity_provider_id": schema.StringAttribute{
				MarkdownDescription: "ID of the root identity provider this provider is derived from, if any. " +
					"Changing it forces a new provider.",
				Optional:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"organization_id": schema.StringAttribute{
				MarkdownDescription: "ID of the organization this provider belongs to.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"created_at": schema.StringAttribute{
				MarkdownDescription: "When the identity provider was created (RFC 3339).",
				Computed:            true,
			},
			"updated_at": schema.StringAttribute{
				MarkdownDescription: "When the identity provider was last updated (RFC 3339).",
				Computed:            true,
			},
			"oauth2_configuration": oauthConfigSchema("OAuth2 client configuration. Mutually exclusive with " +
				"`openid_configuration`."),
			"openid_configuration": oauthConfigSchema("OpenID Connect configuration. Mutually exclusive with " +
				"`oauth2_configuration`. The OAuth2-only fields (`token_endpoint`, `authorization_url`, " +
				"`link_redirect_url`, `token_endpoint_auth_method`) are ignored here."),
		},
	}
}

func oauthConfigSchema(description string) schema.SingleNestedAttribute {
	return schema.SingleNestedAttribute{
		MarkdownDescription: description,
		Optional:            true,
		Attributes: map[string]schema.Attribute{
			"provider_type": schema.StringAttribute{
				MarkdownDescription: "Identity provider protocol. One of `OPENID`, `OAUTH2`.",
				Required:            true,
			},
			"client_id": schema.StringAttribute{
				MarkdownDescription: "OAuth2 client ID issued by the identity provider.",
				Required:            true,
			},
			"client_secret": schema.StringAttribute{
				MarkdownDescription: "OAuth2 client secret issued by the identity provider.",
				Required:            true,
				Sensitive:           true,
			},
			"redirect_url": schema.StringAttribute{
				MarkdownDescription: "URL to redirect users back to after authentication.",
				Required:            true,
			},
			"issuer": schema.StringAttribute{
				MarkdownDescription: "Issuer identifier for the identity provider.",
				Optional:            true,
			},
			"authorization_endpoint": schema.StringAttribute{
				MarkdownDescription: "OAuth2/OIDC authorization endpoint URL.",
				Optional:            true,
			},
			"token_endpoint": schema.StringAttribute{
				MarkdownDescription: "URL of the OAuth2 token endpoint for exchanging authorization codes. " +
					"OAuth2 only.",
				Optional: true,
			},
			"authorization_url": schema.StringAttribute{
				MarkdownDescription: "URL of the OAuth2 authorization endpoint where users are redirected to " +
					"authenticate. OAuth2 only.",
				Optional: true,
			},
			"link_redirect_url": schema.StringAttribute{
				MarkdownDescription: "URL users are redirected to after the identity linking process completes. " +
					"Required for the IdP-based identity linking flow via `generateIdentityLinkUrl`. OAuth2 only.",
				Optional: true,
			},
			"token_endpoint_auth_method": schema.StringAttribute{
				MarkdownDescription: "Authentication method used when calling the token endpoint. One of " +
					"`CLIENT_SECRET_POST`, `CLIENT_SECRET_BASIC`, `CLIENT_SECRET_JWT`, `PRIVATE_KEY_JWT`, " +
					"`TLS_CLIENT_AUTH`, `SELF_SIGNED_TLS_CLIENT_AUTH`, `NONE`. OAuth2 only.",
				Optional: true,
			},
			"data_retrievers": dataRetrieversSchema(),
		},
	}
}

func dataRetrieversSchema() schema.ListNestedAttribute {
	return schema.ListNestedAttribute{
		MarkdownDescription: "Endpoints to retrieve user data from after authentication.",
		Optional:            true,
		NestedObject: schema.NestedAttributeObject{
			Attributes: map[string]schema.Attribute{
				"url": schema.StringAttribute{
					MarkdownDescription: "URL of the external endpoint to fetch user data from.",
					Required:            true,
				},
				"headers": schema.ListNestedAttribute{
					MarkdownDescription: "HTTP headers to include in the data retrieval request.",
					Optional:            true,
					NestedObject: schema.NestedAttributeObject{
						Attributes: map[string]schema.Attribute{
							"name": schema.StringAttribute{
								MarkdownDescription: "Name of the HTTP header.",
								Required:            true,
							},
							"value": schema.StringAttribute{
								MarkdownDescription: "Value of the HTTP header.",
								Required:            true,
								Sensitive:           true,
							},
						},
					},
				},
				"mappings": schema.ListNestedAttribute{
					MarkdownDescription: "Key mappings from retrieved data fields to player profile properties.",
					Optional:            true,
					NestedObject: schema.NestedAttributeObject{
						Attributes: map[string]schema.Attribute{
							"path": schema.StringAttribute{
								MarkdownDescription: "Dot-notation path to the source value in the retrieved object.",
								Required:            true,
							},
							"mapped_to": schema.StringAttribute{
								MarkdownDescription: "Target key name to map the value to.",
								Required:            true,
							},
							"private": schema.BoolAttribute{
								MarkdownDescription: "Whether the mapped value should be treated as private.",
								Optional:            true,
							},
						},
					},
				},
			},
		},
	}
}

func (r *identityProviderResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var data identityProviderModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if data.OAuth2Configuration != nil && data.OpenIDConfiguration != nil {
		resp.Diagnostics.AddError(
			"Conflicting identity provider configuration",
			"Only one of `oauth2_configuration` or `openid_configuration` may be set.",
		)
	}
}

func (r *identityProviderResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *identityProviderResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan identityProviderModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	input := buildIdentityProviderInput(&plan)
	if !plan.IdentityProviderID.IsNull() && !plan.IdentityProviderID.IsUnknown() {
		input["identityProviderId"] = plan.IdentityProviderID.ValueString()
	}

	query := `mutation($input: CreateOrganizationIdentityProvider!) { createIdentityProvider(input: $input) { ` + identityProviderReadFields + ` } }`
	var out struct {
		CreateIdentityProvider gqlIdentityProvider `json:"createIdentityProvider"`
	}
	if err := r.client.Execute(ctx, query, map[string]any{"input": input}, &out); err != nil {
		resp.Diagnostics.AddError("Unable to create identity provider", err.Error())
		return
	}

	applyIdentityProviderComputed(&out.CreateIdentityProvider, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *identityProviderResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state identityProviderModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	query := `query($id: ID!) { identityProvider(id: $id) { ` + identityProviderReadFields + ` } }`
	var out struct {
		IdentityProvider *gqlIdentityProvider `json:"identityProvider"`
	}
	if err := r.client.Execute(ctx, query, map[string]any{"id": state.ID.ValueString()}, &out); err != nil {
		if isIdentityProviderNotFoundErr(err) {
			tflog.Warn(ctx, "Identity provider not found, removing from state", map[string]any{"id": state.ID.ValueString()})
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Unable to read identity provider", err.Error())
		return
	}
	if out.IdentityProvider == nil {
		tflog.Warn(ctx, "Identity provider not found, removing from state", map[string]any{"id": state.ID.ValueString()})
		resp.State.RemoveResource(ctx)
		return
	}

	// Configuration blocks carry the client secret and are not returned readably,
	// so they are managed write-only: keep whatever prior state held rather than
	// refreshing them from the API.
	applyIdentityProviderRead(out.IdentityProvider, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *identityProviderResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state identityProviderModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// identityProviderId is create-only (RequiresReplace) and not part of the
	// update input, so it is omitted here.
	input := buildIdentityProviderInput(&plan)

	query := `mutation($providerId: ID!, $input: UpdateOrganizationIdentityProvider!) { updateIdentityProvider(providerId: $providerId, input: $input) { ` + identityProviderReadFields + ` } }`
	vars := map[string]any{"providerId": state.ID.ValueString(), "input": input}
	var out struct {
		UpdateIdentityProvider gqlIdentityProvider `json:"updateIdentityProvider"`
	}
	if err := r.client.Execute(ctx, query, vars, &out); err != nil {
		resp.Diagnostics.AddError("Unable to update identity provider", err.Error())
		return
	}

	applyIdentityProviderComputed(&out.UpdateIdentityProvider, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *identityProviderResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state identityProviderModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	query := `mutation($id: ID!) { deleteIdentityProvider(id: $id) }`
	if err := r.client.Execute(ctx, query, map[string]any{"id": state.ID.ValueString()}, nil); err != nil {
		resp.Diagnostics.AddError("Unable to delete identity provider", err.Error())
		return
	}
}

func (r *identityProviderResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// isIdentityProviderNotFoundErr reports whether a GraphQL error indicates the
// provider is gone.
func isIdentityProviderNotFoundErr(err error) bool {
	s := strings.ToLower(err.Error())
	return strings.Contains(s, "not found") ||
		strings.Contains(s, "does not exist") ||
		strings.Contains(s, "no identity provider")
}
