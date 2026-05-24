// Copyright (c) Mathéo Cimbaro
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// identityProviderReadFields selects the readable scalar subset of the
// OrganizationIdentityProvider type. The `configuration` union and
// `parentIdentityProvider` are deliberately omitted: configuration carries the
// client secret and is not returned readably, so it is managed write-only from
// config (see Read).
const identityProviderReadFields = `
id
name
description
icon
enabled
requiredForPlayerValidation
allowLogin
identityProviderId
organizationId
createdAt
updatedAt`

// gqlIdentityProvider mirrors the readable scalar subset of the
// OrganizationIdentityProvider type.
type gqlIdentityProvider struct {
	ID                          string  `json:"id"`
	Name                        string  `json:"name"`
	Description                 string  `json:"description"`
	Icon                        *string `json:"icon"`
	Enabled                     bool    `json:"enabled"`
	RequiredForPlayerValidation bool    `json:"requiredForPlayerValidation"`
	AllowLogin                  bool    `json:"allowLogin"`
	IdentityProviderID          *string `json:"identityProviderId"`
	OrganizationID              *string `json:"organizationId"`
	CreatedAt                   string  `json:"createdAt"`
	UpdatedAt                   string  `json:"updatedAt"`
}

// --- Plan -> GraphQL input --------------------------------------------------

// buildIdentityProviderInput maps the model to the create/update input. The two
// inputs share the same field set (aside from identityProviderId, handled by
// the caller on create), so the same builder serves both.
func buildIdentityProviderInput(m *identityProviderModel) map[string]any {
	input := map[string]any{
		"name":                        m.Name.ValueString(),
		"description":                 m.Description.ValueString(),
		"enabled":                     m.Enabled.ValueBool(),
		"requiredForPlayerValidation": m.RequiredForPlayerValidation.ValueBool(),
		"allowLogin":                  m.AllowLogin.ValueBool(),
	}
	putStr(input, "icon", m.Icon)

	if m.OAuth2Configuration != nil {
		input["oauth2Configuration"] = buildOAuthConfigInput(m.OAuth2Configuration)
	}
	if m.OpenIDConfiguration != nil {
		input["openidConfiguration"] = buildOAuthConfigInput(m.OpenIDConfiguration)
	}
	return input
}

func buildOAuthConfigInput(c *oauthConfigModel) map[string]any {
	out := map[string]any{
		"providerType": c.ProviderType.ValueString(),
		"clientId":     c.ClientID.ValueString(),
		"clientSecret": c.ClientSecret.ValueString(),
		"redirectUrl":  c.RedirectURL.ValueString(),
	}
	putStr(out, "issuer", c.Issuer)
	putStr(out, "authorizationEndpoint", c.AuthorizationEndpoint)
	putStr(out, "tokenEndpoint", c.TokenEndpoint)
	putStr(out, "authorizationUrl", c.AuthorizationURL)
	putStr(out, "linkRedirectUrl", c.LinkRedirectURL)
	putStr(out, "tokenEndpointAuthMethod", c.TokenEndpointAuthMethod)

	// dataRetrievers is non-null in the schema: always send an array.
	retrievers := make([]map[string]any, 0, len(c.DataRetrievers))
	for i := range c.DataRetrievers {
		retrievers = append(retrievers, buildDataRetrieverInput(&c.DataRetrievers[i]))
	}
	out["dataRetrievers"] = retrievers
	return out
}

func buildDataRetrieverInput(d *dataRetrieverModel) map[string]any {
	headers := make([]map[string]any, 0, len(d.Headers))
	for i := range d.Headers {
		headers = append(headers, map[string]any{
			"name":  d.Headers[i].Name.ValueString(),
			"value": d.Headers[i].Value.ValueString(),
		})
	}

	mappings := make([]map[string]any, 0, len(d.Mappings))
	for i := range d.Mappings {
		mapping := map[string]any{
			"path":     d.Mappings[i].Path.ValueString(),
			"mappedTo": d.Mappings[i].MappedTo.ValueString(),
		}
		if !d.Mappings[i].Private.IsNull() && !d.Mappings[i].Private.IsUnknown() {
			mapping["private"] = d.Mappings[i].Private.ValueBool()
		}
		mappings = append(mappings, mapping)
	}

	return map[string]any{
		"url":     d.URL.ValueString(),
		"headers": headers,
		// mappingConfiguration is non-null and wraps the mappings list.
		"mappingConfiguration": map[string]any{"mappings": mappings},
	}
}

// --- GraphQL response -> model ----------------------------------------------

// applyIdentityProviderComputed overlays the server-owned fields onto a
// plan-sourced model after create/update. Authored fields are left untouched so
// the applied state matches the plan.
func applyIdentityProviderComputed(g *gqlIdentityProvider, m *identityProviderModel) {
	m.ID = types.StringValue(g.ID)
	m.OrganizationID = strVal(g.OrganizationID)
	m.CreatedAt = types.StringValue(g.CreatedAt)
	m.UpdatedAt = types.StringValue(g.UpdatedAt)
}

// applyIdentityProviderRead refreshes the readable scalar fields onto the
// model during Read. The configuration blocks are left untouched: they are
// managed write-only and restored from prior state by the caller.
func applyIdentityProviderRead(g *gqlIdentityProvider, m *identityProviderModel) {
	m.ID = types.StringValue(g.ID)
	m.Name = types.StringValue(g.Name)
	m.Description = types.StringValue(g.Description)
	m.Icon = strVal(g.Icon)
	m.Enabled = types.BoolValue(g.Enabled)
	m.RequiredForPlayerValidation = types.BoolValue(g.RequiredForPlayerValidation)
	m.AllowLogin = types.BoolValue(g.AllowLogin)
	m.IdentityProviderID = strVal(g.IdentityProviderID)
	m.OrganizationID = strVal(g.OrganizationID)
	m.CreatedAt = types.StringValue(g.CreatedAt)
	m.UpdatedAt = types.StringValue(g.UpdatedAt)
}
