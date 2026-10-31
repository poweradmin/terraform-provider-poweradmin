# Define a reusable zone template
resource "poweradmin_zone_template" "default" {
  name        = "default-hosting"
  description = "Default record set for hosting customers"
}

# A global template (visible to all users) — requires ueberuser permission
resource "poweradmin_zone_template" "global" {
  name        = "company-wide"
  description = "Mandatory records for every internal zone"
  is_global   = true
}

# Share a template with user groups (Poweradmin 4.6.0+); members can then use it
resource "poweradmin_zone_template" "shared" {
  name        = "team-hosting"
  description = "Hosting records the ops team can reuse"
  groups      = [3, 4]
}
