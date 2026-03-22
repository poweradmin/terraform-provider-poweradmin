# Assign a zone to a group — all group members gain access to the zone
resource "poweradmin_group" "ops" {
  name        = "operations"
  description = "Operations team"
}

resource "poweradmin_zone" "example" {
  name = "example.com"
  type = "MASTER"
}

resource "poweradmin_group_zone_assignment" "ops_example" {
  group_id = poweradmin_group.ops.id
  zone_id  = poweradmin_zone.example.id
}
