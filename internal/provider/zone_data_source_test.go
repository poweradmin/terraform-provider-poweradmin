// Copyright (c) Poweradmin Development Team
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccZoneDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create a zone and read it with data source
			{
				Config: testAccZoneDataSourceConfig("test-datasource.example.com"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.poweradmin_zone.test", "name", "test-datasource.example.com"),
					resource.TestCheckResourceAttr("data.poweradmin_zone.test", "type", "MASTER"),
					resource.TestCheckResourceAttrSet("data.poweradmin_zone.test", "id"),
				),
			},
		},
	})
}

func testAccZoneDataSourceConfig(name string) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "poweradmin_zone" "test" {
  name = %[1]q
  type = "MASTER"
}

data "poweradmin_zone" "test" {
  name = poweradmin_zone.test.name
}
`, name)
}
