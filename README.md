# Terraform Provider for Poweradmin

Manage DNS zones and records in [Poweradmin](https://www.poweradmin.org/) using Terraform or OpenTofu.

## Features

- **Zone Management**: Create, update, and delete DNS zones (MASTER, SLAVE, NATIVE types)
- **Record Management**: Full CRUD operations for DNS records (A, AAAA, CNAME, MX, TXT, SRV, NS, PTR, and more)
- **Data Sources**: Query existing zones and records
- **Dual Authentication**: Support for API key and HTTP basic authentication
- **Version Support**: Compatible with Poweradmin 4.0.x (stable) and master (development)
- **OpenTofu Compatible**: Works seamlessly with both Terraform and OpenTofu

## Requirements

- [Terraform](https://developer.hashicorp.com/terraform/downloads) >= 1.0 **OR**
- [OpenTofu](https://opentofu.org/docs/intro/install/) >= 1.6
- [Go](https://golang.org/doc/install) >= 1.24 (for development only)
- Poweradmin instance with API enabled

## Installation

### From Terraform/OpenTofu Registry

The provider is available on both:
- **Terraform Registry**: https://registry.terraform.io/providers/poweradmin/poweradmin
- **OpenTofu Registry**: https://search.opentofu.org/provider/poweradmin/poweradmin

```hcl
terraform {
  required_providers {
    poweradmin = {
      source  = "poweradmin/poweradmin"
      version = "~> 0.1"
    }
  }
}
```

### Local Development Installation

```bash
git clone https://github.com/poweradmin/terraform-provider-poweradmin.git
cd terraform-provider-poweradmin
make install
```

## Usage

### Provider Configuration

```hcl
# Using API Key authentication (recommended)
provider "poweradmin" {
  api_url = "https://dns.example.com"
  api_key = var.poweradmin_api_key
}

# Using Basic Authentication
provider "poweradmin" {
  api_url  = "https://dns.example.com"
  username = var.poweradmin_username
  password = var.poweradmin_password
}

# Using development version (master branch)
provider "poweradmin" {
  api_url     = "https://dns.example.com"
  api_key     = var.poweradmin_api_key
  api_version = "dev"  # Use "v1" for stable (4.0.x) or "dev" for master
}
```

### Creating a DNS Zone

```hcl
resource "poweradmin_zone" "example_com" {
  name        = "example.com"
  type        = "MASTER"
  description = "Example zone managed by Terraform"
}

resource "poweradmin_zone" "slave_zone" {
  name    = "slave.example.com"
  type    = "SLAVE"
  masters = "192.0.2.1,192.0.2.2"
}
```

### Creating DNS Records

```hcl
# A record
resource "poweradmin_record" "www" {
  zone_id = poweradmin_zone.example_com.id
  name    = "www"
  type    = "A"
  content = "192.0.2.100"
  ttl     = 3600
}

# CNAME record
resource "poweradmin_record" "blog" {
  zone_id = poweradmin_zone.example_com.id
  name    = "blog"
  type    = "CNAME"
  content = "www.example.com."
  ttl     = 7200
}

# MX record with priority
resource "poweradmin_record" "mail" {
  zone_id  = poweradmin_zone.example_com.id
  name     = "@"
  type     = "MX"
  content  = "mail.example.com."
  ttl      = 3600
  priority = 10
}
```

### Using Data Sources

```hcl
# Look up an existing zone
data "poweradmin_zone" "existing" {
  name = "existing.example.com"
}

# Use the zone in a resource
resource "poweradmin_record" "api" {
  zone_id = data.poweradmin_zone.existing.id
  name    = "api"
  type    = "A"
  content = "192.0.2.50"
  ttl     = 3600
}
```

## Supported Resources

- `poweradmin_zone` - Manages DNS zones
- `poweradmin_record` - Manages DNS records

## Supported Data Sources

- `poweradmin_zone` - Query zone information by ID or name

## Provider Configuration Options

| Argument | Type | Required | Description |
|----------|------|----------|-------------|
| `api_url` | string | Yes | Poweradmin API base URL (e.g., `https://dns.example.com`) |
| `api_key` | string | No* | API key for X-API-Key authentication (recommended) |
| `username` | string | No* | Username for HTTP basic authentication |
| `password` | string | No* | Password for HTTP basic authentication |
| `api_version` | string | No | API version: `v1` (default, for 4.0.x) or `dev` (for master) |
| `insecure` | bool | No | Skip TLS certificate verification (default: `false`, not recommended for production) |

\* Either `api_key` OR both `username` and `password` must be provided

## Poweradmin API Setup

To use this provider, you need to enable the Poweradmin API. Edit your `config/settings.php`:

```php
'api' => [
    'enabled' => true,
    'basic_auth_enabled' => true,  // For basic auth
]
```

### Creating an API Key (Recommended)

1. Log into Poweradmin as an administrator
2. Navigate to API Keys management
3. Create a new API key for Terraform
4. Store the key securely (e.g., in environment variables or secret management)

```bash
export TF_VAR_poweradmin_api_key="your-api-key-here"
```

## Development

### Building the Provider

```bash
make build
```

### Running Tests

```bash
# Unit tests
make test

# Acceptance tests (requires running Poweradmin instance)
export TF_ACC=1
export POWERADMIN_API_URL="http://localhost:8080"
export POWERADMIN_API_KEY="test-api-key"
make testacc
```

### Generating Documentation

```bash
make generate
```

This will update the `docs/` directory with auto-generated documentation from the schema definitions.

## OpenTofu Compatibility

This provider is built using the Terraform Plugin Framework and is fully compatible with both:

- **Terraform** by HashiCorp (1.0+)
- **OpenTofu** by the OpenTofu Foundation (1.6+)

No special code or configuration is required for dual compatibility. Users can use this provider with either tool interchangeably. The provider will work identically in both environments as they share the same plugin protocol and framework.

## Documentation

- [Provider Documentation](docs/index.md)
- [Poweradmin Documentation](https://docs.poweradmin.org/)
- [Poweradmin API Documentation](https://docs.poweradmin.org/configuration/api/)
- [Poweradmin GitHub](https://github.com/poweradmin/poweradmin)

## Contributing

Contributions are welcome! Please see [CONTRIBUTING.md](CONTRIBUTING.md) for details on:
- Setting up the development environment
- Running tests
- Submitting pull requests
- Code style guidelines

## License

This project is licensed under the MPL-2.0 License - see the [LICENSE](LICENSE) file for details.

## Support

- **Issues**: [GitHub Issues](https://github.com/poweradmin/terraform-provider-poweradmin/issues)
- **Discussions**: [GitHub Discussions](https://github.com/poweradmin/terraform-provider-poweradmin/discussions)
- **Poweradmin Documentation**: [docs.poweradmin.org](https://docs.poweradmin.org/)

## Acknowledgments

This provider is built using the [Terraform Plugin Framework](https://github.com/hashicorp/terraform-plugin-framework).
