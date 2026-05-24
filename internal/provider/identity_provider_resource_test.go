package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccIdentityProviderResource exercises the full lifecycle against a real
// organization: create a provider with an OAuth2 configuration, import it, then
// update its top-level attributes. The configuration block is managed
// write-only, so it can't round-trip through Read and is ignored on import.
// Gated behind TF_ACC and the WELLPLAYED_* auth env vars.
func TestAccIdentityProviderResource(t *testing.T) {
	const name = "wellplayed_identity_provider.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckProvider(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				// Create: an OAuth2 provider with a data retriever.
				Config: testAccIdentityProviderConfigOAuth2,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(name, "name", "Acc Test IdP"),
					resource.TestCheckResourceAttr(name, "enabled", "true"),
					resource.TestCheckResourceAttr(name, "allow_login", "true"),
					resource.TestCheckResourceAttr(name, "oauth2_configuration.provider_type", "OAUTH2"),
					resource.TestCheckResourceAttr(name, "oauth2_configuration.client_id", "acc-client-id"),
					resource.TestCheckResourceAttrSet(name, "id"),
					resource.TestCheckResourceAttrSet(name, "organization_id"),
					resource.TestCheckResourceAttrSet(name, "created_at"),
				),
			},
			{
				// The configuration blocks are write-only and not refreshed, so
				// they cannot be verified on import.
				ResourceName:            name,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"oauth2_configuration", "openid_configuration"},
			},
			{
				// Update: rename, disable, and tweak the configuration.
				Config: testAccIdentityProviderConfigOAuth2Updated,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(name, "name", "Acc Test IdP (updated)"),
					resource.TestCheckResourceAttr(name, "enabled", "false"),
					resource.TestCheckResourceAttr(name, "oauth2_configuration.data_retrievers.#", "1"),
				),
			},
		},
	})
}

const testAccIdentityProviderConfigOAuth2 = `
resource "wellplayed_identity_provider" "test" {
  name                           = "Acc Test IdP"
  description                    = "Created by acceptance test."
  enabled                        = true
  required_for_player_validation = false
  allow_login                    = true

  oauth2_configuration = {
    provider_type = "OAUTH2"
    client_id     = "acc-client-id"
    client_secret = "acc-client-secret"
    redirect_url  = "https://example.test/callback"

    data_retrievers = [
      {
        url = "https://example.test/userinfo"

        headers = [
          {
            name  = "Authorization"
            value = "Bearer token"
          }
        ]

        mappings = [
          {
            path      = "data.email"
            mapped_to = "email"
          }
        ]
      }
    ]
  }
}
`

const testAccIdentityProviderConfigOAuth2Updated = `
resource "wellplayed_identity_provider" "test" {
  name                           = "Acc Test IdP (updated)"
  description                    = "Created by acceptance test."
  enabled                        = false
  required_for_player_validation = false
  allow_login                    = true

  oauth2_configuration = {
    provider_type = "OAUTH2"
    client_id     = "acc-client-id"
    client_secret = "acc-client-secret"
    redirect_url  = "https://example.test/callback"

    data_retrievers = [
      {
        url = "https://example.test/userinfo"

        mappings = [
          {
            path      = "data.email"
            mapped_to = "email"
            private   = true
          }
        ]
      }
    ]
  }
}
`
