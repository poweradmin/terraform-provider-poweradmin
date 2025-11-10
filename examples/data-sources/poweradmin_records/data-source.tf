# List all records in a zone
data "poweradmin_records" "all" {
  zone_id = poweradmin_zone.example_com.id
}

# Filter records by type
data "poweradmin_records" "mx_records" {
  zone_id = poweradmin_zone.example_com.id
  type    = "MX"
}

# Filter records by type (A records)
data "poweradmin_records" "a_records" {
  zone_id = poweradmin_zone.example_com.id
  type    = "A"
}

# Output total record count
output "total_records" {
  value       = length(data.poweradmin_records.all.records)
  description = "Total number of records in the zone"
}

# Output MX records with priorities
output "mx_records_with_priority" {
  value = [
    for record in data.poweradmin_records.mx_records.records :
    "${record.priority} ${record.content}"
  ]
  description = "List of MX records with priorities"
}

# Output A record IPs
output "a_record_ips" {
  value = [
    for record in data.poweradmin_records.a_records.records : record.content
  ]
  description = "List of all A record IP addresses"
}
