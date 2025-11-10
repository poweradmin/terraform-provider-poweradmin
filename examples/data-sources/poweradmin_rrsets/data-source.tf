# List all RRSets in a zone
data "poweradmin_rrsets" "all_records" {
  zone_id = poweradmin_zone.example_com.id
}

# Filter RRSets by type (only MX records)
data "poweradmin_rrsets" "mx_only" {
  zone_id = poweradmin_zone.example_com.id
  type    = "MX"
}

# Filter RRSets by type (only A records)
data "poweradmin_rrsets" "a_records" {
  zone_id = poweradmin_zone.example_com.id
  type    = "A"
}

# Output the total number of RRSets
output "total_rrsets" {
  value       = length(data.poweradmin_rrsets.all_records.rrsets)
  description = "Total number of RRSets in the zone"
}

# Output MX records
output "mx_records" {
  value       = data.poweradmin_rrsets.mx_only.rrsets
  description = "All MX RRSets with their records and priorities"
}

# Output A record IPs
output "a_record_ips" {
  value = flatten([
    for rrset in data.poweradmin_rrsets.a_records.rrsets : [
      for record in rrset.records : record.content
    ]
  ])
  description = "List of all A record IP addresses"
}
