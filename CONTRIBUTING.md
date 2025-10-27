# Contributing to Terraform Provider for Poweradmin

Thank you for your interest in contributing to the Terraform Provider for Poweradmin! This document provides guidelines and instructions for contributing.

## Code of Conduct

This project follows the [Contributor Covenant Code of Conduct](https://www.contributor-covenant.org/). By participating, you are expected to uphold this code.

## Getting Started

### Prerequisites

- [Go](https://golang.org/doc/install) >= 1.24
- [Terraform](https://developer.hashicorp.com/terraform/downloads) >= 1.0 or [OpenTofu](https://opentofu.org/) >= 1.6
- [Make](https://www.gnu.org/software/make/)
- Access to a Poweradmin instance for testing (can be local development setup)

### Development Setup

1. **Fork and Clone**
   ```bash
   git clone https://github.com/YOUR_USERNAME/terraform-provider-poweradmin.git
   cd terraform-provider-poweradmin
   ```

2. **Build the Provider**
   ```bash
   make build
   ```

3. **Install Locally**
   ```bash
   make install
   ```

## Development Workflow

### Building

```bash
# Build the provider
make build

# Or use Go directly
go build -v ./...
```

### Code Formatting

```bash
# Format Go code
make fmt

# Or use gofmt directly
gofmt -s -w -e .
```

### Linting

```bash
# Run linter
make lint

# This runs golangci-lint
golangci-lint run
```

### Testing

#### Unit Tests

```bash
# Run unit tests
make test

# Run with coverage
go test -v -cover -timeout=120s -parallel=10 ./...
```

#### Acceptance Tests

Acceptance tests create real resources in Poweradmin and require a running instance.

```bash
# Set up environment variables
export TF_ACC=1
export POWERADMIN_API_URL="http://localhost:8080"
export POWERADMIN_API_KEY="your-test-api-key"

# Run acceptance tests
make testacc
```

**Important**: Acceptance tests may create real resources and should be run against a test instance, not production.

### Testing with a Local Poweradmin Instance

#### Using Docker

```bash
# Example docker-compose.yml setup (customize as needed)
version: '3'
services:
  poweradmin:
    image: poweradmin/poweradmin:latest
    ports:
      - "8080:80"
    environment:
      - API_ENABLED=true
```

## Making Changes

### Before You Start

1. Check existing [issues](https://github.com/poweradmin/terraform-provider-poweradmin/issues) and [pull requests](https://github.com/poweradmin/terraform-provider-poweradmin/pulls)
2. Create an issue to discuss significant changes before starting work
3. Fork the repository and create a feature branch

### Commit Guidelines

- Write clear, descriptive commit messages
- Use present tense ("Add feature" not "Added feature")
- Reference issues in commit messages (e.g., "Fixes #123")
- Keep commits focused and atomic

### Pull Request Process

1. **Update Documentation**
   - Update README.md if adding features
   - Add/update examples in `examples/`
   - Update CHANGELOG.md

2. **Add Tests**
   - Unit tests for new functionality
   - Acceptance tests for resources/data sources
   - Ensure all tests pass

3. **Generate Documentation**
   ```bash
   make generate
   ```
   This updates provider documentation from schema definitions.

4. **Submit Pull Request**
   - Provide a clear description of changes
   - Reference related issues
   - Ensure CI checks pass
   - Request review from maintainers

## Project Structure

```
terraform-provider-poweradmin/
├── internal/provider/          # Provider implementation
│   ├── provider.go            # Main provider
│   ├── client.go              # API client
│   ├── client_zones.go        # Zone operations
│   ├── client_records.go      # Record operations
│   ├── models.go              # Data structures
│   ├── zone_resource.go       # Zone resource
│   ├── record_resource.go     # Record resource
│   ├── zone_data_source.go    # Zone data source
│   └── *_test.go              # Tests
├── examples/                   # Usage examples
│   ├── provider/
│   ├── resources/
│   └── data-sources/
├── docs/                       # Generated documentation
├── main.go                     # Entry point
└── GNUmakefile                # Build tasks
```

## Adding New Resources

When adding a new resource:

1. **Create Resource File**
   - Follow naming: `{resource_name}_resource.go`
   - Implement CRUD operations
   - Add proper schema with descriptions
   - Include plan modifiers where appropriate

2. **Register Resource**
   - Add to `provider.go` in `Resources()` method

3. **Add Tests**
   - Create `{resource_name}_resource_test.go`
   - Add unit tests
   - Add acceptance tests

4. **Add Examples**
   - Create directory: `examples/resources/poweradmin_{resource_name}/`
   - Add `resource.tf` with usage examples
   - Add `import.sh` if resource supports import

5. **Generate Documentation**
   - Run `make generate`
   - Review generated docs

## Adding New Data Sources

Similar process to resources:

1. Create `{name}_data_source.go`
2. Register in `provider.go`
3. Add tests
4. Add examples in `examples/data-sources/`
5. Generate documentation

## Code Style

- Follow Go best practices and idioms
- Use `gofmt` for formatting
- Add comments for exported functions/types
- Use meaningful variable names
- Keep functions focused and concise

### Schema Descriptions

- Use `MarkdownDescription` for all schema attributes
- Be clear and concise
- Include examples where helpful
- Document default values

Example:
```go
"ttl": schema.Int64Attribute{
    MarkdownDescription: "Time to Live in seconds. Defaults to 3600.",
    Optional:            true,
    Computed:            true,
    Default:             int64default.StaticInt64(3600),
},
```

## API Client Guidelines

When modifying the API client:

- Use context for all operations
- Log requests and responses with `tflog`
- Handle errors comprehensively
- Parse API error responses properly
- Use appropriate HTTP methods

## Testing Guidelines

### Unit Tests

- Test schema validation
- Test CRUD logic
- Test error handling
- Mock API responses when possible

### Acceptance Tests

- Test full resource lifecycle
- Test import functionality
- Test edge cases
- Clean up resources after tests

Example:
```go
func TestAccZoneResource(t *testing.T) {
    resource.Test(t, resource.TestCase{
        ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
        Steps: []resource.TestStep{
            // Create and Read testing
            {
                Config: testAccZoneResourceConfig("example.com"),
                Check: resource.ComposeAggregateTestCheckFunc(
                    resource.TestCheckResourceAttr("poweradmin_zone.test", "name", "example.com"),
                    resource.TestCheckResourceAttr("poweradmin_zone.test", "type", "MASTER"),
                ),
            },
            // ImportState testing
            {
                ResourceName:      "poweradmin_zone.test",
                ImportState:       true,
                ImportStateVerify: true,
            },
            // Update and Read testing
            // Delete testing automatically occurs
        },
    })
}
```

## Documentation

### Provider Documentation

Generated automatically from schemas using `tfplugindocs`:

```bash
make generate
```

Review generated files in `docs/` and make manual adjustments if needed.

### README Updates

Update README.md when:
- Adding new resources or data sources
- Changing provider configuration
- Adding new features
- Updating requirements

## Releasing

Releases are managed by maintainers. The process:

1. Update version in code
2. Update CHANGELOG.md
3. Create git tag
4. Push tag to trigger release workflow
5. Publish to Terraform Registry

## Getting Help

- **Issues**: [GitHub Issues](https://github.com/poweradmin/terraform-provider-poweradmin/issues)
- **Discussions**: [GitHub Discussions](https://github.com/poweradmin/terraform-provider-poweradmin/discussions)
- **Poweradmin**: [Poweradmin Documentation](https://docs.poweradmin.org/)

## Additional Resources

- [Terraform Plugin Framework Documentation](https://developer.hashicorp.com/terraform/plugin/framework)
- [Terraform Plugin Development](https://developer.hashicorp.com/terraform/plugin)
- [Poweradmin API Documentation](https://docs.poweradmin.org/configuration/api/)
- [Go Documentation](https://golang.org/doc/)

## License

By contributing, you agree that your contributions will be licensed under the MPL-2.0 License.
