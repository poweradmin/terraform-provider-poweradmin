// Copyright Poweradmin Development Team 2025, 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
)

func TestAccZoneMetadataResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccZoneMetadataResourceConfig("ALLOW-AXFR-FROM", `["192.0.2.10", "AUTO-NS"]`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestMatchResourceAttr("poweradmin_zone_metadata.test", "id", regexp.MustCompile(`^\d+/ALLOW-AXFR-FROM$`)),
					resource.TestCheckResourceAttrPair("poweradmin_zone_metadata.test", "zone_id", "poweradmin_zone.test", "id"),
					resource.TestCheckResourceAttr("poweradmin_zone_metadata.test", "kind", "ALLOW-AXFR-FROM"),
					resource.TestCheckResourceAttr("poweradmin_zone_metadata.test", "values.#", "2"),
					resource.TestCheckTypeSetElemAttr("poweradmin_zone_metadata.test", "values.*", "192.0.2.10"),
					resource.TestCheckTypeSetElemAttr("poweradmin_zone_metadata.test", "values.*", "AUTO-NS"),
				),
			},
			// Changing the values is an in-place update, not a replacement
			{
				Config: testAccZoneMetadataResourceConfig("ALLOW-AXFR-FROM", `["192.0.2.0/24"]`),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("poweradmin_zone_metadata.test", plancheck.ResourceActionUpdate),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("poweradmin_zone_metadata.test", "values.#", "1"),
					resource.TestCheckTypeSetElemAttr("poweradmin_zone_metadata.test", "values.*", "192.0.2.0/24"),
				),
			},
			{
				ResourceName:      "poweradmin_zone_metadata.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Another kind is another object, so the old one is deleted and the new one created
			{
				Config: testAccZoneMetadataResourceConfig("ALSO-NOTIFY", `["192.0.2.53"]`),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("poweradmin_zone_metadata.test", plancheck.ResourceActionReplace),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestMatchResourceAttr("poweradmin_zone_metadata.test", "id", regexp.MustCompile(`^\d+/ALSO-NOTIFY$`)),
					resource.TestCheckResourceAttr("poweradmin_zone_metadata.test", "values.#", "1"),
				),
			},
		},
	})
}

// Poweradmin checks the values at apply; its refusal must reach the user.
func TestAccZoneMetadataResource_ServerRefusesSecondValue(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccZoneMetadataResourceConfig("AXFR-SOURCE", `["192.0.2.1"]`),
				Check:  resource.TestCheckTypeSetElemAttr("poweradmin_zone_metadata.test", "values.*", "192.0.2.1"),
			},
			{
				Config:      testAccZoneMetadataResourceConfig("AXFR-SOURCE", `["192.0.2.1", "192.0.2.2"]`),
				ExpectError: regexp.MustCompile(`AXFR-SOURCE\s+accepts\s+only\s+a\s+single\s+value`),
			},
		},
	})
}

// Plan-time validation needs no server, so it runs without TF_ACC.
func TestZoneMetadataResource_ValidatesAtPlan(t *testing.T) {
	providerConfig := `
provider "poweradmin" {
  api_url = "http://127.0.0.1:1"
  api_key = "unused"
}
`
	for _, tc := range []struct {
		spec string
		err  string
	}{
		{`kind = "allow-axfr-from"` + "\n" + `values = ["192.0.2.10"]`, `use "ALLOW-AXFR-FROM"`},
		{`kind = " SOA-EDIT"` + "\n" + `values = ["INCEPTION-INCREMENT"]`, `upper case without surrounding whitespace`},
		{`kind = "X-THIS-KIND-IS-LONGER-THAN-32-CHARS"` + "\n" + `values = ["1"]`, `kind must be 1 to 32 characters long`},
		{`kind = "ALSO-NOTIFY"` + "\n" + `values = []`, `values must hold at least one value`},
		{`kind = "ALSO-NOTIFY"` + "\n" + `values = ["192.0.2.53 "]`, `must not be empty or start or end with whitespace`},
		{`kind = "ALSO-NOTIFY"` + "\n" + `values = [""]`, `must not be empty or start or end with whitespace`},
	} {
		resource.UnitTest(t, resource.TestCase{
			ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
			Steps: []resource.TestStep{
				{
					Config: providerConfig + fmt.Sprintf(`
resource "poweradmin_zone_metadata" "test" {
  zone_id = 1
  %s
}
`, tc.spec),
					PlanOnly:    true,
					ExpectError: regexp.MustCompile(regexp.QuoteMeta(tc.err)),
				},
			},
		})
	}
}

func testAccZoneMetadataResourceConfig(kind, values string) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "poweradmin_zone" "test" {
  name = "test-zone-metadata-acc.example.com"
  type = "MASTER"
}

resource "poweradmin_zone_metadata" "test" {
  zone_id = poweradmin_zone.test.id
  kind    = %q
  values  = %s
}
`, kind, values)
}
