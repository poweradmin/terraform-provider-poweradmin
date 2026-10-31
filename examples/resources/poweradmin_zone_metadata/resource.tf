resource "poweradmin_zone" "example" {
  name = "example.com"
  type = "MASTER"
}

# Allow zone transfers to a secondary and to the zone's own NS records
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

# A single-value kind; Poweradmin refuses a second value at apply
resource "poweradmin_zone_metadata" "axfr_source" {
  zone_id = poweradmin_zone.example.id
  kind    = "AXFR-SOURCE"
  values  = ["192.0.2.1"]
}
