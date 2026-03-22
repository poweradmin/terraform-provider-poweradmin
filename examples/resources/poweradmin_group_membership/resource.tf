# Add a user to a group
resource "poweradmin_group" "ops" {
  name        = "operations"
  description = "Operations team"
}

resource "poweradmin_user" "alice" {
  username = "alice"
  password = "SecurePassword123!"
  fullname = "Alice Smith"
  email    = "alice@example.com"
  active   = true
}

resource "poweradmin_group_membership" "alice_ops" {
  group_id = poweradmin_group.ops.id
  user_id  = poweradmin_user.alice.id
}
