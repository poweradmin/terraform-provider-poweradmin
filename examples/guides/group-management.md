# Group Management

Groups let you organize users into teams and control zone access at the group level. All members of a group gain access to zones assigned to that group.

**Requires Poweradmin 4.2.0+**

## Overview

The group management system uses three resources:

| Resource | Purpose |
|----------|---------|
| `poweradmin_group` | Create/manage the group itself |
| `poweradmin_group_membership` | Add users to a group |
| `poweradmin_group_zone_assignment` | Assign zones to a group |

This design follows Terraform conventions for many-to-many relationships (similar to `aws_iam_group_membership`). Each association is a separate resource so different teams can manage groups, memberships, and zone assignments independently.

## Creating a Group

Every group requires a `perm_templ_id` referencing a group-type permission template. Common templates:

| Template ID | Name | Description |
|-------------|------|-------------|
| 6 | Administrators | Full administrative access |
| 7 | Zone Managers | Full zone management |
| 8 | Editors | Edit records (no SOA/NS) |
| 9 | Viewers | Read-only zone access |
| 10 | Guests | No permissions |

```hcl
resource "poweradmin_group" "ops" {
  name          = "operations"
  description   = "Operations team - manages production DNS"
  perm_templ_id = 7  # Zone Managers
}
```

> **Note:** `perm_templ_id` cannot be changed after creation. Changing it forces recreation of the group.

## Adding Members to a Group

```hcl
resource "poweradmin_user" "alice" {
  username = "alice"
  fullname = "Alice Smith"
  email    = "alice@example.com"
  password = var.alice_password
  active   = true
}

resource "poweradmin_user" "bob" {
  username = "bob"
  fullname = "Bob Jones"
  email    = "bob@example.com"
  password = var.bob_password
  active   = true
}

resource "poweradmin_group_membership" "alice_ops" {
  group_id = poweradmin_group.ops.id
  user_id  = poweradmin_user.alice.id
}

resource "poweradmin_group_membership" "bob_ops" {
  group_id = poweradmin_group.ops.id
  user_id  = poweradmin_user.bob.id
}
```

> **Note:** Changing `group_id` or `user_id` forces replacement (destroy + recreate) of the membership.

## Assigning Zones to a Group

All members of a group gain access to assigned zones:

```hcl
resource "poweradmin_zone" "prod" {
  name = "prod.example.com"
  type = "MASTER"
}

resource "poweradmin_zone" "staging" {
  name = "staging.example.com"
  type = "MASTER"
}

# Ops team gets access to both zones
resource "poweradmin_group_zone_assignment" "ops_prod" {
  group_id = poweradmin_group.ops.id
  zone_id  = poweradmin_zone.prod.id
}

resource "poweradmin_group_zone_assignment" "ops_staging" {
  group_id = poweradmin_group.ops.id
  zone_id  = poweradmin_zone.staging.id
}
```

## Looking Up Existing Groups

```hcl
# By name
data "poweradmin_group" "ops" {
  name = "operations"
}

# By ID
data "poweradmin_group" "by_id" {
  id = 1
}

# Use in assignments
resource "poweradmin_group_zone_assignment" "existing_group" {
  group_id = data.poweradmin_group.ops.id
  zone_id  = poweradmin_zone.prod.id
}
```

The data source exposes `perm_templ_id`, `member_count`, and `zone_count` as computed attributes.

## Importing

```bash
# Import a group by ID
terraform import poweradmin_group.ops 1

# Import a membership by group_id/user_id
terraform import poweradmin_group_membership.alice_ops 1/5

# Import a zone assignment by group_id/zone_id
terraform import poweradmin_group_zone_assignment.ops_prod 1/10
```

## Full Example: Team-Based DNS Access

```hcl
# --- Groups ---
resource "poweradmin_group" "platform" {
  name          = "platform-team"
  description   = "Platform engineering team"
  perm_templ_id = 6  # Administrators
}

resource "poweradmin_group" "app_devs" {
  name          = "app-developers"
  description   = "Application developers - staging access only"
  perm_templ_id = 8  # Editors
}

# --- Users ---
resource "poweradmin_user" "platform_lead" {
  username = "platform.lead"
  fullname = "Platform Lead"
  email    = "platform-lead@example.com"
  password = var.platform_lead_password
  active   = true
}

resource "poweradmin_user" "app_dev" {
  username = "app.dev"
  fullname = "App Developer"
  email    = "app-dev@example.com"
  password = var.app_dev_password
  active   = true
}

# --- Memberships ---
resource "poweradmin_group_membership" "lead_platform" {
  group_id = poweradmin_group.platform.id
  user_id  = poweradmin_user.platform_lead.id
}

resource "poweradmin_group_membership" "dev_appdevs" {
  group_id = poweradmin_group.app_devs.id
  user_id  = poweradmin_user.app_dev.id
}

# --- Zones ---
resource "poweradmin_zone" "production" {
  name = "prod.example.com"
  type = "MASTER"
}

resource "poweradmin_zone" "staging" {
  name = "staging.example.com"
  type = "MASTER"
}

# --- Zone Assignments ---
# Platform team gets both prod and staging
resource "poweradmin_group_zone_assignment" "platform_prod" {
  group_id = poweradmin_group.platform.id
  zone_id  = poweradmin_zone.production.id
}

resource "poweradmin_group_zone_assignment" "platform_staging" {
  group_id = poweradmin_group.platform.id
  zone_id  = poweradmin_zone.staging.id
}

# App developers get staging only
resource "poweradmin_group_zone_assignment" "devs_staging" {
  group_id = poweradmin_group.app_devs.id
  zone_id  = poweradmin_zone.staging.id
}
```
