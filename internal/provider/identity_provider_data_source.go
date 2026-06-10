package provider

import (
	"context"
	"fmt"

	"github.com/DiscowZombie/terraform-provider-wellplayed/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure the data source satisfies the framework interfaces.
var (
	_ datasource.DataSource                   = &identityProviderDataSource{}
	_ datasource.DataSourceWithConfigure      = &identityProviderDataSource{}
	_ datasource.DataSourceWithValidateConfig = &identityProviderDataSource{}
)

// NewIdentityProviderDataSource constructs the wellplayed_identity_provider data source.
func NewIdentityProviderDataSource() datasource.DataSource {
	return &identityProviderDataSource{}
}

type identityProviderDataSource struct {
	client *client.Client
}

// identityProviderDataSourceModel maps the data source schema to Go values.
// Unlike the resource model it has no configuration blocks: the configuration
// union carries the client secret and is not returned readably by the API.
type identityProviderDataSourceModel struct {
	ID                          types.String `tfsdk:"id"`
	Name                        types.String `tfsdk:"name"`
	Description                 types.String `tfsdk:"description"`
	Icon                        types.String `tfsdk:"icon"`
	Enabled                     types.Bool   `tfsdk:"enabled"`
	RequiredForPlayerValidation types.Bool   `tfsdk:"required_for_player_validation"`
	AllowLogin                  types.Bool   `tfsdk:"allow_login"`
	IdentityProviderID          types.String `tfsdk:"identity_provider_id"`
	OrganizationID              types.String `tfsdk:"organization_id"`
	CreatedAt                   types.String `tfsdk:"created_at"`
	UpdatedAt                   types.String `tfsdk:"updated_at"`
}

func (d *identityProviderDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_identity_provider"
}

func (d *identityProviderDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Reads a WellPlayed organization identity provider (IdP). Backed by the " +
			"`identityProvider` and `identityProviders` queries.\n\n" +
			"Look the provider up either by `id` or by `name` (exactly one must be set). A `name` lookup " +
			"requires the name to be unique among the organization's identity providers.\n\n" +
			"The OAuth2/OpenID configuration carries the `client_secret` and is returned by the API only as " +
			"an opaque union, so it is not exposed by this data source.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Unique identifier of the identity provider. Exactly one of `id` or `name` " +
					"must be set.",
				Optional: true,
				Computed: true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Display name of the identity provider. Exactly one of `id` or `name` " +
					"must be set; a lookup by name requires the name to be unique.",
				Optional: true,
				Computed: true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "Human-readable description of the identity provider.",
				Computed:            true,
			},
			"icon": schema.StringAttribute{
				MarkdownDescription: "URL or identifier for the provider icon.",
				Computed:            true,
			},
			"enabled": schema.BoolAttribute{
				MarkdownDescription: "Whether this identity provider is currently active.",
				Computed:            true,
			},
			"required_for_player_validation": schema.BoolAttribute{
				MarkdownDescription: "Whether player accounts must be validated through this provider.",
				Computed:            true,
			},
			"allow_login": schema.BoolAttribute{
				MarkdownDescription: "Whether users can log in using this identity provider.",
				Computed:            true,
			},
			"identity_provider_id": schema.StringAttribute{
				MarkdownDescription: "ID of the root identity provider this provider is derived from, if any.",
				Computed:            true,
			},
			"organization_id": schema.StringAttribute{
				MarkdownDescription: "ID of the organization this provider belongs to.",
				Computed:            true,
			},
			"created_at": schema.StringAttribute{
				MarkdownDescription: "When the identity provider was created (RFC 3339).",
				Computed:            true,
			},
			"updated_at": schema.StringAttribute{
				MarkdownDescription: "When the identity provider was last updated (RFC 3339).",
				Computed:            true,
			},
		},
	}
}

func (d *identityProviderDataSource) ValidateConfig(ctx context.Context, req datasource.ValidateConfigRequest, resp *datasource.ValidateConfigResponse) {
	var data identityProviderDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	hasID := !data.ID.IsNull() && !data.ID.IsUnknown()
	hasName := !data.Name.IsNull() && !data.Name.IsUnknown()
	// Unknown values are resolved at apply time, so only reject definite
	// misconfigurations here; Read re-checks once everything is known.
	if hasID && hasName {
		resp.Diagnostics.AddError(
			"Conflicting identity provider lookup",
			"Exactly one of `id` or `name` may be set, not both.",
		)
	}
	if data.ID.IsNull() && data.Name.IsNull() {
		resp.Diagnostics.AddError(
			"Missing identity provider lookup",
			"Exactly one of `id` or `name` must be set.",
		)
	}
}

func (d *identityProviderDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *client.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}
	d.client = c
}

func (d *identityProviderDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data identityProviderDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var provider *gqlIdentityProvider
	switch {
	case !data.ID.IsNull():
		p, err := d.readByID(ctx, data.ID.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Unable to read identity provider", err.Error())
			return
		}
		provider = p
	case !data.Name.IsNull():
		p, err := d.readByName(ctx, data.Name.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Unable to read identity provider", err.Error())
			return
		}
		provider = p
	default:
		resp.Diagnostics.AddError(
			"Missing identity provider lookup",
			"Exactly one of `id` or `name` must be set.",
		)
		return
	}

	data.ID = types.StringValue(provider.ID)
	data.Name = types.StringValue(provider.Name)
	data.Description = types.StringValue(provider.Description)
	data.Icon = strVal(provider.Icon)
	data.Enabled = types.BoolValue(provider.Enabled)
	data.RequiredForPlayerValidation = types.BoolValue(provider.RequiredForPlayerValidation)
	data.AllowLogin = types.BoolValue(provider.AllowLogin)
	data.IdentityProviderID = strVal(provider.IdentityProviderID)
	data.OrganizationID = strVal(provider.OrganizationID)
	data.CreatedAt = types.StringValue(provider.CreatedAt)
	data.UpdatedAt = types.StringValue(provider.UpdatedAt)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (d *identityProviderDataSource) readByID(ctx context.Context, id string) (*gqlIdentityProvider, error) {
	query := `query($id: ID!) { identityProvider(id: $id) { ` + identityProviderReadFields + ` } }`
	var out struct {
		IdentityProvider *gqlIdentityProvider `json:"identityProvider"`
	}
	if err := d.client.Execute(ctx, query, map[string]any{"id": id}, &out); err != nil {
		return nil, err
	}
	if out.IdentityProvider == nil {
		return nil, fmt.Errorf("identity provider %q not found", id)
	}
	return out.IdentityProvider, nil
}

// readByName lists the organization's identity providers and matches on the
// display name. The list query returns a union: entries the caller may only
// see publicly decode with an empty id and are skipped, since they belong to
// other scopes and cannot be read in full anyway.
func (d *identityProviderDataSource) readByName(ctx context.Context, name string) (*gqlIdentityProvider, error) {
	query := `query { identityProviders { ... on OrganizationIdentityProvider { ` + identityProviderReadFields + ` } } }`
	var out struct {
		IdentityProviders []gqlIdentityProvider `json:"identityProviders"`
	}
	if err := d.client.Execute(ctx, query, nil, &out); err != nil {
		return nil, err
	}

	var matches []gqlIdentityProvider
	for _, p := range out.IdentityProviders {
		if p.ID != "" && p.Name == name {
			matches = append(matches, p)
		}
	}
	switch len(matches) {
	case 0:
		return nil, fmt.Errorf("no identity provider named %q", name)
	case 1:
		return &matches[0], nil
	default:
		return nil, fmt.Errorf("%d identity providers named %q; use `id` to disambiguate", len(matches), name)
	}
}
