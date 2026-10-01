# Sign a zone with a single combined signing key (CSK). An active key signs the zone on its own.
resource "poweradmin_zone" "example" {
  name = "example.com"
  type = "MASTER"
}

resource "poweradmin_dnssec_key" "csk" {
  zone_id   = poweradmin_zone.example.id
  type      = "csk"
  algorithm = "ecdsa256"
  bits      = 256
}

# Or a split KSK/ZSK pair. PowerDNS reports a KSK as "csk" until an active ZSK
# with the same algorithm exists, so create the ZSK after the KSK.
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

# A pre-published key for a rollover: created inactive, activated later by setting active = true
resource "poweradmin_dnssec_key" "next" {
  zone_id   = poweradmin_zone.example.id
  type      = "csk"
  algorithm = "ecdsa256"
  bits      = 256
  active    = false
}

# DS records to submit to the registrar
output "ds_records" {
  value = poweradmin_dnssec_key.csk.ds
}
