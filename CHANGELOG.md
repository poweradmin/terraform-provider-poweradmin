# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.1.0] - TBD

### Added
- Initial provider implementation with Terraform Plugin Framework
- Zone resource (`poweradmin_zone`) for managing DNS zones
  - Support for MASTER, SLAVE, and NATIVE zone types
  - Import support by ID or name
  - Template support during creation
- Record resource (`poweradmin_record`) for managing DNS records
  - Support for all DNS record types (A, AAAA, CNAME, MX, TXT, SRV, NS, PTR, etc.)
  - TTL configuration with sensible defaults
  - Priority support for MX and SRV records
  - Import via `zone_id/record_id` format
- User resource (`poweradmin_user`) for managing Poweradmin users
  - Username, password, and profile management
  - Active/inactive status control
  - Permission template assignment
  - LDAP support
- Zone data source (`poweradmin_zone`) for querying existing zones
- Permission data source (`poweradmin_permission`) for querying permission templates
- Dual authentication support (API key and HTTP basic auth)
- Comprehensive acceptance tests for zones, records, and data sources
- Auto-generated documentation using terraform-docs
- GoReleaser configuration for multi-platform builds
- GitHub Actions workflows for testing and releases
- OpenTofu compatibility (1.6+) alongside Terraform (1.0+)

### Changed
- Updated default API version to v2 (for Poweradmin 4.1.0+)
- Enhanced client with user and permission management capabilities

[Unreleased]: https://github.com/poweradmin/terraform-provider-poweradmin/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/poweradmin/terraform-provider-poweradmin/releases/tag/v0.1.0
