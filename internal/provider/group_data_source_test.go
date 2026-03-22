// Copyright Poweradmin Development Team 2025, 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccGroupDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccGroupDataSourceConfig(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.poweradmin_group.test", "name", "test-datasource-group-acc"),
					resource.TestCheckResourceAttrSet("data.poweradmin_group.test", "id"),
					resource.TestCheckResourceAttrSet("data.poweradmin_group.test", "perm_templ_id"),
				),
			},
		},
	})
}

func testAccGroupDataSourceConfig() string {
	return testAccProviderConfig() + `
resource "poweradmin_group" "test" {
  name          = "test-datasource-group-acc"
  description   = "Group for data source testing"
  perm_templ_id = 6
}

data "poweradmin_group" "test" {
  name = poweradmin_group.test.name
}
`
}
