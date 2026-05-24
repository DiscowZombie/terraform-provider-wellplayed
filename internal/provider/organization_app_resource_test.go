package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccOrganizationAppResource exercises the full lifecycle against a real
// organization: create a confidential OAuth2 app, import it, then update its
// attributes. The client secret is returned only on creation, so it is ignored
// on import. Gated behind TF_ACC and the WELLPLAYED_* auth env vars.
func TestAccOrganizationAppResource(t *testing.T) {
	const name = "wellplayed_organization_app.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckProvider(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				// Create: a confidential app with a single redirect URL.
				Config: testAccOrganizationAppConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(name, "name", "Acc Test App"),
					resource.TestCheckResourceAttr(name, "requires_consent", "true"),
					resource.TestCheckResourceAttr(name, "redirect_urls.#", "1"),
					resource.TestCheckResourceAttr(name, "redirect_urls.0", "https://example.test/callback"),
					resource.TestCheckResourceAttr(name, "login_url", "https://example.test/login"),
					resource.TestCheckResourceAttrSet(name, "id"),
					resource.TestCheckResourceAttrSet(name, "secret"),
					resource.TestCheckResourceAttrSet(name, "organization_id"),
					resource.TestCheckResourceAttrSet(name, "created_at"),
				),
			},
			{
				// The secret is returned only on creation, so it can't be
				// verified on import.
				ResourceName:            name,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"secret"},
			},
			{
				// Update: rename, add a redirect URL, and flip requires_consent.
				Config: testAccOrganizationAppConfigUpdated,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(name, "name", "Acc Test App (updated)"),
					resource.TestCheckResourceAttr(name, "requires_consent", "false"),
					resource.TestCheckResourceAttr(name, "redirect_urls.#", "2"),
				),
			},
		},
	})
}

const testAccOrganizationAppConfig = `
resource "wellplayed_organization_app" "test" {
  name        = "Acc Test App"
  description = "Created by acceptance test."

  redirect_urls        = ["https://example.test/callback"]
  logout_redirect_urls = ["https://example.test/"]
  login_url            = "https://example.test/login"
  consent_url          = "https://example.test/consent"
  requires_consent     = true
}
`

const testAccOrganizationAppConfigUpdated = `
resource "wellplayed_organization_app" "test" {
  name        = "Acc Test App (updated)"
  description = "Created by acceptance test."

  redirect_urls        = ["https://example.test/callback", "https://example.test/callback2"]
  logout_redirect_urls = ["https://example.test/"]
  login_url            = "https://example.test/login"
  consent_url          = "https://example.test/consent"
  requires_consent     = false
}
`
