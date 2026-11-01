// Copyright Poweradmin Development Team 2025, 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccZoneResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccZoneResourceConfig("test-example-acc.com", "MASTER", "Test zone"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("poweradmin_zone.test", "name", "test-example-acc.com"),
					resource.TestCheckResourceAttr("poweradmin_zone.test", "type", "MASTER"),
					resource.TestCheckResourceAttr("poweradmin_zone.test", "description", "Test zone"),
					resource.TestCheckResourceAttrSet("poweradmin_zone.test", "id"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "poweradmin_zone.test",
				ImportState:       true,
				ImportStateVerify: true,
				// Ignore template during import as it's only used during creation
				ImportStateVerifyIgnore: []string{"template"},
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func TestAccZoneResource_Slave(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create SLAVE zone
			{
				Config: testAccZoneResourceConfigSlave("test-slave-acc.example.com", "192.0.2.1,192.0.2.2"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("poweradmin_zone.test", "name", "test-slave-acc.example.com"),
					resource.TestCheckResourceAttr("poweradmin_zone.test", "type", "SLAVE"),
					resource.TestCheckResourceAttr("poweradmin_zone.test", "masters", "192.0.2.1,192.0.2.2"),
					resource.TestCheckResourceAttrSet("poweradmin_zone.test", "id"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "poweradmin_zone.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

// account is create-only: keeping it plans nothing, changing it is refused.
func TestAccZoneResource_AccountCreateOnly(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccZoneResourceConfigAccount("test-account-acc.com", "corp"),
				Check:  resource.TestCheckResourceAttr("poweradmin_zone.test", "account", "corp"),
			},
			{
				Config:   testAccZoneResourceConfigAccount("test-account-acc.com", "corp"),
				PlanOnly: true,
			},
			{
				Config:   testAccZoneResourceConfigAccount("test-account-acc.com", ""),
				PlanOnly: true,
			},
			{
				ResourceName:            "poweradmin_zone.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"template"},
			},
			{
				Config:      testAccZoneResourceConfigAccount("test-account-acc.com", "other"),
				ExpectError: regexp.MustCompile(`Account Cannot Be Changed`),
			},
			// A rename replaces the zone, so a new account is allowed with it.
			{
				Config: testAccZoneResourceConfigAccount("test-account2-acc.com", "other"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("poweradmin_zone.test", "name", "test-account2-acc.com"),
					resource.TestCheckResourceAttr("poweradmin_zone.test", "account", "other"),
				),
			},
		},
	})
}

// testAccZoneResourceConfigAccount leaves account out of the configuration when it is empty.
func testAccZoneResourceConfigAccount(name, account string) string {
	accountLine := ""
	if account != "" {
		accountLine = fmt.Sprintf("  account = %q\n", account)
	}
	return testAccProviderConfig() + fmt.Sprintf(`
resource "poweradmin_zone" "test" {
  name    = %[1]q
  type    = "MASTER"
%[2]s}
`, name, accountLine)
}

func testAccZoneResourceConfig(name, zoneType, description string) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "poweradmin_zone" "test" {
  name        = %[1]q
  type        = %[2]q
  description = %[3]q
}
`, name, zoneType, description)
}

func testAccZoneResourceConfigSlave(name, masters string) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "poweradmin_zone" "test" {
  name    = %[1]q
  type    = "SLAVE"
  masters = %[2]q
}
`, name, masters)
}
