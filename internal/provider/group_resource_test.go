// Copyright Poweradmin Development Team 2025, 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccGroupResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccGroupResourceConfig("test-group-acc", "Test group for acceptance tests", 6),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("poweradmin_group.test", "name", "test-group-acc"),
					resource.TestCheckResourceAttr("poweradmin_group.test", "description", "Test group for acceptance tests"),
					resource.TestCheckResourceAttr("poweradmin_group.test", "perm_templ_id", "6"),
					resource.TestCheckResourceAttrSet("poweradmin_group.test", "id"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "poweradmin_group.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update and Read testing
			{
				Config: testAccGroupResourceConfig("test-group-acc", "Updated description", 6),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("poweradmin_group.test", "description", "Updated description"),
				),
			},
		},
	})
}

func testAccGroupResourceConfig(name, description string, permTemplID int) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "poweradmin_group" "test" {
  name          = %[1]q
  description   = %[2]q
  perm_templ_id = %[3]d
}
`, name, description, permTemplID)
}
