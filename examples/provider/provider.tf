# Example using API Key authentication (recommended)
provider "poweradmin" {
  api_url = "https://dns.example.com"
  api_key = var.poweradmin_api_key
}

# Example using Basic Authentication
# provider "poweradmin" {
#   api_url  = "https://dns.example.com"
#   username = var.poweradmin_username
#   password = var.poweradmin_password
# }

# Example specifying API version (optional - defaults to v2)
# provider "poweradmin" {
#   api_url     = "https://dns.example.com"
#   api_key     = var.poweradmin_api_key
#   api_version = "v2"  # Only v2 is supported (Poweradmin 4.1.0+)
# }

# Example for development with insecure TLS (not recommended for production)
# provider "poweradmin" {
#   api_url  = "http://localhost:8080"
#   api_key  = "test-key"
#   insecure = true
# }
