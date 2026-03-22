# Look up a group by name
data "poweradmin_group" "ops" {
  name = "operations"
}

# Look up a group by ID
data "poweradmin_group" "by_id" {
  id = 1
}

# Use group data — id is numeric, compatible with group_id attributes
resource "poweradmin_group_zone_assignment" "example" {
  group_id = data.poweradmin_group.ops.id
  zone_id  = poweradmin_zone.example.id
}
