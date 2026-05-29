# Tournament steps are imported by their ID. Authored fields (name, description,
# order, type) are refreshed from the API on the next plan; `properties` is
# write-only and is not refreshed.
terraform import wellplayed_tournament_step.playoffs step_01h...
