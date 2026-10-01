// Copyright Poweradmin Development Team 2025, 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccZoneDnssecResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccZoneDnssecResourceConfig(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair("poweradmin_zone_dnssec.test", "id", "poweradmin_zone.test", "id"),
					resource.TestCheckResourceAttr("poweradmin_zone_dnssec.test", "presigned", "false"),
					resource.TestMatchResourceAttr("poweradmin_zone_dnssec.test", "ds_records.#", regexp.MustCompile(`^[1-9]`)),
					resource.TestCheckResourceAttrSet("poweradmin_zone_dnssec.test", "ds_records.0.digest"),
					resource.TestCheckResourceAttrSet("poweradmin_zone_dnssec.test", "dnskey"),

					resource.TestCheckResourceAttr("data.poweradmin_zone_dnssec.test", "enabled", "true"),
					resource.TestCheckResourceAttrPair("data.poweradmin_zone_dnssec.test", "dnskey", "poweradmin_zone_dnssec.test", "dnskey"),
					resource.TestMatchResourceAttr("data.poweradmin_dnssec_keys.test", "keys.#", regexp.MustCompile(`^[1-9]`)),
					resource.TestCheckResourceAttrSet("data.poweradmin_dnssec_keys.test", "keys.0.key_id"),
					resource.TestCheckResourceAttrSet("data.poweradmin_dnssec_keys.test", "keys.0.dnskey"),
				),
			},
			{
				ResourceName:      "poweradmin_zone_dnssec.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccZoneDnssecDataSource_Unsigned(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig() + `
resource "poweradmin_zone" "test" {
  name = "test-dnssec-unsigned-acc.example.com"
  type = "MASTER"
}

data "poweradmin_zone_dnssec" "test" {
  zone_id = poweradmin_zone.test.id
}

data "poweradmin_dnssec_keys" "test" {
  zone_id = poweradmin_zone.test.id
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.poweradmin_zone_dnssec.test", "enabled", "false"),
					resource.TestCheckResourceAttr("data.poweradmin_zone_dnssec.test", "ds_records.#", "0"),
					resource.TestCheckNoResourceAttr("data.poweradmin_zone_dnssec.test", "dnskey"),
					resource.TestCheckResourceAttr("data.poweradmin_dnssec_keys.test", "keys.#", "0"),
				),
			},
		},
	})
}

func testAccZoneDnssecResourceConfig() string {
	return testAccProviderConfig() + `
resource "poweradmin_zone" "test" {
  name = "test-zone-dnssec-acc.example.com"
  type = "MASTER"
}

# Signing requires active SOA and NS records
resource "poweradmin_record" "ns" {
  zone_id = poweradmin_zone.test.id
  name    = "@"
  type    = "NS"
  content = "ns1.example.net."
  ttl     = 3600
}

resource "poweradmin_zone_dnssec" "test" {
  zone_id    = poweradmin_zone.test.id
  depends_on = [poweradmin_record.ns]
}

data "poweradmin_zone_dnssec" "test" {
  zone_id    = poweradmin_zone.test.id
  depends_on = [poweradmin_zone_dnssec.test]
}

data "poweradmin_dnssec_keys" "test" {
  zone_id    = poweradmin_zone.test.id
  depends_on = [poweradmin_zone_dnssec.test]
}
`
}
