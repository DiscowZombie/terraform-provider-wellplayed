package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// organizationGroupFields selects the full readable shape of OrganizationGroup.
// Unlike OrganizationMember, the group's permissions are returned readably, so
// they round-trip through Read and out-of-band changes are detected as drift.
const organizationGroupFields = `
id
name
permissions {
  id
  resources
}
organizationId
type
createdAt
updatedAt`

// gqlOrganizationGroup mirrors the readable shape of OrganizationGroup.
type gqlOrganizationGroup struct {
	ID             string               `json:"id"`
	Name           string               `json:"name"`
	Permissions    []gqlGroupPermission `json:"permissions"`
	OrganizationID string               `json:"organizationId"`
	Type           string               `json:"type"`
	CreatedAt      string               `json:"createdAt"`
	UpdatedAt      string               `json:"updatedAt"`
}

// gqlGroupPermission mirrors a single GroupPermission entry.
type gqlGroupPermission struct {
	ID        string   `json:"id"`
	Resources []string `json:"resources"`
}

// groupPermissionAttrTypes is the object attribute schema of a single
// permission, shared by the iam_group resource and iam_groups data source.
var groupPermissionAttrTypes = map[string]attr.Type{
	"id":        types.StringType,
	"resources": types.SetType{ElemType: types.StringType},
}

// groupPermissionSetType is the framework type of the permissions set.
var groupPermissionSetType = types.SetType{
	ElemType: types.ObjectType{AttrTypes: groupPermissionAttrTypes},
}

// --- Plan -> GraphQL input --------------------------------------------------

// buildOrganizationGroupInput maps the model to the create/update input. Both
// mutations share CreateOrganizationGroupInput, so the same builder serves
// both. permissions is non-null on the input, so an absent block sends `[]`.
func buildOrganizationGroupInput(ctx context.Context, m *iamGroupModel) (map[string]any, diag.Diagnostics) {
	perms, diags := permissionsToInput(ctx, m.Permissions)
	return map[string]any{
		"name":        m.Name.ValueString(),
		"permissions": perms,
	}, diags
}

// --- GraphQL response -> model ----------------------------------------------

// groupPermissionsToSet converts GraphQL permissions to a framework set.
func groupPermissionsToSet(ctx context.Context, perms []gqlGroupPermission) (types.Set, diag.Diagnostics) {
	var diags diag.Diagnostics
	elems := make([]attr.Value, 0, len(perms))
	for _, p := range perms {
		resources, d := types.SetValueFrom(ctx, types.StringType, p.Resources)
		diags.Append(d...)
		obj, d := types.ObjectValue(groupPermissionAttrTypes, map[string]attr.Value{
			"id":        types.StringValue(p.ID),
			"resources": resources,
		})
		diags.Append(d...)
		elems = append(elems, obj)
	}
	set, d := types.SetValue(groupPermissionSetType.ElemType, elems)
	diags.Append(d...)
	return set, diags
}

// applyGroupToModel overwrites the model with the server's view of the group.
// Used after create, update, and read.
func applyGroupToModel(ctx context.Context, g *gqlOrganizationGroup, m *iamGroupModel) diag.Diagnostics {
	m.ID = types.StringValue(g.ID)
	m.Name = types.StringValue(g.Name)
	m.Type = types.StringValue(g.Type)
	m.OrganizationID = types.StringValue(g.OrganizationID)
	m.CreatedAt = types.StringValue(g.CreatedAt)
	m.UpdatedAt = types.StringValue(g.UpdatedAt)

	perms, diags := groupPermissionsToSet(ctx, g.Permissions)
	m.Permissions = perms
	return diags
}
