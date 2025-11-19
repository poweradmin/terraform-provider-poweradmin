# Terraform Provider for Poweradmin

Manage DNS zones, records, RRSets, and users in [Poweradmin](https://www.poweradmin.org/) using Terraform or OpenTofu.

## Features

- **Zone Management**: Create, update, and delete DNS zones (MASTER, SLAVE, NATIVE types)
- **Record Management**: Full CRUD operations for DNS records (A, AAAA, CNAME, MX, TXT, SRV, NS, PTR, and more)
- **RRSet Management**: Manage DNS Resource Record Sets (multiple records with same name/type as a single unit)
- **User Management**: Create and manage Poweradmin users with permissions
- **Data Sources**: Query zones, records, RRSets, and permissions
- **Dual Authentication**: Support for API key and HTTP basic authentication
- **Version Support**: Compatible with Poweradmin 4.1.0+ (master/unreleased) - requires API v2
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

# API version is optional - defaults to v2
provider "poweradmin" {
  api_url     = "https://dns.example.com"
  api_key     = var.poweradmin_api_key
  api_version = "v2"  # Only v2 is supported (Poweradmin 4.1.0+), this is the default
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

### Managing RRSets (Multiple Records)

```hcl
# RRSet with multiple A records (load balancing)
resource "poweradmin_rrset" "web_servers" {
  zone_id = poweradmin_zone.example_com.id
  name    = "www"
  type    = "A"
  ttl     = 300

  records = [
    { content = "192.0.2.10", disabled = false },
    { content = "192.0.2.11", disabled = false },
    { content = "192.0.2.12", disabled = false },
  ]
}

# RRSet with MX records
resource "poweradmin_rrset" "mail" {
  zone_id = poweradmin_zone.example_com.id
  name    = "@"
  type    = "MX"
  ttl     = 3600

  records = [
    { content = "mail1.example.com.", priority = 10, disabled = false },
    { content = "mail2.example.com.", priority = 20, disabled = false },
  ]
}
```

### Managing Users

```hcl
resource "poweradmin_user" "dns_admin" {
  username    = "dns.admin"
  fullname    = "DNS Administrator"
  email       = "dns-admin@example.com"
  password    = var.dns_admin_password
  active      = true
  description = "DNS team administrator"
  perm_templ  = 1  # Administrator permission template
}
```

### Using Data Sources

```hcl
# Look up an existing zone
data "poweradmin_zone" "existing" {
  name = "existing.example.com"
}

# Query all A records in a zone
data "poweradmin_records" "a_records" {
  zone_id = data.poweradmin_zone.existing.id
  type    = "A"
}

# Query all RRSets in a zone
data "poweradmin_rrsets" "all" {
  zone_id = data.poweradmin_zone.existing.id
}

# Look up a permission
data "poweradmin_permission" "zone_edit" {
  name = "zone_content_edit_own"
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

- `poweradmin_zone` - Manages DNS zones (MASTER, SLAVE, NATIVE types)
- `poweradmin_record` - Manages individual DNS records
- `poweradmin_rrset` - Manages DNS Resource Record Sets (recommended for multiple records)
- `poweradmin_user` - Manages Poweradmin users with permissions

## Supported Data Sources

- `poweradmin_zone` - Query zone information by ID or name
- `poweradmin_records` - Query multiple DNS records from a zone with optional filtering
- `poweradmin_rrsets` - Query Resource Record Sets from a zone
- `poweradmin_permission` - Query permission information by ID or name

## Provider Configuration Options

| Argument | Type | Required | Description |
|----------|------|----------|-------------|
| `api_url` | string | Yes | Poweradmin API base URL (e.g., `https://dns.example.com`) |
| `api_key` | string | No* | API key for X-API-Key authentication (recommended) |
| `username` | string | No* | Username for HTTP basic authentication |
| `password` | string | No* | Password for HTTP basic authentication |
| `api_version` | string | No | API version: only `v2` is supported (Poweradmin 4.1.0+). Defaults to `v2` |
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

## Sponsors

<a href="https://www.stepping-stone.ch/">
  <img src=".github/stepping_stone_logo.svg" alt="stepping stone AG" height="60">
</a>

We thank [stepping stone AG](https://www.stepping-stone.ch/) for their support of this project.
