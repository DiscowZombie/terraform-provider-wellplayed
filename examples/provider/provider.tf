# Copyright (c) HashiCorp, Inc.

provider "wellplayed" {
  organization_id = "my-org" # or WELLPLAYED_ORGANIZATION_ID

  # Application flow (recommended for CI / automation):
  client_id     = var.wellplayed_client_id     # or WELLPLAYED_CLIENT_ID
  client_secret = var.wellplayed_client_secret # or WELLPLAYED_CLIENT_SECRET

  # Static token flow (alternative): supply a pre-obtained OIDC access token.
  # token = var.wellplayed_token # or WELLPLAYED_TOKEN
}
