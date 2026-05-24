// Copyright (c) Mathéo Cimbaro
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccTournamentResource exercises the full lifecycle against a real
// organization: create a minimal tournament, import it, then update it with a
// full configuration block (team sizing, registration conditions, and a custom
// field). It is gated behind TF_ACC and the WELLPLAYED_* auth env vars.
func TestAccTournamentResource(t *testing.T) {
	const name = "wellplayed_tournament.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckProvider(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				// Create: only the required fields.
				Config: testAccTournamentConfigMinimal,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(name, "title", "Acc Test Tournament"),
					resource.TestCheckResourceAttr(name, "description", "Created by acceptance test."),
					resource.TestCheckResourceAttrSet(name, "id"),
					resource.TestCheckResourceAttrSet(name, "organization_id"),
					resource.TestCheckResourceAttrSet(name, "created_at"),
				),
			},
			{
				ResourceName:      name,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				// Update: rename and attach a full configuration.
				Config: testAccTournamentConfigFull,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(name, "title", "Acc Test Tournament (updated)"),
					resource.TestCheckResourceAttr(name, "configuration.team_min_size", "1"),
					resource.TestCheckResourceAttr(name, "configuration.team_max_size", "5"),
					resource.TestCheckResourceAttr(name, "configuration.registration_conditions.member_conditions.#", "1"),
					resource.TestCheckResourceAttr(name, "configuration.custom_fields.#", "1"),
					resource.TestCheckResourceAttr(name, "configuration.custom_fields.0.property", "jersey_name"),
				),
			},
		},
	})
}

const testAccTournamentConfigMinimal = `
resource "wellplayed_tournament" "test" {
  title       = "Acc Test Tournament"
  description = "Created by acceptance test."
}
`

const testAccTournamentConfigFull = `
resource "wellplayed_tournament" "test" {
  title       = "Acc Test Tournament (updated)"
  description = "Created by acceptance test."

  configuration {
    team_min_size = 1
    team_max_size = 5
    teams_count   = 8

    registration_conditions {
      member_conditions {
        property_source = "PLAYER"

        condition {
          property           = "age"
          property_condition = "EXISTS"

          numeric_condition {
            condition_type = "BTE"
            value          = 18
          }
        }
      }
    }

    custom_fields {
      property = "jersey_name"
      name     = "Jersey Name"
      type     = "STRING"
      required = true
      order    = 1
      unique   = false
    }
  }
}
`
