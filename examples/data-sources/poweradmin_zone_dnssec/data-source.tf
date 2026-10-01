# Read whether a zone is signed, with the DS records for the registrar
data "poweradmin_zone_dnssec" "example" {
  zone_id = 10
}

output "signed" {
  value = data.poweradmin_zone_dnssec.example.enabled
}

output "ds_records" {
  value = data.poweradmin_zone_dnssec.example.ds_records
}
