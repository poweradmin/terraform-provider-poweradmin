// Copyright Poweradmin Development Team 2025, 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccRRSetsDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccRRSetsDataSourceConfig("test-rrsets-ds-acc.example.com"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.poweradmin_rrsets.test", "zone_id"),
					resource.TestCheckResourceAttrSet("data.poweradmin_rrsets.test", "rrsets.#"),
				),
			},
		},
	})
}

func testAccRRSetsDataSourceConfig(zoneName string) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "poweradmin_zone" "test" {
  name = %[1]q
  type = "MASTER"
}

resource "poweradmin_rrset" "test" {
  zone_id = poweradmin_zone.test.id
  name    = "www"
  type    = "A"
  ttl     = 3600

  records = [
    { content = "192.0.2.1" },
  ]
}

data "poweradmin_rrsets" "test" {
  zone_id = poweradmin_zone.test.id

  depends_on = [poweradmin_rrset.test]
}
`, zoneName)
}
