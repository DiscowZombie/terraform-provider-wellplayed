# Step presets are imported by their tournament step id. Only the resulting rule
# set is recovered; set preset_script_id (and parameters) in configuration to
# match, since the API does not echo them back.
terraform import wellplayed_tournament_step_preset.playoffs step_01h...
