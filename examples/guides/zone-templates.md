# Zone Templates

Zone templates are reusable record sets that Poweradmin applies when a zone is created from them.

**Requires Poweradmin 4.2.0+** (sharing templates with groups requires 4.6.0+)

## Overview

| Resource / Data Source | Purpose |
|------------------------|---------|
| `poweradmin_zone_template` (resource) | Create a template, make it global or share it with groups |
| `poweradmin_zone_template_record` (resource) | Add records to a template |
| `poweradmin_zone_template` (data source) | Look up one template, with its records |
| `poweradmin_zone_templates` (data source) | List the templates visible to the caller |

## Defining a Template

Template records can use placeholders that are substituted when a zone is created: `[ZONE]`, `[NS1]`, `[NS2]`, `[HOSTMASTER]` and `[SERIAL]`.

```hcl
resource "poweradmin_zone_template" "hosting" {
  name        = "default-hosting"
  description = "Default record set for hosting customers"
}

resource "poweradmin_zone_template_record" "soa" {
  template_id = poweradmin_zone_template.hosting.id
  name        = "[ZONE]"
  type        = "SOA"
  content     = "[NS1] [HOSTMASTER] [SERIAL] 28800 7200 604800 86400"
  ttl         = 86400
}

resource "poweradmin_zone_template_record" "ns1" {
  template_id = poweradmin_zone_template.hosting.id
  name        = "[ZONE]"
  type        = "NS"
  content     = "[NS1]"
}

resource "poweradmin_zone_template_record" "mx" {
  template_id = poweradmin_zone_template.hosting.id
  name        = "[ZONE]"
  type        = "MX"
  content     = "mail.[ZONE]"
  ttl         = 3600
  priority    = 10
}
```

`ttl` defaults to `86400` and `priority` to `0`. Template content is stored as written: unlike `poweradmin_record`, trailing dots are not normalized.

## Creating a Zone from a Template

Reference the template by name:

```hcl
resource "poweradmin_zone" "customer" {
  name     = "customer.example.com"
  type     = "MASTER"
  template = poweradmin_zone_template.hosting.name
}
```

The template is applied only when the zone is created. Setting or changing `template` replaces the zone; removing it from the configuration does not. Later changes to the template do not touch existing zones.

## Global and Shared Templates

```hcl
# Visible to all users; requires ueberuser permission
resource "poweradmin_zone_template" "global" {
  name        = "company-wide"
  description = "Mandatory records for every internal zone"
  is_global   = true
}

# Shared with user groups (Poweradmin 4.6.0+)
resource "poweradmin_zone_template" "shared" {
  name        = "team-hosting"
  description = "Hosting records the ops team can reuse"
  groups      = [poweradmin_group.ops.id]
}
```

- Toggling `is_global` reassigns ownership on the server; `owner` is `0` for a global template.
- Leave `groups` out to keep the current shares. When set, it replaces them, so list every group the template should keep.
- The owner can share only with groups they belong to; a super admin can share with any group. A global template cannot be shared.
- A group member who is not the owner must leave `groups` out.
- Poweradmin before 4.6.0 ignores `groups` with a warning.

## Looking Up Templates

```hcl
data "poweradmin_zone_template" "hosting" {
  name = "default-hosting"
}

data "poweradmin_zone_templates" "all" {}

output "hosting_records" {
  value = data.poweradmin_zone_template.hosting.records
}

output "template_names" {
  value = [for t in data.poweradmin_zone_templates.all.templates : t.name]
}
```

## Importing

```shell
# A template, by ID
terraform import poweradmin_zone_template.hosting 1

# A template record, by template_id/record_id
terraform import poweradmin_zone_template_record.soa 1/10
```
