# A minimal tournament: only a title and description are required.
resource "wellplayed_tournament" "minimal" {
  title       = "Spring Open"
  description = "Our first open bracket of the season."
}

# A fully configured tournament with a registration window, team sizing,
# registration conditions, and custom registration fields.
resource "wellplayed_tournament" "championship" {
  title       = "Summer Championship"
  description = "Invitational 5v5 championship."

  start_at               = "2026-07-01T18:00:00Z"
  end_at                 = "2026-07-03T22:00:00Z"
  start_registrations_at = "2026-06-01T00:00:00Z"
  end_registrations_at   = "2026-06-25T23:59:59Z"

  configuration = {
    team_min_size = 2
    team_max_size = 5
    teams_count   = 16

    team_status_after_registration = "AWAITING_FOR_PRESENCE_CONFIRMATION"

    registration_conditions = {
      team_conditions = [
        {
          property           = "region"
          property_condition = "EXISTS"
          error_message      = "Teams must declare a region."

          string_condition = {
            condition_type = "EQ"
            value          = "EU"
          }
        }
      ]

      member_conditions = [
        {
          property_source  = "PLAYER"
          error_message    = "Players must be 18 or older."
          rule_description = "Minimum age 18"

          condition = {
            property           = "age"
            property_condition = "EXISTS"

            numeric_condition = {
              condition_type = "BTE" # greater-than-or-equal
              value          = 18
            }
          }
        }
      ]
    }

    custom_fields = [
      {
        property = "jersey_name"
        name     = "Jersey Name"
        type     = "STRING"
        required = true
        order    = 1
        unique   = false
      },

      {
        property   = "contact_email"
        name       = "Contact Email"
        type       = "EMAIL"
        required   = true
        order      = 2
        unique     = true
        visibility = "OWNER_OR_PERMISSION"
      }
    ]
  }
}
