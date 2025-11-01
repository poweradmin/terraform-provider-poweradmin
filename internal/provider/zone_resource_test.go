// Copyright (c) Poweradmin Development Team
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
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
				Config: testAccZoneResourceConfig("test-example.com", "MASTER", "Test zone"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("poweradmin_zone.test", "name", "test-example.com"),
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
			// Update and Read testing
			{
				Config: testAccZoneResourceConfig("test-example.com", "MASTER", "Updated test zone"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("poweradmin_zone.test", "description", "Updated test zone"),
				),
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
				Config: testAccZoneResourceConfigSlave("test-slave.example.com", "192.0.2.1,192.0.2.2"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("poweradmin_zone.test", "name", "test-slave.example.com"),
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
