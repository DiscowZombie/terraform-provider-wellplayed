# Look up an identity provider by ID.
data "wellplayed_identity_provider" "by_id" {
  id = "idp_01h..."
}

# Look up an identity provider by its (unique) display name.
data "wellplayed_identity_provider" "by_name" {
  name = "Acme SSO"
}

# For example, derive a provider-scoped resource from the lookup.
output "acme_sso_enabled" {
  value = data.wellplayed_identity_provider.by_name.enabled
}
