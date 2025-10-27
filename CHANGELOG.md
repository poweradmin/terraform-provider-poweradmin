## 0.1.0 (Unreleased)

FEATURES:

**Initial Release**

This is the initial release of the Terraform Provider for Poweradmin, enabling infrastructure-as-code management of DNS zones and records.

**New Resources:**
- **`poweradmin_zone`**: Manage DNS zones with support for MASTER, SLAVE, and NATIVE types
  - Create, update, and delete zones
  - Import existing zones by ID or name
  - Support for master nameservers (SLAVE zones)
  - Account and description fields
  - Template support during creation

- **`poweradmin_record`**: Manage DNS records within zones
  - Full CRUD operations for all DNS record types (A, AAAA, CNAME, MX, TXT, SRV, NS, PTR, etc.)
  - TTL configuration with sensible defaults
  - Priority support for MX and SRV records
  - Disabled flag for record activation control
  - Import via `zone_id/record_id` format

**New Data Sources:**
- **`poweradmin_zone`**: Query zone information by ID or name
  - Retrieve zone details for use in other resources
  - Flexible lookup options

**Provider Features:**
- **Dual Authentication**: Support for both API key (recommended) and HTTP basic authentication
- **Version Support**: Compatible with Poweradmin 4.0.x (stable, v1 API) and master branch (dev)
- **OpenTofu Compatible**: Works seamlessly with both Terraform (1.0+) and OpenTofu (1.6+)
- **TLS Configuration**: Secure by default with optional insecure mode for development
- **Comprehensive Logging**: Context-aware logging for debugging via tflog

**Documentation:**
- Complete provider configuration examples
- Resource usage examples for common DNS scenarios
- Data source examples
- Import examples for existing infrastructure
- Poweradmin API setup instructions

**Notes:**
- This provider uses the Terraform Plugin Framework
- Compatible with both Terraform and OpenTofu without any special configuration
- Requires Poweradmin instance with API enabled
- All API operations use the `/api/v1/` endpoint structure
