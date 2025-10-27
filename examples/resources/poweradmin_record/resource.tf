# Create an A record
resource "poweradmin_record" "www" {
  zone_id = poweradmin_zone.example_com.id
  name    = "www"
  type    = "A"
  content = "192.0.2.100"
  ttl     = 3600
}

# Create an AAAA record
resource "poweradmin_record" "www_ipv6" {
  zone_id = poweradmin_zone.example_com.id
  name    = "www"
  type    = "AAAA"
  content = "2001:db8::1"
  ttl     = 3600
}

# Create a CNAME record
resource "poweradmin_record" "blog" {
  zone_id = poweradmin_zone.example_com.id
  name    = "blog"
  type    = "CNAME"
  content = "www.example.com."
  ttl     = 7200
}

# Create an MX record with priority
resource "poweradmin_record" "mail" {
  zone_id  = poweradmin_zone.example_com.id
  name     = "@"
  type     = "MX"
  content  = "mail.example.com."
  ttl      = 3600
  priority = 10
}

# Create a TXT record
resource "poweradmin_record" "spf" {
  zone_id = poweradmin_zone.example_com.id
  name    = "@"
  type    = "TXT"
  content = "v=spf1 mx -all"
  ttl     = 3600
}

# Create a disabled record
resource "poweradmin_record" "maintenance" {
  zone_id  = poweradmin_zone.example_com.id
  name     = "maintenance"
  type     = "A"
  content  = "192.0.2.200"
  ttl      = 300
  disabled = true
}
