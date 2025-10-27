# Copyright header configuration for Poweradmin Terraform Provider
# Uses HashiCorp's copywrite tool: https://github.com/hashicorp/copywrite
schema_version = 1

project {
  license        = "MPL-2.0"
  copyright_holder = "Poweradmin Development Team"
  copyright_year = 2025

  header_ignore = [
    # examples used within documentation (prose)
    "examples/**",

    # GitHub configuration
    ".github/**",

    # Configuration files
    ".golangci.yml",
    ".goreleaser.yml",

    # Documentation
    "docs/**",
    "*.md",
  ]
}
