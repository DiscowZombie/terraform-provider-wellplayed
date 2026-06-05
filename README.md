# Terraform Provider for WellPlayed

[![Tests](https://github.com/DiscowZombie/terraform-provider-wellplayed/actions/workflows/test.yml/badge.svg)](https://github.com/DiscowZombie/terraform-provider-wellplayed/actions/workflows/test.yml)

A [Terraform](https://www.terraform.io) provider for managing
[WellPlayed](https://well-played.gg/) resources through its GraphQL API.
Built on the [Terraform Plugin Framework](https://github.com/hashicorp/terraform-plugin-framework).

## Requirements

- [Terraform](https://developer.hashicorp.com/terraform/downloads) >= 1.0
- [Go](https://golang.org/doc/install) >= 1.25 (only to build the provider from source)

## Using the provider

```terraform
terraform {
  required_providers {
    wellplayed = {
      source = "DiscowZombie/wellplayed"
    }
  }
}

provider "wellplayed" {
  organization_id = "my-org" # or WELLPLAYED_ORGANIZATION_ID

  # Application flow (recommended for CI / automation):
  client_id     = var.wellplayed_client_id     # or WELLPLAYED_CLIENT_ID
  client_secret = var.wellplayed_client_secret # or WELLPLAYED_CLIENT_SECRET

  # Static token flow (alternative): supply a pre-obtained OIDC access token.
  # token = var.wellplayed_token # or WELLPLAYED_TOKEN
}
```

### Authentication

Every request needs an `organization_id` (the org short id) plus exactly one of
the two auth flows:

- **Application flow** — set `client_id` and `client_secret`; the provider
  exchanges them for a service token at the OAuth2 token endpoint. Best for CI
  and automation.
- **Static token flow** — set `token` to a pre-obtained OIDC access token.

The two flows are mutually exclusive. Every attribute also resolves from a
`WELLPLAYED_*` environment variable (`WELLPLAYED_ORGANIZATION_ID`,
`WELLPLAYED_CLIENT_ID`, `WELLPLAYED_CLIENT_SECRET`, `WELLPLAYED_TOKEN`,
`WELLPLAYED_ENDPOINT`, `WELLPLAYED_TOKEN_URL`) so secrets need not appear in
HCL.

## Resources

| Resource | Description |
|----------|-------------|
| `wellplayed_iam_member` | Assigns an account to an organization permission group, granting it that group's permissions. |
| `wellplayed_iam_group` | Manages an organization permission group: a named set of permissions that members and apps can be assigned to. |
| `wellplayed_tournament` | Manages a tournament, including registration windows, team sizing, registration conditions, and custom fields. |
| `wellplayed_tournament_step` | Manages a step (phase) within a tournament, such as a group, round-robin, or elimination bracket. |
| `wellplayed_tournament_step_preset` | Applies a preset script to a tournament step to produce its rule set, optionally validating the result. |
| `wellplayed_preset_script` | Manages a named Lua preset script (with declared parameters) that builds a tournament step rule set. |
| `wellplayed_identity_provider` | Configures an OAuth2/OpenID Connect identity provider (IdP) for the organization. |
| `wellplayed_organization_app` | Manages an organization OAuth2 application (client credentials). |
| `wellplayed_organization_attached_app` | Attaches an app to the organization and places it in a permission group — the app-side counterpart of `wellplayed_iam_member`. |
| `wellplayed_custom_field_definition` | Manages a custom field definition attached to an object type (Tournament, Account, Team, etc.). |
| `wellplayed_custom_field_value` | Sets the value of a single custom field on an object (Tournament, Account, Team, etc.). |

See the [`examples/`](./examples) directory and the generated
[`docs/`](./docs) for full schemas and usage. Each resource supports import —
see the per-resource `import.sh` files under `examples/resources/`.

## Building the provider

```shell
go install
```

To run the provider against a local build, see the
[Development Overrides](https://developer.hashicorp.com/terraform/cli/config/config-file#development-overrides-for-provider-developers)
documentation and the `dev_overrides` block pattern.

## Developing the provider

Common tasks are wired through the `GNUmakefile`:

```shell
make build      # compile the provider
make testacc    # run acceptance tests (requires a live WellPlayed org)
make generate   # regenerate docs from schema + examples
make lint       # run golangci-lint
```

Acceptance tests create real resources and require credentials in the
environment:

```shell
export TF_ACC=1
export WELLPLAYED_ORGANIZATION_ID=...
export WELLPLAYED_CLIENT_ID=...
export WELLPLAYED_CLIENT_SECRET=...
make testacc
```

## License

Distributed under the Mozilla Public License 2.0. See [LICENSE](./LICENSE).
