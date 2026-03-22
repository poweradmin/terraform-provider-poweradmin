# Terraform Provider for Poweradmin - Development Guide

## Current Status

**Version**: 0.3.0
**Last Updated**: 2026-03-22
**Build Status**: Passing
**License**: MPL-2.0

## Quick Start

```bash
make install     # Build and install provider locally
make test        # Run unit tests
make testacc     # Run acceptance tests (requires API credentials)
make generate    # Update docs and copyright headers
```

## What's Implemented

### Resources
- `poweradmin_zone` - DNS zone management (MASTER, SLAVE, NATIVE)
- `poweradmin_record` - Individual DNS record management
- `poweradmin_rrset` - DNS Resource Record Set management
- `poweradmin_user` - User management with permissions
- `poweradmin_group` - Group management (Poweradmin 4.2.0+)
- `poweradmin_group_membership` - Group member management (Poweradmin 4.2.0+)
- `poweradmin_group_zone_assignment` - Group zone assignment (Poweradmin 4.2.0+)

### Data Sources
- `poweradmin_zone` - Query zones by ID or name
- `poweradmin_records` - Query records with filtering
- `poweradmin_rrsets` - Query RRSets with filtering
- `poweradmin_permission` - Query permissions by ID or name
- `poweradmin_group` - Query groups by ID or name (Poweradmin 4.2.0+)

### API Client
- Full HTTP client with context support
- Dual authentication (API key + basic auth)
- TLS configuration (including insecure mode for development)
- Request/response logging via terraform-plugin-log

## File Structure

```
terraform-provider-poweradmin/
├── internal/provider/
│   ├── provider.go                        # Provider definition & schema
│   ├── provider_test.go                   # Test framework setup
│   ├── client.go                          # HTTP client & API communication
│   ├── client_zones.go                    # Zone API operations
│   ├── client_records.go                  # Record API operations
│   ├── client_rrsets.go                   # RRSet API operations
│   ├── client_users.go                    # User API operations
│   ├── client_permissions.go              # Permission API operations
│   ├── client_groups.go                   # Group API operations
│   ├── client_bulk.go                     # Bulk operations API
│   ├── client_test.go                     # Unit tests (mock HTTP server)
│   ├── client_groups_test.go              # Group API unit tests
│   ├── models.go                          # Data models
│   ├── zone_resource.go                   # Zone resource
│   ├── record_resource.go                 # Record resource
│   ├── rrset_resource.go                  # RRSet resource
│   ├── user_resource.go                   # User resource
│   ├── group_resource.go                  # Group resource
│   ├── group_membership_resource.go       # Group membership resource
│   ├── group_zone_assignment_resource.go  # Group zone assignment resource
│   ├── zone_data_source.go               # Zone data source
│   ├── records_data_source.go            # Records data source
│   ├── rrsets_data_source.go             # RRSets data source
│   ├── permission_data_source.go         # Permission data source
│   ├── group_data_source.go              # Group data source
│   └── *_test.go                         # Acceptance tests
├── examples/                             # Usage examples (used by doc generator)
├── docs/                                 # Auto-generated documentation
├── .github/workflows/                    # CI/CD workflows
│   ├── test.yml                          # Build, lint, and test matrix
│   └── release.yml                       # GoReleaser on version tags
├── main.go                               # Entry point
├── go.mod                                # Go module definition
├── CHANGELOG.md                          # Release notes
├── README.md                             # User documentation
└── DEVELOPMENT.md                        # This file
```

## Compatibility

| Provider | Poweradmin | Terraform | OpenTofu | Go |
|---|---|---|---|---|
| 0.3.0 | 4.2.0+ (groups), 4.1.0+ (core) | >= 1.5 | >= 1.6 | >= 1.26 |
| 0.2.0 | 4.1.0+ | >= 1.0 | >= 1.6 | >= 1.25 |
| 0.1.x | 4.1.0+ | >= 1.0 | >= 1.6 | >= 1.24 |

## Testing

### Unit Tests
Unit tests use `httptest.Server` to mock the Poweradmin API. No live instance needed.

```bash
make test
```

### Acceptance Tests
Require a running Poweradmin instance with API enabled.

```bash
export TF_ACC=1
export POWERADMIN_API_URL="http://localhost:8080"
export POWERADMIN_API_KEY="your-api-key"
make testacc
```

### CI Test Matrix
CI runs acceptance tests against Terraform 1.5-1.10 and OpenTofu (latest).

## Release Process

Releases are automated via [release-please](https://github.com/googleapis/release-please):

1. Merge PRs with [conventional commits](https://www.conventionalcommits.org/) (`feat:`, `fix:`, `docs:`, `chore:`)
2. release-please automatically creates/updates a **Release PR** with version bump and CHANGELOG
3. Review and merge the Release PR
4. release-please creates a `v*` tag
5. The tag triggers GoReleaser which builds binaries, signs checksums, and creates the GitHub Release
6. Terraform/OpenTofu registries pick up the release automatically

**Manual release** (if needed): `git tag v0.x.0 && git push origin v0.x.0`

## Technical Decisions

- **Terraform Plugin Framework** (not legacy SDK) for modern type safety and protocol v6
- **Dual authentication** for flexibility (API key preferred, basic auth fallback)
- **Separate association resources** for group memberships and zone assignments (follows AWS provider conventions for many-to-many relationships)
- **Composite IDs** (`group_id/user_id`) for association resources, parsed with `strings.SplitN`

## Known Limitations

1. No pagination handling for list endpoints
2. No retry logic for transient failures
3. No DNSSEC management
4. Group API endpoints need verification against live Poweradmin 4.2.0 instance
