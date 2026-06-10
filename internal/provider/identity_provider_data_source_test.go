package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccIdentityProviderDataSource creates a provider via the resource, then
// reads it back through the data source by id and by name. Gated behind TF_ACC
// and the WELLPLAYED_* auth env vars.
func TestAccIdentityProviderDataSource(t *testing.T) {
	const (
		resourceName = "wellplayed_identity_provider.test"
		byID         = "data.wellplayed_identity_provider.by_id"
		byName       = "data.wellplayed_identity_provider.by_name"
	)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckProvider(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccIdentityProviderDataSourceConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					// By id.
					resource.TestCheckResourceAttrPair(byID, "id", resourceName, "id"),
					resource.TestCheckResourceAttr(byID, "name", "Acc Test IdP DS"),
					resource.TestCheckResourceAttr(byID, "description", "Created by acceptance test."),
					resource.TestCheckResourceAttr(byID, "enabled", "true"),
					resource.TestCheckResourceAttr(byID, "required_for_player_validation", "false"),
					resource.TestCheckResourceAttr(byID, "allow_login", "true"),
					resource.TestCheckResourceAttrSet(byID, "organization_id"),
					resource.TestCheckResourceAttrSet(byID, "created_at"),
					resource.TestCheckResourceAttrSet(byID, "updated_at"),
					// By name.
					resource.TestCheckResourceAttrPair(byName, "id", resourceName, "id"),
					resource.TestCheckResourceAttr(byName, "name", "Acc Test IdP DS"),
					resource.TestCheckResourceAttr(byName, "enabled", "true"),
				),
			},
		},
	})
}

const testAccIdentityProviderDataSourceConfig = `
resource "wellplayed_identity_provider" "test" {
  name                           = "Acc Test IdP DS"
  description                    = "Created by acceptance test."
  enabled                        = true
  required_for_player_validation = false
  allow_login                    = true

  oauth2_configuration = {
    provider_type = "OAUTH2"
    client_id     = "acc-client-id"
    client_secret = "acc-client-secret"
    redirect_url  = "https://example.test/callback"

    data_retrievers = []
  }
}

data "wellplayed_identity_provider" "by_id" {
  id = wellplayed_identity_provider.test.id
}

data "wellplayed_identity_provider" "by_name" {
  name = wellplayed_identity_provider.test.name
}
`
