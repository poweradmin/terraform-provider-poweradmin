# DNSSEC

Sign zones, manage their keys and read the DS records your registrar needs.

**Requires Poweradmin 4.5.0+ with the PowerDNS API configured and DNSSEC enabled on the server.**

## Overview

| Resource / Data Source | Purpose |
|------------------------|---------|
| `poweradmin_zone_dnssec` (resource) | Sign a zone through Poweradmin's signing steps |
| `poweradmin_dnssec_key` (resource) | Manage individual keys (CSK, or a KSK/ZSK pair) |
| `poweradmin_zone_dnssec` (data source) | Read whether a zone is signed, with its DS records |
| `poweradmin_dnssec_keys` (data source) | List every key of a zone |

Pick one way to sign a zone:

- **`poweradmin_zone_dnssec`** runs Poweradmin's signing steps: it checks the zone has active SOA and NS records, bumps the SOA serial, lets PowerDNS create its default keys, and rectifies the zone.
- **`poweradmin_dnssec_key`** gives you control over algorithms and sizes. In PowerDNS an active key signs the zone on its own.

> **Warning:** Destroying `poweradmin_zone_dnssec` unsigns the zone, which deletes **every** key of the zone in PowerDNS, including keys managed by `poweradmin_dnssec_key`.

## Signing a Zone with Default Keys

```hcl
resource "poweradmin_zone" "example" {
  name = "example.com"
  type = "MASTER"
}

resource "poweradmin_record" "ns" {
  zone_id = poweradmin_zone.example.id
  name    = "@"
  type    = "NS"
  content = "ns1.example.net."
  ttl     = 3600
}

resource "poweradmin_zone_dnssec" "example" {
  zone_id    = poweradmin_zone.example.id
  depends_on = [poweradmin_record.ns]
}
```

Presigned zones (DNSSEC managed at the primary server) are refused.

## Managing Keys Yourself

### A single combined signing key (CSK)

```hcl
resource "poweradmin_dnssec_key" "csk" {
  zone_id   = poweradmin_zone.example.id
  type      = "csk"
  algorithm = "ecdsa256"
  bits      = 256
}
```

### A KSK/ZSK pair

PowerDNS reports a KSK as `csk` until an active ZSK with the same algorithm exists, so create the ZSK after the KSK:

```hcl
resource "poweradmin_dnssec_key" "ksk" {
  zone_id   = poweradmin_zone.example.id
  type      = "ksk"
  algorithm = "rsasha256"
  bits      = 2048
}

resource "poweradmin_dnssec_key" "zsk" {
  zone_id    = poweradmin_zone.example.id
  type       = "zsk"
  algorithm  = "rsasha256"
  bits       = 1024
  depends_on = [poweradmin_dnssec_key.ksk]
}
```

| Algorithm | `bits` |
|-----------|--------|
| `rsasha1`, `rsasha1-nsec3-sha1`, `rsasha256`, `rsasha512` | 1024 or 2048 |
| `ecdsa256`, `ed25519` | 256 |
| `ecdsa384` | 384 |
| `ed448` | 456 |

### Key rollover

Pre-publish the next key inactive, then set `active = true` when it is time to switch:

```hcl
resource "poweradmin_dnssec_key" "next" {
  zone_id   = poweradmin_zone.example.id
  type      = "csk"
  algorithm = "ecdsa256"
  bits      = 256
  active    = false
}
```

`active` defaults to `true` in this provider (the API and web UI default to inactive).

## DS Records for the Registrar

From the signing resource:

```hcl
output "ds_records" {
  value = [for ds in poweradmin_zone_dnssec.example.ds_records : "${ds.key_tag} ${ds.algorithm} ${ds.digest_type} ${ds.digest}"]
}
```

From a key (zone-file form, `keytag algorithm digest_type digest`):

```hcl
output "csk_ds" {
  value = poweradmin_dnssec_key.csk.ds
}
```

PowerDNS exports DS records only for keys it currently counts as KSK or CSK, so a ZSK has none while an active KSK of its algorithm exists.

## Reading DNSSEC State of Existing Zones

```hcl
data "poweradmin_zone_dnssec" "example" {
  zone_id = 10
}

data "poweradmin_dnssec_keys" "example" {
  zone_id = 10
}

output "signed" {
  value = data.poweradmin_zone_dnssec.example.enabled
}

output "active_ds_records" {
  value = flatten([for k in data.poweradmin_dnssec_keys.example.keys : k.ds if k.active])
}
```

`poweradmin_dnssec_keys` also lists the default keys PowerDNS created when the zone was signed.

## Importing

```shell
# Signing state, by zone ID
terraform import poweradmin_zone_dnssec.example 10

# A key, by zone_id/key_id
terraform import poweradmin_dnssec_key.csk 10/3
```
