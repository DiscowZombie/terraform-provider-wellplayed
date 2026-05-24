# Copyright (c) HashiCorp, Inc.

# A minimal OAuth2 identity provider.
resource "wellplayed_identity_provider" "oauth2" {
  name                           = "Acme SSO"
  description                    = "Acme corporate single sign-on."
  enabled                        = true
  required_for_player_validation = false
  allow_login                    = true

  oauth2_configuration = {
    provider_type = "OAUTH2"
    client_id     = "acme-client-id"
    client_secret = var.acme_client_secret
    redirect_url  = "https://play.example.com/auth/callback"

    authorization_url          = "https://sso.acme.com/oauth2/authorize"
    token_endpoint             = "https://sso.acme.com/oauth2/token"
    token_endpoint_auth_method = "CLIENT_SECRET_POST"

    # Fetch the user profile after authentication and map fields onto the
    # player profile.
    data_retrievers = [
      {
        url = "https://sso.acme.com/oauth2/userinfo"

        headers = [
          {
            name  = "Accept"
            value = "application/json"
          }
        ]

        mappings = [
          {
            path      = "email"
            mapped_to = "email"
          },
          {
            path      = "sub"
            mapped_to = "external_id"
            private   = true
          }
        ]
      }
    ]
  }
}

# An OpenID Connect provider derived from its issuer.
resource "wellplayed_identity_provider" "openid" {
  name                           = "Acme OIDC"
  description                    = "Acme OpenID Connect."
  enabled                        = true
  required_for_player_validation = true
  allow_login                    = true

  openid_configuration = {
    provider_type = "OPENID"
    client_id     = "acme-oidc-client"
    client_secret = var.acme_oidc_secret
    redirect_url  = "https://play.example.com/auth/callback"
    issuer        = "https://sso.acme.com"

    data_retrievers = []
  }
}

variable "acme_client_secret" {
  type      = string
  sensitive = true
}

variable "acme_oidc_secret" {
  type      = string
  sensitive = true
}
