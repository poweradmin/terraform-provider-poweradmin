# List every metadata kind of a zone with its values
data "poweradmin_zone_metadata_entries" "example" {
  zone_id = 10
}

output "metadata" {
  value = { for e in data.poweradmin_zone_metadata_entries.example.entries : e.kind => e.values }
}
