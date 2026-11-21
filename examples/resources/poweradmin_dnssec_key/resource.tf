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

# Import an existing private key, e.g. when moving a signed zone without a rollover
# (Poweradmin 4.6.0+, Terraform/OpenTofu 1.11+). The key is write-only and never stored
# in state; algorithm and bits are read from it. Bump the version to import another key.
resource "poweradmin_dnssec_key" "imported" {
  zone_id                = poweradmin_zone.example.id
  type                   = "ksk"
  private_key_wo         = file("Kexample.com.+013+12345.private")
  private_key_wo_version = 1
}

# DS records to submit to the registrar
output "ds_records" {
  value = poweradmin_dnssec_key.csk.ds
}
