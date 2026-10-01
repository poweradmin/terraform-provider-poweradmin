# List every DNSSEC key of a zone, including keys PowerDNS created when the zone was signed
data "poweradmin_dnssec_keys" "example" {
  zone_id = 10
}

output "active_ds_records" {
  value = flatten([for k in data.poweradmin_dnssec_keys.example.keys : k.ds if k.active])
}
