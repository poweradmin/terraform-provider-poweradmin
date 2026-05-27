# Look up a zone template by name
data "poweradmin_zone_template" "default" {
  name = "default-hosting"
}

# Look up a zone template by ID
data "poweradmin_zone_template" "by_id" {
  id = 1
}

# Inspect the records the template will apply
output "default_template_records" {
  value = data.poweradmin_zone_template.default.records
}
