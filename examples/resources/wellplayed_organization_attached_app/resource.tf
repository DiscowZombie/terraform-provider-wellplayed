# A permission group the app will be assigned to.
resource "wellplayed_iam_group" "bots" {
  name = "Service Bots"

  permissions {
    id        = "tournament:read"
    resources = ["*"]
  }
}

# Attach an application to the organization and place it in the group above.
resource "wellplayed_organization_attached_app" "scoreboard" {
  app_id   = "app_01h..."
  group_id = wellplayed_iam_group.bots.id
}

# Attach an app with extra, app-specific permissions on top of (or instead of)
# a group.
resource "wellplayed_organization_attached_app" "importer" {
  app_id = "app_02h..."

  permissions {
    id        = "tournament:write"
    resources = ["tournament_123"]
  }
}
