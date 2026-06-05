package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccOrganizationAttachedAppResource exercises the full lifecycle against a
// real organization: create an app and two groups, attach the app to the first
// group, import it, then move it to the second group and grant an extra
// permission. Gated behind TF_ACC and the WELLPLAYED_* auth env vars.
func TestAccOrganizationAttachedAppResource(t *testing.T) {
	const name = "wellplayed_organization_attached_app.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckProvider(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				// Create: attach the app to the first group, no extra permissions.
				Config: testAccOrganizationAttachedAppConfig(false),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair(name, "app_id", "wellplayed_organization_app.test", "id"),
					resource.TestCheckResourceAttrPair(name, "group_id", "wellplayed_iam_group.a", "id"),
					resource.TestCheckResourceAttrSet(name, "id"),
					resource.TestCheckResourceAttrSet(name, "organization_id"),
					resource.TestCheckResourceAttrSet(name, "created_at"),
				),
			},
			{
				// Import by app ID; every field round-trips through Read.
				ResourceName:      name,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				// Update: move to the second group and grant an extra permission.
				Config: testAccOrganizationAttachedAppConfig(true),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair(name, "group_id", "wellplayed_iam_group.b", "id"),
					resource.TestCheckResourceAttr(name, "permissions.#", "1"),
					resource.TestCheckResourceAttr(name, "permissions.0.id", "tournament:write"),
					resource.TestCheckResourceAttr(name, "permissions.0.resources.0", "tournament_acc_test"),
				),
			},
		},
	})
}

func testAccOrganizationAttachedAppConfig(updated bool) string {
	group := "wellplayed_iam_group.a"
	perms := ""
	if updated {
		group = "wellplayed_iam_group.b"
		perms = `
  permissions {
    id        = "tournament:write"
    resources = ["tournament_acc_test"]
  }`
	}

	return fmt.Sprintf(`
resource "wellplayed_organization_app" "test" {
  name        = "Acc Test Attached App"
  description = "Created by acceptance test."

  redirect_urls        = ["https://example.test/callback"]
  logout_redirect_urls = ["https://example.test/"]
  login_url            = "https://example.test/login"
  consent_url          = "https://example.test/consent"
  requires_consent     = true
}

resource "wellplayed_iam_group" "a" {
  name = "tf-acc-attached-app-a"
}

resource "wellplayed_iam_group" "b" {
  name = "tf-acc-attached-app-b"
}

resource "wellplayed_organization_attached_app" "test" {
  app_id   = wellplayed_organization_app.test.id
  group_id = %[1]s.id
%[2]s
}
`, group, perms)
}
