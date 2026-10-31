// Copyright Poweradmin Development Team 2025, 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccZoneMetadataDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccZoneMetadataDataSourceConfig(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair("data.poweradmin_zone_metadata.test", "id", "poweradmin_zone_metadata.axfr", "id"),
					resource.TestCheckResourceAttr("data.poweradmin_zone_metadata.test", "values.#", "2"),
					resource.TestCheckTypeSetElemAttr("data.poweradmin_zone_metadata.test", "values.*", "192.0.2.10"),
					resource.TestCheckTypeSetElemAttr("data.poweradmin_zone_metadata.test", "values.*", "AUTO-NS"),

					// Poweradmin matches the kind case-insensitively; the configured spelling is kept
					resource.TestCheckResourceAttr("data.poweradmin_zone_metadata.lower", "kind", "also-notify"),
					resource.TestCheckResourceAttr("data.poweradmin_zone_metadata.lower", "values.0", "192.0.2.53"),

					resource.TestCheckResourceAttrPair("data.poweradmin_zone_metadata_entries.test", "id", "poweradmin_zone.test", "id"),
					resource.TestCheckTypeSetElemNestedAttrs("data.poweradmin_zone_metadata_entries.test", "entries.*", map[string]string{
						"kind":     "ALLOW-AXFR-FROM",
						"values.#": "2",
					}),
					resource.TestCheckTypeSetElemNestedAttrs("data.poweradmin_zone_metadata_entries.test", "entries.*", map[string]string{
						"kind":     "ALSO-NOTIFY",
						"values.0": "192.0.2.53",
					}),
				),
			},
		},
	})
}

func TestAccZoneMetadataDataSource_KindWithoutValues(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig() + `
resource "poweradmin_zone" "test" {
  name = "test-zone-metadata-empty-acc.example.com"
  type = "MASTER"
}

data "poweradmin_zone_metadata" "test" {
  zone_id = poweradmin_zone.test.id
  kind    = "ALSO-NOTIFY"
}
`,
				ExpectError: regexp.MustCompile(`Error Reading Zone Metadata`),
			},
		},
	})
}

func testAccZoneMetadataDataSourceConfig() string {
	return testAccProviderConfig() + `
resource "poweradmin_zone" "test" {
  name = "test-zone-metadata-ds-acc.example.com"
  type = "MASTER"
}

resource "poweradmin_zone_metadata" "axfr" {
  zone_id = poweradmin_zone.test.id
  kind    = "ALLOW-AXFR-FROM"
  values  = ["192.0.2.10", "AUTO-NS"]
}

resource "poweradmin_zone_metadata" "notify" {
  zone_id = poweradmin_zone.test.id
  kind    = "ALSO-NOTIFY"
  values  = ["192.0.2.53"]
}

data "poweradmin_zone_metadata" "test" {
  zone_id = poweradmin_zone.test.id
  kind    = poweradmin_zone_metadata.axfr.kind
}

data "poweradmin_zone_metadata" "lower" {
  zone_id    = poweradmin_zone.test.id
  kind       = "also-notify"
  depends_on = [poweradmin_zone_metadata.notify]
}

data "poweradmin_zone_metadata_entries" "test" {
  zone_id    = poweradmin_zone.test.id
  depends_on = [poweradmin_zone_metadata.axfr, poweradmin_zone_metadata.notify]
}
`
}
