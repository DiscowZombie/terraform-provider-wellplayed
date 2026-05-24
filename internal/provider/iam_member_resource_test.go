package provider

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccIAMMemberResource exercises the full lifecycle against a real
// organization: create, import, then update (group change + scoped
// permission). It is gated behind TF_ACC and the WELLPLAYED_* env vars
// asserted in testAccPreCheck.
func TestAccIAMMemberResource(t *testing.T) {
	userID := os.Getenv(envTestUserID)
	group1 := os.Getenv(envTestGroup)
	group2 := os.Getenv(envTestGroup2)

	const name = "wellplayed_iam_member.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				// Create: assign the member to the first group.
				Config: testAccIAMMemberConfigBasic(userID, group1),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(name, "user_id", userID),
					resource.TestCheckResourceAttr(name, "group_id", group1),
					resource.TestCheckResourceAttr(name, "member_id", userID),
					resource.TestCheckResourceAttrSet(name, "id"),
					resource.TestCheckResourceAttrSet(name, "organization_id"),
					resource.TestCheckResourceAttrSet(name, "created_at"),
				),
			},
			{
				// Import by account (member) ID. user_id/email are not
				// returned by the API, so they can't round-trip through Read.
				ResourceName:            name,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"user_id", "email"},
			},
			{
				// Update: move to the second group and grant a scoped permission.
				Config: testAccIAMMemberConfigWithPermission(userID, group2),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(name, "group_id", group2),
					resource.TestCheckResourceAttr(name, "permissions.#", "1"),
					resource.TestCheckResourceAttr(name, "permissions.0.id", "tournament:write"),
					resource.TestCheckResourceAttr(name, "permissions.0.resources.#", "1"),
				),
			},
		},
	})
}

func testAccIAMMemberConfigBasic(userID, groupID string) string {
	return fmt.Sprintf(`
resource "wellplayed_iam_member" "test" {
  user_id  = %[1]q
  group_id = %[2]q
}
`, userID, groupID)
}

func testAccIAMMemberConfigWithPermission(userID, groupID string) string {
	return fmt.Sprintf(`
resource "wellplayed_iam_member" "test" {
  user_id  = %[1]q
  group_id = %[2]q

  permissions {
    id        = "tournament:write"
    resources = ["tournament_acc_test"]
  }
}
`, userID, groupID)
}
