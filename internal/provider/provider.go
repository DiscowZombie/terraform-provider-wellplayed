// Copyright (c) Mathéo Cimbaro
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"os"

	"github.com/DiscowZombie/terraform-provider-wellplayed/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Environment variables used as fallbacks for provider configuration.
const (
	envEndpoint       = "WELLPLAYED_ENDPOINT"
	envOrganizationID = "WELLPLAYED_ORGANIZATION_ID"
	envToken          = "WELLPLAYED_TOKEN"
	envClientID       = "WELLPLAYED_CLIENT_ID"
	envClientSecret   = "WELLPLAYED_CLIENT_SECRET"
	envTokenURL       = "WELLPLAYED_TOKEN_URL"
)

// Ensure WellPlayedProvider satisfies the provider interface.
var _ provider.Provider = &WellPlayedProvider{}

// WellPlayedProvider is the WellPlayed GraphQL Terraform provider.
type WellPlayedProvider struct {
	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance
	// testing.
	version string
}

// WellPlayedProviderModel maps provider schema attributes to Go values.
type WellPlayedProviderModel struct {
	Endpoint       types.String `tfsdk:"endpoint"`
	OrganizationID types.String `tfsdk:"organization_id"`
	Token          types.String `tfsdk:"token"`
	ClientID       types.String `tfsdk:"client_id"`
	ClientSecret   types.String `tfsdk:"client_secret"`
	TokenURL       types.String `tfsdk:"token_url"`
}

func (p *WellPlayedProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "wellplayed"
	resp.Version = p.version
}

func (p *WellPlayedProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Interact with the [WellPlayed](https://well-played.gg/) GraphQL API. " +
			"Authenticate either with the application flow (`client_id` + `client_secret`, exchanged for a " +
			"service token) or by supplying a pre-obtained OIDC access `token`.",
		Attributes: map[string]schema.Attribute{
			"endpoint": schema.StringAttribute{
				MarkdownDescription: "WellPlayed GraphQL endpoint. Defaults to `" + client.DefaultEndpoint +
					"`. May also be set with the `" + envEndpoint + "` environment variable.",
				Optional: true,
			},
			"organization_id": schema.StringAttribute{
				MarkdownDescription: "Organization short id, sent as the `organization-id` header on every request. " +
					"May also be set with the `" + envOrganizationID + "` environment variable.",
				Optional: true,
			},
			"token": schema.StringAttribute{
				MarkdownDescription: "Pre-obtained OIDC access token (static token flow). Mutually exclusive with " +
					"`client_id`/`client_secret`. May also be set with the `" + envToken + "` environment variable.",
				Optional:  true,
				Sensitive: true,
			},
			"client_id": schema.StringAttribute{
				MarkdownDescription: "OAuth2 client id for the application flow. Requires `client_secret`. " +
					"May also be set with the `" + envClientID + "` environment variable.",
				Optional: true,
			},
			"client_secret": schema.StringAttribute{
				MarkdownDescription: "OAuth2 client secret for the application flow. Requires `client_id`. " +
					"May also be set with the `" + envClientSecret + "` environment variable.",
				Optional:  true,
				Sensitive: true,
			},
			"token_url": schema.StringAttribute{
				MarkdownDescription: "OAuth2 token endpoint for the application flow. Defaults to `" +
					client.DefaultTokenURL + "`. May also be set with the `" + envTokenURL + "` environment variable.",
				Optional: true,
			},
		},
	}
}

func (p *WellPlayedProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var data WellPlayedProviderModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Config values take precedence; fall back to environment variables.
	endpoint := stringOrEnv(data.Endpoint, envEndpoint)
	organizationID := stringOrEnv(data.OrganizationID, envOrganizationID)
	token := stringOrEnv(data.Token, envToken)
	clientID := stringOrEnv(data.ClientID, envClientID)
	clientSecret := stringOrEnv(data.ClientSecret, envClientSecret)
	tokenURL := stringOrEnv(data.TokenURL, envTokenURL)

	if organizationID == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("organization_id"),
			"Missing organization id",
			"The provider requires an organization id. Set the `organization_id` attribute or the "+
				envOrganizationID+" environment variable.",
		)
	}

	// Exactly one auth flow must be configured.
	hasToken := token != ""
	hasApp := clientID != "" || clientSecret != ""
	switch {
	case hasToken && hasApp:
		resp.Diagnostics.AddError(
			"Conflicting authentication configuration",
			"Set either `token` (static token flow) or `client_id`/`client_secret` (application flow), not both.",
		)
	case !hasToken && !hasApp:
		resp.Diagnostics.AddError(
			"Missing authentication configuration",
			"Configure either `token`, or both `client_id` and `client_secret` "+
				"(or the equivalent WELLPLAYED_* environment variables).",
		)
	case hasApp && (clientID == "" || clientSecret == ""):
		resp.Diagnostics.AddError(
			"Incomplete application credentials",
			"The application flow requires both `client_id` and `client_secret`.",
		)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	c, err := client.New(ctx, client.Config{
		Endpoint:       endpoint,
		OrganizationID: organizationID,
		Token:          token,
		ClientID:       clientID,
		ClientSecret:   clientSecret,
		TokenURL:       tokenURL,
	})
	if err != nil {
		resp.Diagnostics.AddError("Unable to create WellPlayed client", err.Error())
		return
	}

	resp.DataSourceData = c
	resp.ResourceData = c
}

func (p *WellPlayedProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewIAMMemberResource,
		NewTournamentResource,
	}
}

func (p *WellPlayedProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{}
}

// stringOrEnv returns the configured value if known and non-null, otherwise
// the named environment variable.
func stringOrEnv(v types.String, env string) string {
	if !v.IsNull() && !v.IsUnknown() {
		return v.ValueString()
	}
	return os.Getenv(env)
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &WellPlayedProvider{
			version: version,
		}
	}
}
