# Zone Metadata

Manage PowerDNS per-zone metadata such as `ALLOW-AXFR-FROM`, `ALSO-NOTIFY` and `AXFR-SOURCE`.

**Requires Poweradmin 4.3.0+**

## Overview

| Resource / Data Source | Purpose |
|------------------------|---------|
| `poweradmin_zone_metadata` (resource) | Manage one metadata kind of a zone with all of its values |
| `poweradmin_zone_metadata` (data source) | Read the values of one kind |
| `poweradmin_zone_metadata_entries` (data source) | List every kind of a zone with its values |

Each `poweradmin_zone_metadata` resource owns one kind. Creating it **replaces** any values the kind already has.

## Zone Transfers and Notifies

```hcl
resource "poweradmin_zone" "example" {
  name = "example.com"
  type = "MASTER"
}

# Allow transfers to a secondary and to the zone's own NS records
resource "poweradmin_zone_metadata" "allow_axfr" {
  zone_id = poweradmin_zone.example.id
  kind    = "ALLOW-AXFR-FROM"
  values  = ["192.0.2.10", "AUTO-NS"]
}

# Send NOTIFY messages to extra servers
resource "poweradmin_zone_metadata" "also_notify" {
  zone_id = poweradmin_zone.example.id
  kind    = "ALSO-NOTIFY"
  values  = ["192.0.2.10"]
}

# A single-value kind
resource "poweradmin_zone_metadata" "axfr_source" {
  zone_id = poweradmin_zone.example.id
  kind    = "AXFR-SOURCE"
  values  = ["192.0.2.1"]
}
```

- `kind` is upper case, at most 32 characters.
- `values` is a set with at least one value. Values must not start or end with whitespace, since Poweradmin trims them.
- A kind that only takes effect next to another kind needs that kind first, so add `depends_on`.

## What Poweradmin Refuses

Poweradmin checks values at apply time and refuses:

- kinds that PowerDNS maintains itself,
- more than one value for a single-value kind,
- values outside a kind's vocabulary,
- operator-only kinds for non-administrators,
- on the PowerDNS API backend, custom kinds without the `X-` prefix.

Before Poweradmin 4.5.0 these kinds are refused as read-only: `SOA-EDIT`, `API-RECTIFY`, `NSEC3PARAM`, `NSEC3NARROW`, `PRESIGNED`, `AXFR-MASTER-TSIG`, `LUA-AXFR-SCRIPT`, `ENABLE-LUA-RECORDS`.

> **Note:** Do not manage `SOA-EDIT-API` here together with `poweradmin_zone.soa_edit_api`.

> **Note:** On the PowerDNS API backend, `API-RECTIFY` and `NSEC3NARROW` set to `0` cannot be read back. The resource keeps `0` while the zone exists, so a change made outside Terraform is not detected, and importing such a kind fails.

## Reading Metadata

```hcl
# One kind; reading a kind that has no values is an error
data "poweradmin_zone_metadata" "axfr" {
  zone_id = 10
  kind    = "ALLOW-AXFR-FROM"
}

# Every kind, as a map
data "poweradmin_zone_metadata_entries" "all" {
  zone_id = 10
}

output "metadata" {
  value = { for e in data.poweradmin_zone_metadata_entries.all.entries : e.kind => e.values }
}
```

## Importing

```shell
# By zone_id/kind
terraform import poweradmin_zone_metadata.allow_axfr 10/ALLOW-AXFR-FROM
```
