# Add a member by account ID and assign them a permission group.
resource "wellplayed_iam_member" "by_id" {
  user_id  = "acc_01h..."
  group_id = "grp_admins"
}

# Add a member by email, with an extra scoped permission on top of the group.
resource "wellplayed_iam_member" "by_email" {
  email    = "teammate@example.com"
  group_id = "grp_editors"

  permissions {
    id        = "tournament:write"
    resources = ["tournament_123", "tournament_456"]
  }
}
