package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccTournamentStepResource exercises the full lifecycle against a real
// organization: create a tournament and a minimal step inside it, import the
// step, then update the step's authored fields. It is gated behind TF_ACC and
// the WELLPLAYED_* auth env vars.
func TestAccTournamentStepResource(t *testing.T) {
	const name = "wellplayed_tournament_step.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckProvider(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				// Create: only the required fields.
				Config: testAccTournamentStepConfigMinimal,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(name, "name", "Group Stage"),
					resource.TestCheckResourceAttr(name, "description", "Created by acceptance test."),
					resource.TestCheckResourceAttr(name, "order", "0"),
					resource.TestCheckResourceAttr(name, "type", "ROUND_ROBIN"),
					resource.TestCheckResourceAttrSet(name, "id"),
					resource.TestCheckResourceAttrPair(name, "tournament_id", "wellplayed_tournament.test", "id"),
					resource.TestCheckResourceAttrSet(name, "status"),
					resource.TestCheckResourceAttrSet(name, "created_at"),
				),
			},
			{
				ResourceName:      name,
				ImportState:       true,
				ImportStateVerify: true,
				// properties is write-only and not returned by the API, so it
				// is absent after import and cannot be verified.
				ImportStateVerifyIgnore: []string{"properties"},
			},
			{
				// Update: rename, reorder, change type, and set properties.
				Config: testAccTournamentStepConfigUpdated,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(name, "name", "Playoffs"),
					resource.TestCheckResourceAttr(name, "order", "1"),
					resource.TestCheckResourceAttr(name, "type", "SINGLE_ELIM"),
					resource.TestCheckResourceAttr(name, "properties.#", "1"),
					resource.TestCheckResourceAttr(name, "properties.0.property", "points_per_win"),
				),
			},
		},
	})
}

const testAccTournamentStepConfigMinimal = `
resource "wellplayed_tournament" "test" {
  title       = "Acc Test Tournament"
  description = "Created by acceptance test."
}

resource "wellplayed_tournament_step" "test" {
  tournament_id = wellplayed_tournament.test.id

  name        = "Group Stage"
  description = "Created by acceptance test."
  order       = 0
  type        = "ROUND_ROBIN"
}
`

const testAccTournamentStepConfigUpdated = `
resource "wellplayed_tournament" "test" {
  title       = "Acc Test Tournament"
  description = "Created by acceptance test."
}

resource "wellplayed_tournament_step" "test" {
  tournament_id = wellplayed_tournament.test.id

  name        = "Playoffs"
  description = "Created by acceptance test."
  order       = 1
  type        = "SINGLE_ELIM"

  properties = [
    {
      property = "points_per_win"
      value    = "3"
    },
  ]
}
`
