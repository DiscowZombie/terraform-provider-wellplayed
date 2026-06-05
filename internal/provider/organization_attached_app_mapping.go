package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// organizationAttachedAppFields selects the readable shape of
// OrganizationAttachedApp. The entity has no `id` field — the (organization,
// appId) pair is its identity — so appId is used as the resource id.
//
// It intentionally omits `permissions`: like OrganizationMember, the API's
// resolver for OrganizationAttachedApp.permissions returns a non-iterable and
// errors ("Expected Iterable, but did not find one") when selected. Permissions
// are managed write-only from configuration instead.
const organizationAttachedAppFields = `
organizationId
groupId
appId
createdAt
updatedAt`

// gqlOrganizationAttachedApp mirrors the readable shape of
// OrganizationAttachedApp. groupId is nullable: an app may be attached with no
// group assigned. permissions is deliberately excluded; see
// organizationAttachedAppFields.
type gqlOrganizationAttachedApp struct {
	OrganizationID string  `json:"organizationId"`
	GroupID        *string `json:"groupId"`
	AppID          string  `json:"appId"`
	CreatedAt      string  `json:"createdAt"`
	UpdatedAt      string  `json:"updatedAt"`
}

// --- Plan -> GraphQL input --------------------------------------------------

// buildSetAttachedAppInput maps the model to SetOrganizationAttachedAppInput.
// appId is required; groupId is omitted when unset (the API leaves the app
// without a group). permissions is always sent — empty when absent — so the
// resource owns the full permission set authoritatively.
func buildSetAttachedAppInput(ctx context.Context, m *organizationAttachedAppModel) (map[string]any, diag.Diagnostics) {
	input := map[string]any{"appId": m.AppID.ValueString()}
	if !m.GroupID.IsNull() && !m.GroupID.IsUnknown() {
		input["groupId"] = m.GroupID.ValueString()
	}
	perms, diags := permissionsToInput(ctx, m.Permissions)
	input["permissions"] = perms
	return input, diags
}

// --- GraphQL response -> model ----------------------------------------------

// applyAttachedAppToModel overwrites the model with the server's view of the
// attachment. Used after create, update, and read. It deliberately leaves
// permissions untouched: those are managed write-only from configuration and
// are not returned readably by the API.
func applyAttachedAppToModel(a *gqlOrganizationAttachedApp, m *organizationAttachedAppModel) {
	m.ID = types.StringValue(a.AppID)
	m.AppID = types.StringValue(a.AppID)
	m.OrganizationID = types.StringValue(a.OrganizationID)
	m.CreatedAt = types.StringValue(a.CreatedAt)
	m.UpdatedAt = types.StringValue(a.UpdatedAt)
	if a.GroupID != nil {
		m.GroupID = types.StringValue(*a.GroupID)
	} else {
		m.GroupID = types.StringNull()
	}
}
