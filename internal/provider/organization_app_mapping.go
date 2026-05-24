package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// organizationAppFields selects the readable subset of the OrganizationApp
// type. `secret` is returned only on creation/reset (null otherwise); it is
// requested here so it can be captured at create time. The relations
// (`creator`, `manifest`, `releases`, `appWebhooks`) are not managed here.
const organizationAppFields = `
id
name
description
icon
shortDescription
public
secret
creatorId
organizationId
createdAt
updatedAt
configuration {
  redirectUrls
  logoutRedirectUrls
  metadata { loginUrl consentUrl requiresConsent public }
}`

// gqlOrganizationApp mirrors the readable subset of the OrganizationApp type.
type gqlOrganizationApp struct {
	ID               string                   `json:"id"`
	Name             string                   `json:"name"`
	Description      string                   `json:"description"`
	Icon             *string                  `json:"icon"`
	ShortDescription *string                  `json:"shortDescription"`
	Public           *bool                    `json:"public"`
	Secret           *string                  `json:"secret"`
	CreatorID        string                   `json:"creatorId"`
	OrganizationID   string                   `json:"organizationId"`
	CreatedAt        string                   `json:"createdAt"`
	UpdatedAt        string                   `json:"updatedAt"`
	Configuration    gqlOrganizationAppConfig `json:"configuration"`
}

type gqlOrganizationAppConfig struct {
	RedirectURLs       []string                         `json:"redirectUrls"`
	LogoutRedirectURLs []string                         `json:"logoutRedirectUrls"`
	Metadata           gqlOrganizationAppConfigMetadata `json:"metadata"`
}

type gqlOrganizationAppConfigMetadata struct {
	LoginURL        string `json:"loginUrl"`
	ConsentURL      string `json:"consentUrl"`
	RequiresConsent bool   `json:"requiresConsent"`
	Public          bool   `json:"public"`
}

// --- Plan -> GraphQL input --------------------------------------------------

// buildOrganizationAppInput maps the model to the create/update input. The two
// inputs share the same field set except for `public`, which is create-only;
// includePublic gates it so the same builder serves both.
func buildOrganizationAppInput(ctx context.Context, m *organizationAppModel, includePublic bool) (map[string]any, diag.Diagnostics) {
	var diags diag.Diagnostics

	redirectURLs, d := listToStrings(ctx, m.RedirectURLs)
	diags.Append(d...)
	logoutRedirectURLs, d := listToStrings(ctx, m.LogoutRedirectURLs)
	diags.Append(d...)

	input := map[string]any{
		"name":               m.Name.ValueString(),
		"description":        m.Description.ValueString(),
		"redirectUrls":       redirectURLs,
		"logoutRedirectUrls": logoutRedirectURLs,
		"loginUrl":           m.LoginURL.ValueString(),
		"consentUrl":         m.ConsentURL.ValueString(),
		"requiresConsent":    m.RequiresConsent.ValueBool(),
	}
	putStr(input, "icon", m.Icon)
	putStr(input, "shortDescription", m.ShortDescription)

	if includePublic && !m.Public.IsNull() && !m.Public.IsUnknown() {
		input["public"] = m.Public.ValueBool()
	}

	return input, diags
}

// --- GraphQL response -> model ----------------------------------------------

// applyOrganizationAppComputed overlays the server-owned fields onto a
// plan-sourced model after create/update. Authored fields are left untouched so
// the applied state matches the plan. `secret` is captured only when the server
// returns it (creation); on update it comes back null and the prior value is
// kept. `public` is filled only when the plan left it unknown (Optional+Computed).
func applyOrganizationAppComputed(g *gqlOrganizationApp, m *organizationAppModel) {
	m.ID = types.StringValue(g.ID)
	m.CreatorID = types.StringValue(g.CreatorID)
	m.OrganizationID = types.StringValue(g.OrganizationID)
	m.CreatedAt = types.StringValue(g.CreatedAt)
	m.UpdatedAt = types.StringValue(g.UpdatedAt)

	if g.Secret != nil {
		m.Secret = types.StringValue(*g.Secret)
	} else if m.Secret.IsUnknown() {
		m.Secret = types.StringNull()
	}

	if m.Public.IsUnknown() {
		m.Public = boolVal(g.Public)
	}
}

// applyOrganizationAppRead refreshes the readable fields onto the model during
// Read. `secret` is left untouched: it is returned only on creation and is
// preserved from prior state.
func applyOrganizationAppRead(g *gqlOrganizationApp, m *organizationAppModel) {
	m.ID = types.StringValue(g.ID)
	m.Name = types.StringValue(g.Name)
	m.Description = types.StringValue(g.Description)
	m.Icon = strVal(g.Icon)
	m.ShortDescription = strVal(g.ShortDescription)
	m.Public = boolVal(g.Public)
	m.RedirectURLs = stringListVal(g.Configuration.RedirectURLs)
	m.LogoutRedirectURLs = stringListVal(g.Configuration.LogoutRedirectURLs)
	m.LoginURL = types.StringValue(g.Configuration.Metadata.LoginURL)
	m.ConsentURL = types.StringValue(g.Configuration.Metadata.ConsentURL)
	m.RequiresConsent = types.BoolValue(g.Configuration.Metadata.RequiresConsent)
	m.CreatorID = types.StringValue(g.CreatorID)
	m.OrganizationID = types.StringValue(g.OrganizationID)
	m.CreatedAt = types.StringValue(g.CreatedAt)
	m.UpdatedAt = types.StringValue(g.UpdatedAt)
}

// --- conversion helpers -----------------------------------------------------

// listToStrings converts a types.List of strings to a Go slice, returning an
// empty (non-nil) slice for null/unknown values so required, non-null list
// inputs are always sent.
func listToStrings(ctx context.Context, l types.List) ([]string, diag.Diagnostics) {
	if l.IsNull() || l.IsUnknown() {
		return []string{}, nil
	}
	out := make([]string, 0, len(l.Elements()))
	diags := l.ElementsAs(ctx, &out, false)
	return out, diags
}

// stringListVal builds a types.List of strings from a Go slice.
func stringListVal(vals []string) types.List {
	elems := make([]attr.Value, len(vals))
	for i, v := range vals {
		elems[i] = types.StringValue(v)
	}
	return types.ListValueMust(types.StringType, elems)
}

// boolVal converts an optional bool pointer to a types.Bool.
func boolVal(p *bool) types.Bool {
	if p == nil {
		return types.BoolNull()
	}
	return types.BoolValue(*p)
}
