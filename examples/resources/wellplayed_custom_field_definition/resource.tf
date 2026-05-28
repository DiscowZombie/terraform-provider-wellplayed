# A free-text custom field attached to every tournament in the organization.
resource "wellplayed_custom_field_definition" "tournament_sponsor" {
  object_type = "Tournament"
  key         = "sponsor"
  name        = "Sponsor"
  description = "Headline sponsor displayed on the tournament page."

  type        = "STRING"
  required    = false
  visibility  = "PUBLIC"
  editability = "ALWAYS"
}

# A required single-select field on player accounts, with a fixed set of
# options. SELECT/MULTI_SELECT fields are the only ones that read `options`.
resource "wellplayed_custom_field_definition" "account_region" {
  object_type = "Account"
  key         = "region"
  name        = "Region"

  type        = "SELECT"
  required    = true
  visibility  = "PUBLIC"
  editability = "ONE_TIME"

  options = [
    { label = "Europe", value = "EU" },
    { label = "North America", value = "NA" },
    { label = "Asia", value = "AS" },
  ]
}

# A regex-validated string only visible to staff with the right permission.
resource "wellplayed_custom_field_definition" "account_internal_id" {
  object_type = "Account"
  key         = "internal_id"
  name        = "Internal ID"
  description = "Internal CRM identifier."

  type             = "STRING"
  visibility       = "WITH_PERMISSION"
  editability      = "WITH_PERMISSION"
  unique           = true
  validation_regex = "^[A-Z]{2}-\\d{6}$"
}
