// Copyright (c) Poweradmin Development Team
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
)

// testAccProtoV6ProviderFactories are used to instantiate a provider during
// acceptance testing. The factory function will be invoked for every Terraform
// CLI command executed to create a provider server to which the CLI can
// reattach.
var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"poweradmin": providerserver.NewProtocol6WithError(New("test")()),
}

func testAccPreCheck(t *testing.T) {
	// You can add environment variable checks here to ensure the test environment is properly configured
	if v := os.Getenv("POWERADMIN_API_URL"); v == "" {
		t.Skip("POWERADMIN_API_URL must be set for acceptance tests")
	}

	// Check for authentication
	hasAPIKey := os.Getenv("POWERADMIN_API_KEY") != ""
	hasBasicAuth := os.Getenv("POWERADMIN_USERNAME") != "" && os.Getenv("POWERADMIN_PASSWORD") != ""

	if !hasAPIKey && !hasBasicAuth {
		t.Skip("Either POWERADMIN_API_KEY or POWERADMIN_USERNAME and POWERADMIN_PASSWORD must be set for acceptance tests")
	}
}

// testAccProviderConfig returns a basic provider configuration for testing.
func testAccProviderConfig() string {
	apiKey := os.Getenv("POWERADMIN_API_KEY")
	if apiKey != "" {
		return `
provider "poweradmin" {
  api_url = "` + os.Getenv("POWERADMIN_API_URL") + `"
  api_key = "` + apiKey + `"
}
`
	}

	// Use basic auth if API key is not set
	return `
provider "poweradmin" {
  api_url  = "` + os.Getenv("POWERADMIN_API_URL") + `"
  username = "` + os.Getenv("POWERADMIN_USERNAME") + `"
  password = "` + os.Getenv("POWERADMIN_PASSWORD") + `"
}
`
}
