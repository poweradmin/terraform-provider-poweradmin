# User Management

This guide covers managing Poweradmin users and their permissions.

## Creating Users

```hcl
resource "poweradmin_user" "dns_admin" {
  username    = "dns.admin"
  fullname    = "DNS Administrator"
  email       = "dns-admin@example.com"
  password    = var.dns_admin_password
  active      = true
  description = "DNS team administrator"
  perm_templ  = 1  # Permission template ID
}
```

## User Attributes

| Attribute | Type | Required | Description |
|-----------|------|----------|-------------|
| `username` | string | Yes | Unique login name |
| `password` | string | Yes | User password (write-only, hashed by Poweradmin) |
| `fullname` | string | Yes | Display name |
| `email` | string | Yes | Email address |
| `description` | string | No | Notes about the user |
| `active` | bool | No | Account status (default: `true`) |
| `perm_templ` | number | No | Permission template ID to assign |
| `use_ldap` | bool | No | Use LDAP authentication (default: `false`) |

## Permission Templates

Permission templates define what actions a user can perform. Look up available templates with the data source:

```hcl
data "poweradmin_permission" "operator" {
  name = "zone_content_edit_own"
}

resource "poweradmin_user" "operator" {
  username   = "zone.operator"
  fullname   = "Zone Operator"
  email      = "operator@example.com"
  password   = var.operator_password
  active     = true
  perm_templ = data.poweradmin_permission.operator.id
}
```

## LDAP Users

Users can authenticate via LDAP instead of local passwords. The password field is still required but will not be used for authentication.

```hcl
resource "poweradmin_user" "ldap_user" {
  username = "jsmith"
  fullname = "John Smith"
  email    = "jsmith@example.com"
  password = "placeholder"  # Not used with LDAP
  active   = true
  use_ldap = true
}
```

## Deactivating Users

Set `active = false` to disable a user without deleting them. They will not be able to log in but their zone ownership is preserved.

```hcl
resource "poweradmin_user" "departing" {
  username = "former.employee"
  fullname = "Former Employee"
  email    = "former@example.com"
  password = var.placeholder_password
  active   = false
}
```

## Importing Users

```bash
# Import by user ID
terraform import poweradmin_user.dns_admin 5
```

> **Note:** The `password` attribute cannot be read from the API. After importing, you must set it in your Terraform configuration. The provider will keep the existing password until you change it.

## Managing User Access with Groups

For managing which zones a user can access, consider using [Groups](group-management.md) (Poweradmin 4.2.0+). Groups let you assign zones to teams rather than individual users.
