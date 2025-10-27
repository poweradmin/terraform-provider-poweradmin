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
