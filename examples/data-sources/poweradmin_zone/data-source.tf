# Look up a zone by name
data "poweradmin_zone" "example" {
  name = "example.com"
}

# Look up a zone by ID
data "poweradmin_zone" "by_id" {
  id = "123"
}

# Use zone data in a resource
resource "poweradmin_record" "www" {
  zone_id = data.poweradmin_zone.example.id
  name    = "www"
  type    = "A"
  content = "192.0.2.100"
  ttl     = 3600
}

# Output zone information
output "zone_type" {
  value = data.poweradmin_zone.example.type
}

output "zone_description" {
  value = data.poweradmin_zone.example.description
}
