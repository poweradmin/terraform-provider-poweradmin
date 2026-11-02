# Read the values of one metadata kind of a zone
data "poweradmin_zone_metadata" "example" {
  zone_id = 10
  kind    = "ALLOW-AXFR-FROM"
}

output "axfr_allowed_from" {
  value = data.poweradmin_zone_metadata.example.values
}
