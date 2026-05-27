// Copyright Poweradmin Development Team 2025, 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccZoneTemplateDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccZoneTemplateDataSourceConfig(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.poweradmin_zone_template.test", "name", "acc-datasource-template"),
					resource.TestCheckResourceAttrSet("data.poweradmin_zone_template.test", "id"),
					resource.TestCheckResourceAttrSet("data.poweradmin_zone_template.test", "owner"),
				),
			},
		},
	})
}

func TestAccZoneTemplatesDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccZoneTemplatesDataSourceConfig(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.poweradmin_zone_templates.all", "templates.#"),
				),
			},
		},
	})
}

func testAccZoneTemplateDataSourceConfig() string {
	return testAccProviderConfig() + `
resource "poweradmin_zone_template" "test" {
  name        = "acc-datasource-template"
  description = "Template for data source acceptance test"
}

data "poweradmin_zone_template" "test" {
  name = poweradmin_zone_template.test.name
}
`
}

func testAccZoneTemplatesDataSourceConfig() string {
	return testAccProviderConfig() + `
resource "poweradmin_zone_template" "seed" {
  name        = "acc-zone-templates-list"
  description = "Seed template so the list is non-empty"
}

data "poweradmin_zone_templates" "all" {
  depends_on = [poweradmin_zone_template.seed]
}
`
}
