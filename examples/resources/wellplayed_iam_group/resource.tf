# A permission group that tournament editors are assigned to.
resource "wellplayed_iam_group" "editors" {
  name = "Tournament Editors"

  permissions {
    id        = "tournament:write"
    resources = ["tournament_123", "tournament_456"]
  }

  permissions {
    id        = "tournament:read"
    resources = ["*"]
  }
}

# Assign a member to the group above (see wellplayed_iam_member).
resource "wellplayed_iam_member" "editor" {
  email    = "teammate@example.com"
  group_id = wellplayed_iam_group.editors.id
}
