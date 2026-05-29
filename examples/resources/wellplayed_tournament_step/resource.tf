# A tournament step belongs to a tournament. Reference the tournament's id so
# Terraform creates the tournament first.
resource "wellplayed_tournament" "championship" {
  title       = "Summer Championship"
  description = "Invitational 5v5 championship."
}

# A minimal step: a single-elimination bracket as the first phase.
resource "wellplayed_tournament_step" "playoffs" {
  tournament_id = wellplayed_tournament.championship.id

  name        = "Playoffs"
  description = "Single-elimination playoff bracket."
  order       = 1
  type        = "SINGLE_ELIM"
}

# A group-stage step with custom properties, running before the playoffs.
resource "wellplayed_tournament_step" "group_stage" {
  tournament_id = wellplayed_tournament.championship.id

  name        = "Group Stage"
  description = "Round-robin group stage."
  order       = 0
  type        = "ROUND_ROBIN"

  properties = [
    {
      property = "points_per_win"
      value    = "3"
    },
    {
      property = "points_per_draw"
      value    = "1"
    },
  ]
}
