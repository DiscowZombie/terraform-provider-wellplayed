# A simple single-elimination bracket preset, parameterized by team count and
# match length. The Lua script runs when the preset is applied to a tournament
# step and builds the resulting rule set.
resource "wellplayed_preset_script" "single_elim" {
  name        = "Single Elimination"
  description = "Single-elimination bracket with configurable best-of."

  script = <<-LUA
    local team_count = params.team_count
    local best_of = params.best_of or 1

    add_best_of_series_resolution_rule(best_of)
    add_winner_advances_rule()
    add_loser_eliminated_rule()
  LUA

  parameters = [
    {
      name        = "team_count"
      type        = "INT"
      required    = true
      description = "Number of teams competing in this step."
    },
    {
      name          = "best_of"
      type          = "INT"
      required      = false
      default_value = "1"
      description   = "Number of games per series."
    },
  ]
}

# A preset with no parameters.
resource "wellplayed_preset_script" "promote_winners" {
  name   = "Promote Step Winners"
  script = <<-LUA
    add_promote_step_winners_rule()
  LUA
}
