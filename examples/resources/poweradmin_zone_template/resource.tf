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
