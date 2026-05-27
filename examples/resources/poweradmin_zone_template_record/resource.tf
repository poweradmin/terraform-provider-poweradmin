# A zone template populated with placeholder records that get substituted
# (e.g. [ZONE], [NS1], [HOSTMASTER], [SERIAL]) when a zone is created from it.
resource "poweradmin_zone_template" "hosting" {
  name        = "default-hosting"
  description = "Default record set for hosting customers"
}

resource "poweradmin_zone_template_record" "soa" {
  template_id = poweradmin_zone_template.hosting.id
  name        = "[ZONE]"
  type        = "SOA"
  content     = "[NS1] [HOSTMASTER] [SERIAL] 28800 7200 604800 86400"
  ttl         = 86400
}

resource "poweradmin_zone_template_record" "ns1" {
  template_id = poweradmin_zone_template.hosting.id
  name        = "[ZONE]"
  type        = "NS"
  content     = "[NS1]"
  ttl         = 86400
}

resource "poweradmin_zone_template_record" "ns2" {
  template_id = poweradmin_zone_template.hosting.id
  name        = "[ZONE]"
  type        = "NS"
  content     = "[NS2]"
  ttl         = 86400
}

resource "poweradmin_zone_template_record" "mx" {
  template_id = poweradmin_zone_template.hosting.id
  name        = "[ZONE]"
  type        = "MX"
  content     = "mail.[ZONE]"
  ttl         = 3600
  priority    = 10
}
