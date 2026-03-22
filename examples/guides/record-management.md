# DNS Record Management

This guide covers managing individual DNS records. For managing multiple records with the same name and type as a group, see [RRSet Management](rrset-management.md).

## Supported Record Types

| Type | Description | Priority | Example Content |
|------|-------------|----------|-----------------|
| `A` | IPv4 address | No | `192.0.2.100` |
| `AAAA` | IPv6 address | No | `2001:db8::1` |
| `CNAME` | Canonical name alias | No | `www.example.com.` |
| `MX` | Mail exchange | Yes | `mail.example.com.` |
| `TXT` | Text record | No | `"v=spf1 include:_spf.google.com ~all"` |
| `SRV` | Service locator | Yes | `0 5 5060 sip.example.com.` |
| `NS` | Name server | No | `ns1.example.com.` |
| `PTR` | Pointer (reverse DNS) | No | `host.example.com.` |
| `CAA` | Certificate Authority Authorization | No | `0 issue "letsencrypt.org"` |
| `SOA` | Start of Authority | No | Typically auto-managed |

## Basic Records

```hcl
resource "poweradmin_zone" "example" {
  name = "example.com"
  type = "MASTER"
}

# A record
resource "poweradmin_record" "www" {
  zone_id = poweradmin_zone.example.id
  name    = "www"
  type    = "A"
  content = "192.0.2.100"
  ttl     = 3600
}

# AAAA record
resource "poweradmin_record" "www_ipv6" {
  zone_id = poweradmin_zone.example.id
  name    = "www"
  type    = "AAAA"
  content = "2001:db8::1"
  ttl     = 3600
}

# CNAME record
resource "poweradmin_record" "blog" {
  zone_id = poweradmin_zone.example.id
  name    = "blog"
  type    = "CNAME"
  content = "www.example.com."
  ttl     = 7200
}
```

## Records with Priority

MX and SRV records support a `priority` field. Lower values indicate higher priority.

```hcl
# Primary mail server
resource "poweradmin_record" "mx_primary" {
  zone_id  = poweradmin_zone.example.id
  name     = "@"
  type     = "MX"
  content  = "mail1.example.com."
  ttl      = 3600
  priority = 10
}

# Backup mail server
resource "poweradmin_record" "mx_backup" {
  zone_id  = poweradmin_zone.example.id
  name     = "@"
  type     = "MX"
  content  = "mail2.example.com."
  ttl      = 3600
  priority = 20
}

# SRV record for SIP
resource "poweradmin_record" "sip_srv" {
  zone_id  = poweradmin_zone.example.id
  name     = "_sip._tcp"
  type     = "SRV"
  content  = "0 5 5060 sip.example.com."
  ttl      = 3600
  priority = 10
}
```

## TXT Records

```hcl
# SPF record
resource "poweradmin_record" "spf" {
  zone_id = poweradmin_zone.example.id
  name    = "@"
  type    = "TXT"
  content = "\"v=spf1 include:_spf.google.com ~all\""
  ttl     = 3600
}

# DKIM record
resource "poweradmin_record" "dkim" {
  zone_id = poweradmin_zone.example.id
  name    = "default._domainkey"
  type    = "TXT"
  content = "\"v=DKIM1; k=rsa; p=MIGfMA0GCSqGSIb...\""
  ttl     = 3600
}

# DMARC record
resource "poweradmin_record" "dmarc" {
  zone_id = poweradmin_zone.example.id
  name    = "_dmarc"
  type    = "TXT"
  content = "\"v=DMARC1; p=reject; rua=mailto:dmarc@example.com\""
  ttl     = 3600
}
```

## CAA Records

Certificate Authority Authorization restricts which CAs can issue certificates for your domain.

```hcl
resource "poweradmin_record" "caa" {
  zone_id = poweradmin_zone.example.id
  name    = "@"
  type    = "CAA"
  content = "0 issue \"letsencrypt.org\""
  ttl     = 3600
}
```

## Disabled Records

Records can be disabled without deleting them. Disabled records are not served by PowerDNS.

```hcl
resource "poweradmin_record" "maintenance" {
  zone_id  = poweradmin_zone.example.id
  name     = "old-service"
  type     = "A"
  content  = "192.0.2.200"
  ttl      = 3600
  disabled = true
}
```

## TTL Defaults

If `ttl` is not specified, it defaults to `3600` (1 hour). Common TTL values:

| TTL (seconds) | Duration | Use Case |
|------|----------|----------|
| 60 | 1 minute | Failover records, during migrations |
| 300 | 5 minutes | Frequently changing records |
| 3600 | 1 hour | Standard records (default) |
| 86400 | 1 day | Stable records (MX, NS) |

## Querying Records

Use the `poweradmin_records` data source to list records in a zone:

```hcl
# All records
data "poweradmin_records" "all" {
  zone_id = poweradmin_zone.example.id
}

# Only A records
data "poweradmin_records" "a_only" {
  zone_id = poweradmin_zone.example.id
  type    = "A"
}
```

## Importing Records

Records are imported using the format `zone_id/record_id`:

```bash
terraform import poweradmin_record.www 1/42
```
