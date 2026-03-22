// Copyright Poweradmin Development Team 2025, 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccGroupZoneAssignmentResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccGroupZoneAssignmentResourceConfig(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("poweradmin_group_zone_assignment.test", "id"),
					resource.TestCheckResourceAttrSet("poweradmin_group_zone_assignment.test", "group_id"),
					resource.TestCheckResourceAttrSet("poweradmin_group_zone_assignment.test", "zone_id"),
				),
			},
			{
				ResourceName:      "poweradmin_group_zone_assignment.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccGroupZoneAssignmentResourceConfig() string {
	return testAccProviderConfig() + `
resource "poweradmin_group" "test" {
  name          = "test-zone-assign-group-acc"
  description   = "Group for zone assignment testing"
  perm_templ_id = 7
}

resource "poweradmin_zone" "test" {
  name = "test-group-zone-acc.example.com"
  type = "MASTER"
}

resource "poweradmin_group_zone_assignment" "test" {
  group_id = poweradmin_group.test.id
  zone_id  = poweradmin_zone.test.id
}
`
}
