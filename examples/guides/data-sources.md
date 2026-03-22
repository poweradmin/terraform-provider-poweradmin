# Data Sources

Data sources let you read existing Poweradmin objects without managing them. Use them to reference resources created outside of Terraform or in other Terraform configurations.

## Zone Data Source

Look up a zone by name or ID:

```hcl
# By name
data "poweradmin_zone" "existing" {
  name = "existing.example.com"
}

# By ID
data "poweradmin_zone" "by_id" {
  id = "42"
}

output "zone_type" {
  value = data.poweradmin_zone.existing.type
}
```

**Returned attributes:** `id`, `name`, `type`, `masters`, `account`, `description`

## Records Data Source

List DNS records in a zone, with optional type filtering:

```hcl
# All records in a zone
data "poweradmin_records" "all" {
  zone_id = data.poweradmin_zone.existing.id
}

# Only A records
data "poweradmin_records" "a_records" {
  zone_id = data.poweradmin_zone.existing.id
  type    = "A"
}

# Only MX records
data "poweradmin_records" "mx_records" {
  zone_id = data.poweradmin_zone.existing.id
  type    = "MX"
}
```

**Returned attributes per record:** `id`, `name`, `type`, `content`, `ttl`, `priority`, `disabled`

## RRSets Data Source

List Resource Record Sets in a zone:

```hcl
# All RRSets
data "poweradmin_rrsets" "all" {
  zone_id = data.poweradmin_zone.existing.id
}

# Only A-type RRSets
data "poweradmin_rrsets" "a_rrsets" {
  zone_id = data.poweradmin_zone.existing.id
  type    = "A"
}
```

**Returned attributes per RRSet:** `name`, `type`, `ttl`, `records` (list of content/disabled/priority)

## Permission Data Source

Look up permission templates to use when creating users:

```hcl
# By name
data "poweradmin_permission" "zone_edit" {
  name = "zone_content_edit_own"
}

# By ID
data "poweradmin_permission" "by_id" {
  id = 1
}

# Use with a user resource
resource "poweradmin_user" "editor" {
  username   = "zone.editor"
  fullname   = "Zone Editor"
  email      = "editor@example.com"
  password   = var.editor_password
  perm_templ = data.poweradmin_permission.zone_edit.id
}
```

**Returned attributes:** `id`, `name`, `description`

## Group Data Source (Poweradmin 4.2.0+)

Look up a group by name or ID:

```hcl
data "poweradmin_group" "ops" {
  name = "operations"
}

output "group_members" {
  value = data.poweradmin_group.ops.member_count
}

output "group_zones" {
  value = data.poweradmin_group.ops.zone_count
}
```

**Returned attributes:** `id`, `name`, `description`, `perm_templ_id`, `member_count`, `zone_count`

## Common Patterns

### Reference a zone from another state

```hcl
# In your DNS records configuration, reference a zone managed elsewhere
data "poweradmin_zone" "shared" {
  name = "shared.example.com"
}

resource "poweradmin_record" "my_service" {
  zone_id = data.poweradmin_zone.shared.id
  name    = "my-service"
  type    = "A"
  content = "10.0.1.50"
  ttl     = 300
}
```

### Audit existing records

```hcl
data "poweradmin_zone" "audit_target" {
  name = "example.com"
}

data "poweradmin_records" "all_records" {
  zone_id = data.poweradmin_zone.audit_target.id
}

output "record_count" {
  value = length(data.poweradmin_records.all_records.records)
}
```
