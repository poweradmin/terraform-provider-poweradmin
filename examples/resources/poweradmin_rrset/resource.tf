# Create an RRSet with multiple A records (load balancing)
resource "poweradmin_rrset" "web_servers" {
  zone_id = poweradmin_zone.example_com.id
  name    = "www"
  type    = "A"
  ttl     = 300

  records = [
    {
      content  = "192.0.2.10"
      disabled = false
      priority = 0
    },
    {
      content  = "192.0.2.11"
      disabled = false
      priority = 0
    },
    {
      content  = "192.0.2.12"
      disabled = false
      priority = 0
    },
  ]
}

# Create an RRSet with MX records (with priorities)
resource "poweradmin_rrset" "mail_servers" {
  zone_id = poweradmin_zone.example_com.id
  name    = "@"
  type    = "MX"
  ttl     = 3600

  records = [
    {
      content  = "mail1.example.com"
      priority = 10
      disabled = false
    },
    {
      content  = "mail2.example.com"
      priority = 20
      disabled = false
    },
  ]
}

# Create an RRSet with AAAA records (IPv6)
resource "poweradmin_rrset" "ipv6" {
  zone_id = poweradmin_zone.example_com.id
  name    = "www"
  type    = "AAAA"
  ttl     = 3600

  records = [
    {
      content  = "2001:db8::1"
      disabled = false
      priority = 0
    },
    {
      content  = "2001:db8::2"
      disabled = false
      priority = 0
    },
  ]
}

# Create an RRSet with TXT record
resource "poweradmin_rrset" "spf" {
  zone_id = poweradmin_zone.example_com.id
  name    = "@"
  type    = "TXT"
  ttl     = 3600

  records = [
    {
      content  = "v=spf1 include:_spf.example.com ~all"
      disabled = false
      priority = 0
    },
  ]
}

# Create an RRSet with NS records (subdomain delegation)
resource "poweradmin_rrset" "subdomain_ns" {
  zone_id = poweradmin_zone.example_com.id
  name    = "subdomain"
  type    = "NS"
  ttl     = 86400

  records = [
    {
      content  = "ns1.subdomain-provider.example"
      disabled = false
      priority = 0
    },
    {
      content  = "ns2.subdomain-provider.example"
      disabled = false
      priority = 0
    },
  ]
}
