# Zone Management

This guide covers all aspects of managing DNS zones with the Poweradmin Terraform provider.

## Zone Types

Poweradmin supports three zone types:

| Type | Description | Use Case |
|------|-------------|----------|
| `MASTER` | Primary authoritative zone | You manage the zone's records directly |
| `SLAVE` | Secondary zone replicated from a master | Redundancy; records are synced from master nameservers |
| `NATIVE` | Zone without AXFR-based replication | Used with database-backed replication or single-server setups |

## Creating a Master Zone

```hcl
resource "poweradmin_zone" "example_com" {
  name        = "example.com"
  type        = "MASTER"
  description = "Production DNS zone"
}
```

## Creating a Slave Zone

Slave zones require one or more master nameservers to replicate from. Multiple masters are comma-separated.

```hcl
# Basic slave with two masters
resource "poweradmin_zone" "slave_basic" {
  name    = "slave.example.com"
  type    = "SLAVE"
  masters = "192.0.2.1,192.0.2.2"
}

# Slave with masters on custom ports
resource "poweradmin_zone" "slave_custom_ports" {
  name    = "slave-ports.example.com"
  type    = "SLAVE"
  masters = "192.0.2.1:5300,192.0.2.2:5300"
}

# Slave with IPv6 masters (brackets required for ports)
resource "poweradmin_zone" "slave_ipv6" {
  name    = "slave-ipv6.example.com"
  type    = "SLAVE"
  masters = "[2001:db8::1]:5300,[2001:db8::2]:5300"
}

# Mixed IPv4 and IPv6 masters
resource "poweradmin_zone" "slave_mixed" {
  name    = "slave-mixed.example.com"
  type    = "SLAVE"
  masters = "192.0.2.1:5300,[2001:db8::1]:5300"
}
```

## Creating a Zone from a Template

Templates pre-populate a zone with default records (SOA, NS, etc.) during creation. The template name must match a template defined in your Poweradmin instance.

```hcl
resource "poweradmin_zone" "templated" {
  name     = "templated.example.com"
  type     = "MASTER"
  template = "default-template"
}
```

> **Note:** The `template` attribute is only used during creation. Changing it later has no effect on existing records.

## Zone with Account

Accounts let you organize zones by customer or department.

```hcl
resource "poweradmin_zone" "customer_zone" {
  name        = "customer.example.com"
  type        = "MASTER"
  account     = "customer-001"
  description = "Customer DNS zone"
}
```

## Looking Up Existing Zones

Use the data source to reference zones not managed by Terraform:

```hcl
# By name
data "poweradmin_zone" "existing" {
  name = "existing.example.com"
}

# By ID
data "poweradmin_zone" "by_id" {
  id = "42"
}

# Use in other resources
resource "poweradmin_record" "api" {
  zone_id = data.poweradmin_zone.existing.id
  name    = "api"
  type    = "A"
  content = "192.0.2.50"
  ttl     = 3600
}
```

## Importing Existing Zones

```bash
# Import by numeric ID
terraform import poweradmin_zone.example_com 1

# Import by zone name
terraform import poweradmin_zone.example_com example.com
```

## Full Example: Multi-Environment DNS

```hcl
variable "environments" {
  default = ["dev", "staging", "prod"]
}

resource "poweradmin_zone" "env_zones" {
  for_each    = toset(var.environments)
  name        = "${each.key}.example.com"
  type        = "MASTER"
  description = "${each.key} environment DNS"
}
```
