// Copyright Poweradmin Development Team 2025, 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccZoneTemplateResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccZoneTemplateResourceConfig("acc-template", "Acceptance template"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("poweradmin_zone_template.test", "name", "acc-template"),
					resource.TestCheckResourceAttr("poweradmin_zone_template.test", "description", "Acceptance template"),
					resource.TestCheckResourceAttr("poweradmin_zone_template.test", "is_global", "false"),
					resource.TestCheckResourceAttrSet("poweradmin_zone_template.test", "id"),
					resource.TestCheckResourceAttrSet("poweradmin_zone_template.test", "owner"),
				),
			},
			{
				ResourceName:      "poweradmin_zone_template.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccZoneTemplateResourceConfig("acc-template", "Updated description"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("poweradmin_zone_template.test", "description", "Updated description"),
				),
			},
		},
	})
}

func TestAccZoneTemplateRecordResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccZoneTemplateRecordResourceConfig("acc-template-rec", "www", "A", "192.0.2.10", 3600),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("poweradmin_zone_template_record.www", "name", "www"),
					resource.TestCheckResourceAttr("poweradmin_zone_template_record.www", "type", "A"),
					resource.TestCheckResourceAttr("poweradmin_zone_template_record.www", "content", "192.0.2.10"),
					resource.TestCheckResourceAttr("poweradmin_zone_template_record.www", "ttl", "3600"),
					resource.TestCheckResourceAttrSet("poweradmin_zone_template_record.www", "id"),
					resource.TestCheckResourceAttrSet("poweradmin_zone_template_record.www", "record_id"),
				),
			},
			{
				ResourceName:      "poweradmin_zone_template_record.www",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccZoneTemplateRecordResourceConfig("acc-template-rec", "www", "A", "192.0.2.20", 7200),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("poweradmin_zone_template_record.www", "content", "192.0.2.20"),
					resource.TestCheckResourceAttr("poweradmin_zone_template_record.www", "ttl", "7200"),
				),
			},
		},
	})
}

func testAccZoneTemplateResourceConfig(name, description string) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "poweradmin_zone_template" "test" {
  name        = %[1]q
  description = %[2]q
}
`, name, description)
}

func testAccZoneTemplateRecordResourceConfig(tmplName, recName, recType, content string, ttl int) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "poweradmin_zone_template" "test" {
  name        = %[1]q
  description = "Template for record acceptance test"
}

resource "poweradmin_zone_template_record" "www" {
  template_id = poweradmin_zone_template.test.id
  name        = %[2]q
  type        = %[3]q
  content     = %[4]q
  ttl         = %[5]d
}
`, tmplName, recName, recType, content, ttl)
}
