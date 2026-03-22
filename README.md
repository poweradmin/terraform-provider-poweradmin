# Terraform Provider for Poweradmin

Manage DNS zones, records, RRSets, users, and groups in [Poweradmin](https://www.poweradmin.org/) using Terraform or OpenTofu.

## Features

- **Zone Management**: Create, update, and delete DNS zones (MASTER, SLAVE, NATIVE types)
- **Record Management**: Full CRUD for all DNS record types (A, AAAA, CNAME, MX, TXT, SRV, NS, PTR, CAA, and more)
- **RRSet Management**: Manage multiple records with same name/type as a single atomic unit
- **User Management**: Create and manage Poweradmin users with permission templates
- **Group Management**: Organize users into groups with zone access control (Poweradmin 4.2.0+)
- **Data Sources**: Query zones, records, RRSets, permissions, and groups
- **Dual Authentication**: API key or HTTP basic authentication
- **OpenTofu Compatible**: Works with both Terraform and OpenTofu

## Version Compatibility

| Provider Version | Poweradmin Version | Terraform | OpenTofu | Go (dev) |
|---|---|---|---|---|
| 0.3.0 | 4.2.0+ (groups), 4.1.0+ (core) | >= 1.5 | >= 1.6 | >= 1.26 |
| 0.2.0 | 4.1.0+ | >= 1.0 | >= 1.6 | >= 1.25 |
| 0.1.x | 4.1.0+ | >= 1.0 | >= 1.6 | >= 1.24 |

## Quick Start

### Installation

```hcl
terraform {
  required_providers {
    poweradmin = {
      source  = "poweradmin/poweradmin"
      version = "~> 0.3"
    }
  }
}

provider "poweradmin" {
  api_url = "https://dns.example.com"
  api_key = var.poweradmin_api_key
}
```

The provider is available on both the [Terraform Registry](https://registry.terraform.io/providers/poweradmin/poweradmin) and the [OpenTofu Registry](https://search.opentofu.org/provider/poweradmin/poweradmin).

### Minimal Example

```hcl
# Create a zone and add a record
resource "poweradmin_zone" "example" {
  name = "example.com"
  type = "MASTER"
}

resource "poweradmin_record" "www" {
  zone_id = poweradmin_zone.example.id
  name    = "www"
  type    = "A"
  content = "192.0.2.100"
  ttl     = 3600
}
```

## Guides

Detailed guides with examples for each feature area:

- **[Zone Management](examples/guides/zone-management.md)** - Master/Slave/Native zones, templates, accounts, imports
- **[Record Management](examples/guides/record-management.md)** - All DNS record types, priorities, TTLs, disabled records
- **[RRSet Management](examples/guides/rrset-management.md)** - Atomic multi-record sets, load balancing, partial disabling
- **[User Management](examples/guides/user-management.md)** - Users, permissions, LDAP, deactivation
- **[Group Management](examples/guides/group-management.md)** - Groups, memberships, zone assignments, MFA enforcement (4.2.0+)
- **[Data Sources](examples/guides/data-sources.md)** - Querying zones, records, RRSets, permissions, groups

## Resources and Data Sources

### Resources

| Resource | Description | Min Poweradmin |
|----------|-------------|----------------|
| `poweradmin_zone` | DNS zones (MASTER, SLAVE, NATIVE) | 4.1.0 |
| `poweradmin_record` | Individual DNS records | 4.1.0 |
| `poweradmin_rrset` | Resource Record Sets (atomic multi-record) | 4.1.0 |
| `poweradmin_user` | Users with permission templates | 4.1.0 |
| `poweradmin_group` | User groups with MFA enforcement | 4.2.0 |
| `poweradmin_group_membership` | Group member associations | 4.2.0 |
| `poweradmin_group_zone_assignment` | Group zone access associations | 4.2.0 |

### Data Sources

| Data Source | Description | Min Poweradmin |
|-------------|-------------|----------------|
| `poweradmin_zone` | Look up zone by ID or name | 4.1.0 |
| `poweradmin_records` | List records with optional type filter | 4.1.0 |
| `poweradmin_rrsets` | List RRSets with optional type filter | 4.1.0 |
| `poweradmin_permission` | Look up permission by ID or name | 4.1.0 |
| `poweradmin_group` | Look up group by ID or name | 4.2.0 |

## Provider Configuration

| Argument | Type | Required | Description |
|----------|------|----------|-------------|
| `api_url` | string | Yes | Poweradmin API base URL (e.g., `https://dns.example.com`) |
| `api_key` | string | No* | API key for authentication (recommended) |
| `username` | string | No* | Username for HTTP basic authentication |
| `password` | string | No* | Password for HTTP basic authentication |
| `api_version` | string | No | API version: only `v2` supported. Defaults to `v2` |
| `insecure` | bool | No | Skip TLS verification (default: `false`) |

\* Either `api_key` OR both `username` and `password` must be provided.

### Authentication Methods

```hcl
# API Key (recommended)
provider "poweradmin" {
  api_url = "https://dns.example.com"
  api_key = var.poweradmin_api_key
}

# Basic Auth
provider "poweradmin" {
  api_url  = "https://dns.example.com"
  username = var.poweradmin_username
  password = var.poweradmin_password
}
```

## Poweradmin API Setup

Enable the API in your Poweradmin `config/settings.php`:

```php
'api' => [
    'enabled' => true,
    'basic_auth_enabled' => true,  // Optional: for basic auth
]
```

To create an API key: log in as admin, navigate to API Keys, create a new key, and store it securely:

```bash
export TF_VAR_poweradmin_api_key="your-api-key-here"
```

## Development

See [DEVELOPMENT.md](DEVELOPMENT.md) for the full development guide and release process.

```bash
make build       # Build provider
make test        # Unit tests
make testacc     # Acceptance tests (requires API credentials)
make generate    # Generate docs
make lint        # Run linter
```

## Documentation

- [Provider Documentation](docs/index.md) (auto-generated schema reference)
- [Poweradmin Documentation](https://docs.poweradmin.org/)
- [Poweradmin API Documentation](https://docs.poweradmin.org/configuration/api/)
- [Poweradmin GitHub](https://github.com/poweradmin/poweradmin)

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for development setup, testing, and code style guidelines.

## License

MPL-2.0 - see [LICENSE](LICENSE).

## Support

- [GitHub Issues](https://github.com/poweradmin/terraform-provider-poweradmin/issues)

## Sponsors

<a href="https://www.stepping-stone.ch/">
  <img src=".github/stepping_stone_logo.svg" alt="stepping stone AG" height="60">
</a>

We thank [stepping stone AG](https://www.stepping-stone.ch/) for their support of this project.
