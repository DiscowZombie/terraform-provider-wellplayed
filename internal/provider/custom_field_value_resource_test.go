package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccCustomFieldValueResource exercises the full lifecycle against a real
// organization: define a Tournament custom field, create a tournament, set the
// field's value, import it, then update the value. It is gated behind TF_ACC
// and the WELLPLAYED_* auth env vars.
func TestAccCustomFieldValueResource(t *testing.T) {
	const name = "wellplayed_custom_field_value.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckProvider(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				// Create: set the initial value.
				Config: testAccCustomFieldValueConfig("Acme Corp"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(name, "object_type", "Tournament"),
					resource.TestCheckResourceAttr(name, "key", "tf_acc_sponsor"),
					resource.TestCheckResourceAttr(name, "value", "Acme Corp"),
					resource.TestCheckResourceAttrSet(name, "object_id"),
					resource.TestCheckResourceAttrSet(name, "id"),
					resource.TestCheckResourceAttrPair(name, "object_id", "wellplayed_tournament.test", "id"),
				),
			},
			{
				ResourceName:      name,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				// Update: change the value in place.
				Config: testAccCustomFieldValueConfig("Globex"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(name, "value", "Globex"),
				),
			},
		},
	})
}

func testAccCustomFieldValueConfig(value string) string {
	return `
resource "wellplayed_custom_field_definition" "test" {
  object_type = "Tournament"
  key         = "tf_acc_sponsor"
  name        = "Sponsor"

  type        = "STRING"
  visibility  = "PUBLIC"
  editability = "ALWAYS"
}

resource "wellplayed_tournament" "test" {
  title       = "Acc Test Tournament"
  description = "Created by acceptance test."
}

resource "wellplayed_custom_field_value" "test" {
  object_type = "Tournament"
  object_id   = wellplayed_tournament.test.id
  key         = wellplayed_custom_field_definition.test.key

  value = "` + value + `"
}
`
}
