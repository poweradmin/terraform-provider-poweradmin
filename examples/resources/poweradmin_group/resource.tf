# Create a group with the Administrators permission template
resource "poweradmin_group" "ops" {
  name          = "operations"
  description   = "Operations team"
  perm_templ_id = 7  # Zone Managers group template
}

# Create an admin group
resource "poweradmin_group" "admins" {
  name          = "admins"
  description   = "Administrators group"
  perm_templ_id = 6  # Administrators group template
}
