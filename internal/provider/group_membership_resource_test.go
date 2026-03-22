// Copyright Poweradmin Development Team 2025, 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccGroupMembershipResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccGroupMembershipResourceConfig(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("poweradmin_group_membership.test", "id"),
					resource.TestCheckResourceAttrSet("poweradmin_group_membership.test", "group_id"),
					resource.TestCheckResourceAttrSet("poweradmin_group_membership.test", "user_id"),
				),
			},
			{
				ResourceName:      "poweradmin_group_membership.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccGroupMembershipResourceConfig() string {
	return testAccProviderConfig() + `
resource "poweradmin_group" "test" {
  name          = "test-membership-group-acc"
  description   = "Group for membership testing"
  perm_templ_id = 6
}

resource "poweradmin_user" "test" {
  username = "test-member-user-acc"
  password = "TestPassword123!"
  fullname = "Test Member"
  email    = "member-acc@example.com"
  active   = true
}

resource "poweradmin_group_membership" "test" {
  group_id = poweradmin_group.test.id
  user_id  = poweradmin_user.test.id
}
`
}
