// Copyright (c) Poweradmin Development Team
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/echoprovider"
)

// testAccProtoV6ProviderFactories is used to instantiate a provider during acceptance testing.
// The factory function is called for each Terraform CLI command to create a provider
// server that the CLI can connect to and interact with.
var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"poweradmin": providerserver.NewProtocol6WithError(New("test")()),
}

// testAccProtoV6ProviderFactoriesWithEcho includes the echo provider alongside the poweradmin provider.
// It allows for testing assertions on data returned by an ephemeral resource during Open.
// The echoprovider is used to arrange tests by echoing ephemeral data into the Terraform state.
// This lets the data be referenced in test assertions with state checks.
var testAccProtoV6ProviderFactoriesWithEcho = map[string]func() (tfprotov6.ProviderServer, error){
	"poweradmin": providerserver.NewProtocol6WithError(New("test")()),
	"echo":       echoprovider.NewProviderServer(),
}

func testAccPreCheck(t *testing.T) {
	// Check that required environment variables are set for acceptance testing
	if v := os.Getenv("POWERADMIN_API_URL"); v == "" {
		t.Fatal("POWERADMIN_API_URL must be set for acceptance tests")
	}
	if v := os.Getenv("POWERADMIN_API_KEY"); v == "" {
		if v := os.Getenv("POWERADMIN_USERNAME"); v == "" {
			t.Fatal("Either POWERADMIN_API_KEY or POWERADMIN_USERNAME/POWERADMIN_PASSWORD must be set for acceptance tests")
		}
		if v := os.Getenv("POWERADMIN_PASSWORD"); v == "" {
			t.Fatal("POWERADMIN_PASSWORD must be set when using username authentication")
		}
	}
}
