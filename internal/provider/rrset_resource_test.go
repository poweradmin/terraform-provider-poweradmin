// Copyright Poweradmin Development Team 2025, 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccRRSetResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccRRSetResourceConfig("test-rrset-acc.example.com", "www", "A", 3600, "192.0.2.1"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("poweradmin_rrset.test", "name", "www"),
					resource.TestCheckResourceAttr("poweradmin_rrset.test", "type", "A"),
					resource.TestCheckResourceAttr("poweradmin_rrset.test", "ttl", "3600"),
					resource.TestCheckResourceAttrSet("poweradmin_rrset.test", "zone_id"),
				),
			},
			{
				ResourceName:      "poweradmin_rrset.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccRRSetResourceConfig("test-rrset-acc.example.com", "www", "A", 7200, "192.0.2.1"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("poweradmin_rrset.test", "ttl", "7200"),
				),
			},
		},
	})
}

func testAccRRSetResourceConfig(zoneName, name, recordType string, ttl int, content string) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "poweradmin_zone" "test" {
  name = %[1]q
  type = "MASTER"
}

resource "poweradmin_rrset" "test" {
  zone_id = poweradmin_zone.test.id
  name    = %[2]q
  type    = %[3]q
  ttl     = %[4]d

  records = [
    { content = %[5]q },
  ]
}
`, zoneName, name, recordType, ttl, content)
}
