# RRSet Management

RRSets (Resource Record Sets) manage multiple DNS records that share the same name and type as a single unit. This matches how PowerDNS stores records internally and is the recommended approach when you need multiple records of the same name/type.

## When to Use RRSets vs Individual Records

| Scenario | Use |
|----------|-----|
| Single A record for `www` | `poweradmin_record` |
| Multiple A records for `www` (load balancing) | `poweradmin_rrset` |
| Single MX record | `poweradmin_record` |
| Multiple MX records with priorities | `poweradmin_rrset` |
| Mix of different record types | `poweradmin_record` for each |

> **Important:** Do not mix `poweradmin_record` and `poweradmin_rrset` for the same name/type combination. They will conflict.

## Basic RRSet

```hcl
resource "poweradmin_zone" "example" {
  name = "example.com"
  type = "MASTER"
}

# Round-robin A records for load balancing
resource "poweradmin_rrset" "web_servers" {
  zone_id = poweradmin_zone.example.id
  name    = "www"
  type    = "A"
  ttl     = 300

  records {
    content = "192.0.2.10"
  }

  records {
    content = "192.0.2.11"
  }

  records {
    content = "192.0.2.12"
  }
}
```

## RRSet with Priorities

MX and SRV records support priority. Lower values = higher priority.

```hcl
# Mail servers with failover
resource "poweradmin_rrset" "mail" {
  zone_id = poweradmin_zone.example.id
  name    = "@"
  type    = "MX"
  ttl     = 3600

  records {
    content  = "mail1.example.com."
    priority = 10
  }

  records {
    content  = "mail2.example.com."
    priority = 20
  }

  records {
    content  = "mail-backup.example.com."
    priority = 30
  }
}
```

## Partially Disabled Records

Individual records within an RRSet can be disabled without removing them. This is useful for maintenance windows.

```hcl
resource "poweradmin_rrset" "web_servers" {
  zone_id = poweradmin_zone.example.id
  name    = "www"
  type    = "A"
  ttl     = 300

  records {
    content = "192.0.2.10"
  }

  records {
    content  = "192.0.2.11"
    disabled = true  # Under maintenance
  }

  records {
    content = "192.0.2.12"
  }
}
```

## Atomic Updates

RRSet updates are atomic. When you change any record in the set, the entire RRSet is replaced in a single API call. This prevents inconsistent states where some records are updated and others are not.

## Querying RRSets

```hcl
# List all RRSets in a zone
data "poweradmin_rrsets" "all" {
  zone_id = poweradmin_zone.example.id
}

# Filter by type
data "poweradmin_rrsets" "mx_records" {
  zone_id = poweradmin_zone.example.id
  type    = "MX"
}
```

## Importing RRSets

RRSets are imported using the format `zone_id/name/type`:

```bash
terraform import poweradmin_rrset.web_servers 1/www/A
terraform import poweradmin_rrset.mail 1/@/MX
```
