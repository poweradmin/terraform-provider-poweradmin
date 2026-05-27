# List every zone template visible to the authenticated caller
data "poweradmin_zone_templates" "all" {}

output "zone_template_names" {
  value = [for t in data.poweradmin_zone_templates.all.templates : t.name]
}
