# Apply a preset script to a tournament step and validate the resulting rule set.
# The tournament, the step, and the preset script are all managed here so
# Terraform builds them in dependency order.
resource "wellplayed_tournament" "championship" {
  title       = "Summer Championship"
  description = "Invitational 5v5 championship."
}

resource "wellplayed_tournament_step" "playoffs" {
  tournament_id = wellplayed_tournament.championship.id

  name        = "Playoffs"
  description = "Single-elimination playoff bracket."
  order       = 1
  type        = "SINGLE_ELIM"
}

resource "wellplayed_preset_script" "single_elim" {
  name   = "Single Elimination"
  script = <<-LUA
    local best_of = params.best_of or 1
    add_best_of_series_resolution_rule(best_of)
    add_winner_advances_rule()
    add_loser_eliminated_rule()
  LUA

  parameters = [
    {
      name     = "team_count"
      type     = "INT"
      required = true
    },
    {
      name          = "best_of"
      type          = "INT"
      required      = false
      default_value = "1"
    },
  ]
}

# Applies the preset to the step and blocks the apply until the resulting rule
# set passes validation.
resource "wellplayed_tournament_step_preset" "playoffs" {
  tournament_step_id = wellplayed_tournament_step.playoffs.id
  preset_script_id   = wellplayed_preset_script.single_elim.id

  parameters = jsonencode({
    team_count = 8
    best_of    = 3
  })

  validate = true
}
