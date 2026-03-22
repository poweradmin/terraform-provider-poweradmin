// Copyright Poweradmin Development Team 2025, 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccPermissionDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccPermissionDataSourceConfig(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.poweradmin_permission.test", "id"),
					resource.TestCheckResourceAttr("data.poweradmin_permission.test", "name", "zone_master_add"),
				),
			},
		},
	})
}

func testAccPermissionDataSourceConfig() string {
	return testAccProviderConfig() + `
data "poweradmin_permission" "test" {
  name = "zone_master_add"
}
`
}
