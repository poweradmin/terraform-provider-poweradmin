# Sign a zone with PowerDNS's default keys. The zone needs active SOA and NS records.
# Destroying this resource unsigns the zone and deletes all of its keys.
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

# Parsed DS records for the registrar
output "ds_records" {
  value = [for ds in poweradmin_zone_dnssec.example.ds_records : "${ds.key_tag} ${ds.algorithm} ${ds.digest_type} ${ds.digest}"]
}
