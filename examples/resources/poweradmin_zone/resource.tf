# Create a MASTER zone
resource "poweradmin_zone" "example_com" {
  name        = "example.com"
  type        = "MASTER"
  description = "Example zone managed by Terraform"
}

# Create a SLAVE zone with master nameservers
resource "poweradmin_zone" "slave_example" {
  name    = "slave.example.com"
  type    = "SLAVE"
  masters = "192.0.2.1,192.0.2.2"
}

# Create a SLAVE zone with masters on custom ports
resource "poweradmin_zone" "slave_with_ports" {
  name    = "slave-ports.example.com"
  type    = "SLAVE"
  masters = "192.0.2.1:5300,192.0.2.2:5300"
}

# Create a SLAVE zone with IPv6 masters
resource "poweradmin_zone" "slave_ipv6" {
  name    = "slave-ipv6.example.com"
  type    = "SLAVE"
  masters = "[2001:db8::1]:5300,[2001:db8::2]:5300"
}

# Create a SLAVE zone with mixed IPv4 and IPv6 masters
resource "poweradmin_zone" "slave_mixed" {
  name    = "slave-mixed.example.com"
  type    = "SLAVE"
  masters = "192.0.2.1:5300,[2001:db8::1]:5300"
}

# Create a zone with account
resource "poweradmin_zone" "customer_zone" {
  name        = "customer.example.com"
  type        = "MASTER"
  account     = "customer-001"
  description = "Customer DNS zone"
}

# Create a zone from a template
resource "poweradmin_zone" "templated_zone" {
  name     = "templated.example.com"
  type     = "MASTER"
  template = "default-template"
}
