// Copyright Poweradmin Development Team 2025, 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccUserResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccUserResourceConfig("testuser", "Test User", "testuser@example.com", true),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("poweradmin_user.test", "username", "testuser"),
					resource.TestCheckResourceAttr("poweradmin_user.test", "fullname", "Test User"),
					resource.TestCheckResourceAttr("poweradmin_user.test", "email", "testuser@example.com"),
					resource.TestCheckResourceAttr("poweradmin_user.test", "active", "true"),
					resource.TestCheckResourceAttrSet("poweradmin_user.test", "id"),
				),
			},
			// ImportState testing
			{
				ResourceName:            "poweradmin_user.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"password"},
			},
			// Update and Read testing
			{
				Config: testAccUserResourceConfig("testuser", "Updated User", "updated@example.com", false),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("poweradmin_user.test", "fullname", "Updated User"),
					resource.TestCheckResourceAttr("poweradmin_user.test", "email", "updated@example.com"),
					resource.TestCheckResourceAttr("poweradmin_user.test", "active", "false"),
				),
			},
		},
	})
}

func testAccUserResourceConfig(username, fullname, email string, active bool) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "poweradmin_user" "test" {
  username = %[1]q
  password = "TestPassword123!"
  fullname = %[2]q
  email    = %[3]q
  active   = %[4]t
}
`, username, fullname, email, active)
}
