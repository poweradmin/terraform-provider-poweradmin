# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.4.0](https://github.com/poweradmin/terraform-provider-poweradmin/compare/v0.3.0...v0.4.0) (2026-03-25)


### ⚠ BREAKING CHANGES

* update client for standardized v2 API response wrapping (Poweradmin 4.3.0+)

### Features

* update client for standardized v2 API response wrapping (Poweradmin 4.3.0+) ([fca172d](https://github.com/poweradmin/terraform-provider-poweradmin/commit/fca172d9021d52ad72bf5595a7e8f70de3aac37d))


### Bug Fixes

* **ci:** correct release-please action SHA to v4.4.0 ([e634f40](https://github.com/poweradmin/terraform-provider-poweradmin/commit/e634f40edf98e7c564f4fa31b79d10451bad1b21))


### Miscellaneous

* **deps:** bump github.com/hashicorp/copywrite from 0.25.1 to 0.25.2 in /tools ([226f0c6](https://github.com/poweradmin/terraform-provider-poweradmin/commit/226f0c6732c592807d72e991841de7517e782d4c))
* **deps:** bump github.com/hashicorp/copywrite in /tools ([0dc4a23](https://github.com/poweradmin/terraform-provider-poweradmin/commit/0dc4a2376205ab84e79336e2654e19603a3b830a))
* **deps:** bump opentofu/setup-opentofu from 1.0.3 to 2.0.0 in the github-actions group ([66c3e98](https://github.com/poweradmin/terraform-provider-poweradmin/commit/66c3e98937b3d119b7bb7b7ac762ac1a8d13cdfd))
* **deps:** bump opentofu/setup-opentofu in the github-actions group ([38b7e1f](https://github.com/poweradmin/terraform-provider-poweradmin/commit/38b7e1f513ab28644673427b58fac4468faf6f7d))


### Documentation

* remove broken discussions link from CONTRIBUTING.md ([f2b3c4f](https://github.com/poweradmin/terraform-provider-poweradmin/commit/f2b3c4fd4d657f293ac25035f0584804490ccfbc))
* remove CLAUDE.md references and dead discussions link ([52479fc](https://github.com/poweradmin/terraform-provider-poweradmin/commit/52479fc446425c5bb899ac481a90fb638726607d))

## [Unreleased]

## [0.3.0] - 2026-03-22

### Added
- Group resource (`poweradmin_group`) for managing user groups (Poweradmin 4.2.0+)
  - Create groups with name, description, and permission template assignment
  - Import support by ID
- Group membership resource (`poweradmin_group_membership`) for managing group members
  - Add/remove users from groups using composite `group_id/user_id` identifier
  - Import support
- Group zone assignment resource (`poweradmin_group_zone_assignment`) for assigning zones to groups
  - Assign/unassign zones using composite `group_id/zone_id` identifier
  - All group members gain access to assigned zones
  - Import support
- Group data source (`poweradmin_group`) for querying groups by ID or name
- Comprehensive unit tests with mock HTTP server for all API client operations
- Acceptance tests for RRSet resource, User resource, Permission data source,
  Records data source, and RRSets data source

### Changed
- Updated Go version from 1.25.0 to 1.26.0
- Updated CI test matrix from Terraform 1.0-1.4 to Terraform 1.5-1.10
- Added OpenTofu acceptance test job in CI

## [0.2.0] - 2025-11-10

### Added
- RRSet resource (`poweradmin_rrset`) for DNS-correct record management
  - Manages multiple records with same name and type as a single unit
  - Matches PowerDNS RRSet behavior for atomic record updates
  - Full support for priority field in MX, SRV, and other record types
  - Supports disabled flag per record
- RRSets data source (`poweradmin_rrsets`) for querying zone RRSets
  - List all RRSets in a zone
  - Filter by record type
  - Returns complete RRSet data including all records with priorities
- Records data source (`poweradmin_records`) for querying zone records
  - List all records in a zone
  - Filter by record type
- Bulk operations API client for atomic multi-record operations
  - Supports create, update, and delete actions in single transaction
  - Available through Go client API for custom implementations

### Enhanced
- Improved master server validation for slave zones
  - Support for custom ports: `192.0.2.1:5300`
  - IPv6 with ports (requires brackets): `[2001:db8::1]:5300`
  - Multiple masters with mixed formats: `192.0.2.1:5300,[2001:db8::1]:5300`
  - Updated schema documentation with all supported formats

### Documentation
- Added comprehensive examples for new RRSet resource
- Added examples for RRSets and Records data sources
- Updated slave zone examples with new master server formats
- Added bulk operations usage guide for advanced scenarios

## [0.1.1] - 2024-11-04

### Fixed
- Removed non-functional 'dev' API version option (endpoint does not exist in Poweradmin)
- Clarified that only v2 API is supported (Poweradmin 4.1.0+)
- Updated all documentation to accurately reflect supported API version

### Documentation
- Updated README.md to remove references to unsupported v1 and 'dev' API versions
- Updated DEVELOPMENT.md compatibility section
- Updated provider schema description for api_version parameter

## [0.1.0] - 2024-11-03

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

[Unreleased]: https://github.com/poweradmin/terraform-provider-poweradmin/compare/v0.3.0...HEAD
[0.3.0]: https://github.com/poweradmin/terraform-provider-poweradmin/compare/v0.2.0...v0.3.0
[0.2.0]: https://github.com/poweradmin/terraform-provider-poweradmin/compare/v0.1.1...v0.2.0
[0.1.1]: https://github.com/poweradmin/terraform-provider-poweradmin/compare/v0.1.0...v0.1.1
[0.1.0]: https://github.com/poweradmin/terraform-provider-poweradmin/releases/tag/v0.1.0
