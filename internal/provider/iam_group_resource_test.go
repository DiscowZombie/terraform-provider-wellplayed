package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccIAMGroupResource exercises the full lifecycle against a real
// organization: create a group with a scoped permission, import it, then
// update (rename + change permissions). Group creation is self-contained, so
// it only needs the provider-level fixtures.
func TestAccIAMGroupResource(t *testing.T) {
	const name = "wellplayed_iam_group.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckProvider(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				// Create: a group with a single scoped permission.
				Config: testAccIAMGroupConfig("tf-acc-group", "tournament:write", "tournament_acc_test"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(name, "name", "tf-acc-group"),
					resource.TestCheckResourceAttr(name, "permissions.#", "1"),
					resource.TestCheckResourceAttr(name, "permissions.0.id", "tournament:write"),
					resource.TestCheckResourceAttr(name, "permissions.0.resources.#", "1"),
					resource.TestCheckResourceAttrSet(name, "id"),
					resource.TestCheckResourceAttrSet(name, "type"),
					resource.TestCheckResourceAttrSet(name, "organization_id"),
					resource.TestCheckResourceAttrSet(name, "created_at"),
				),
			},
			{
				// Import by group ID; every field round-trips through Read.
				ResourceName:      name,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				// Update: rename and re-scope the permission.
				Config: testAccIAMGroupConfig("tf-acc-group-renamed", "tournament:read", "tournament_acc_other"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(name, "name", "tf-acc-group-renamed"),
					resource.TestCheckResourceAttr(name, "permissions.0.id", "tournament:read"),
					resource.TestCheckResourceAttr(name, "permissions.0.resources.0", "tournament_acc_other"),
				),
			},
		},
	})
}

func testAccIAMGroupConfig(name, permission, resourceID string) string {
	return fmt.Sprintf(`
resource "wellplayed_iam_group" "test" {
  name = %[1]q

  permissions {
    id        = %[2]q
    resources = [%[3]q]
  }
}
`, name, permission, resourceID)
}
