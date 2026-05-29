# Set the value of a custom field on a specific object. The field definition
# must already exist for the object type — manage it with
# wellplayed_custom_field_definition.

resource "wellplayed_custom_field_definition" "tournament_sponsor" {
  object_type = "Tournament"
  key         = "sponsor"
  name        = "Sponsor"

  type        = "STRING"
  visibility  = "PUBLIC"
  editability = "ALWAYS"
}

resource "wellplayed_tournament" "summer_cup" {
  title       = "Summer Cup"
  description = "Annual summer tournament."
}

# Fill in the "sponsor" field on that tournament.
resource "wellplayed_custom_field_value" "summer_cup_sponsor" {
  object_type = "Tournament"
  object_id   = wellplayed_tournament.summer_cup.id
  key         = wellplayed_custom_field_definition.tournament_sponsor.key

  value = "Acme Corp"
}
