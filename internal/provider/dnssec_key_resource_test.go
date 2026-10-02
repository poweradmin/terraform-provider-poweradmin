// Copyright Poweradmin Development Team 2025, 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccDnssecKeyResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDnssecKeyResourceConfig(""),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("poweradmin_dnssec_key.test", "id"),
					resource.TestCheckResourceAttrSet("poweradmin_dnssec_key.test", "key_id"),
					resource.TestCheckResourceAttr("poweradmin_dnssec_key.test", "type", "csk"),
					resource.TestCheckResourceAttr("poweradmin_dnssec_key.test", "algorithm", "ecdsa256"),
					resource.TestCheckResourceAttr("poweradmin_dnssec_key.test", "bits", "256"),
					resource.TestCheckResourceAttr("poweradmin_dnssec_key.test", "active", "true"),
					resource.TestCheckResourceAttr("poweradmin_dnssec_key.test", "algorithm_id", "13"),
					resource.TestCheckResourceAttrSet("poweradmin_dnssec_key.test", "keytag"),
					resource.TestMatchResourceAttr("poweradmin_dnssec_key.test", "dnskey", regexp.MustCompile(`^257 3 13 `)),
					resource.TestMatchResourceAttr("poweradmin_dnssec_key.test", "ds.#", regexp.MustCompile(`^[1-9]`)),
				),
			},
			// Deactivating is an in-place update, not a replacement
			{
				Config: testAccDnssecKeyResourceConfig("active = false"),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("poweradmin_dnssec_key.test", plancheck.ResourceActionUpdate),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("poweradmin_dnssec_key.test", "active", "false"),
					resource.TestCheckResourceAttr("poweradmin_dnssec_key.test", "type", "csk"),
				),
			},
			{
				Config: testAccDnssecKeyResourceConfig("active = true"),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("poweradmin_dnssec_key.test", plancheck.ResourceActionUpdate),
					},
				},
				Check: resource.TestCheckResourceAttr("poweradmin_dnssec_key.test", "active", "true"),
			},
			{
				ResourceName:      "poweradmin_dnssec_key.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccDnssecKeyResource_ZSKHasNoDS(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig() + `
resource "poweradmin_zone" "test" {
  name = "test-dnssec-zsk-acc.example.com"
  type = "MASTER"
}

resource "poweradmin_dnssec_key" "ksk" {
  zone_id   = poweradmin_zone.test.id
  type      = "ksk"
  algorithm = "rsasha256"
  bits      = 2048
}

# PowerDNS only splits roles when an active SEP and non-SEP key share an algorithm
resource "poweradmin_dnssec_key" "zsk" {
  zone_id    = poweradmin_zone.test.id
  type       = "zsk"
  algorithm  = "rsasha256"
  bits       = 1024
  depends_on = [poweradmin_dnssec_key.ksk]
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("poweradmin_dnssec_key.ksk", "type", "ksk"),
					resource.TestCheckResourceAttr("poweradmin_dnssec_key.ksk", "algorithm_id", "8"),
					resource.TestCheckResourceAttr("poweradmin_dnssec_key.ksk", "bits", "2048"),
					resource.TestMatchResourceAttr("poweradmin_dnssec_key.ksk", "ds.#", regexp.MustCompile(`^[1-9]`)),
					resource.TestCheckResourceAttr("poweradmin_dnssec_key.zsk", "type", "zsk"),
					resource.TestCheckResourceAttr("poweradmin_dnssec_key.zsk", "active", "true"),
					resource.TestCheckResourceAttr("poweradmin_dnssec_key.zsk", "ds.#", "0"),
				),
			},
		},
	})
}

// PowerDNS reports a lone KSK as csk, so an import lands as csk; a ksk config
// must then update in place instead of replacing the key the registrar trusts.
func TestAccDnssecKeyResource_ImportLoneKSKKeepsKey(t *testing.T) {
	zone := `
resource "poweradmin_zone" "test" {
  name = "test-dnssec-ksk-import-acc.example.com"
  type = "MASTER"
}
`
	key := `
resource "poweradmin_dnssec_key" "ksk" {
  zone_id   = poweradmin_zone.test.id
  type      = "ksk"
  algorithm = "rsasha256"
  bits      = 2048
}
`
	// Forget the key without deleting it, so it can be imported as a fresh resource
	forget := `
removed {
  from = poweradmin_dnssec_key.ksk
  lifecycle {
    destroy = false
  }
}
`
	var importID string
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig() + zone + key,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("poweradmin_dnssec_key.ksk", "type", "ksk"),
					func(s *terraform.State) error {
						importID = s.RootModule().Resources["poweradmin_dnssec_key.ksk"].Primary.ID
						return nil
					},
				),
			},
			{
				Config: testAccProviderConfig() + zone + forget,
			},
			{
				Config:             testAccProviderConfig() + zone + key,
				ResourceName:       "poweradmin_dnssec_key.ksk",
				ImportState:        true,
				ImportStatePersist: true,
				ImportStateIdFunc:  func(*terraform.State) (string, error) { return importID, nil },
				ImportStateCheck: func(states []*terraform.InstanceState) error {
					if got := states[0].Attributes["type"]; got != "csk" {
						return fmt.Errorf("expected PowerDNS to report the lone KSK as csk on import, got %q", got)
					}
					return nil
				},
			},
			{
				Config: testAccProviderConfig() + zone + key,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("poweradmin_dnssec_key.ksk", plancheck.ResourceActionUpdate),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("poweradmin_dnssec_key.ksk", "type", "ksk"),
					func(s *terraform.State) error {
						if got := s.RootModule().Resources["poweradmin_dnssec_key.ksk"].Primary.ID; got != importID {
							return fmt.Errorf("key was replaced: id %s, want %s", got, importID)
						}
						return nil
					},
				),
			},
		},
	})
}

// Plan-time validation needs no server, so it runs without TF_ACC.
func TestDnssecKeyResource_ValidatesSpecAtPlan(t *testing.T) {
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
		{`type = "csk"` + "\n" + `algorithm = "ecdsa256"` + "\n" + `bits = 384`, `ecdsa256 requires 256 bits`},
		{`type = "csk"` + "\n" + `algorithm = "rsasha256"` + "\n" + `bits = 4096`, `rsasha256 requires 1024 or 2048 bits`},
		{`type = "csk"` + "\n" + `algorithm = "gost"` + "\n" + `bits = 512`, `algorithm must be one of`},
		{`type = "both"` + "\n" + `algorithm = "ecdsa256"` + "\n" + `bits = 256`, `type must be one of ksk, zsk, csk`},
	} {
		resource.UnitTest(t, resource.TestCase{
			ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
			Steps: []resource.TestStep{
				{
					Config: providerConfig + fmt.Sprintf(`
resource "poweradmin_dnssec_key" "test" {
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

func testAccDnssecKeyResourceConfig(activeLine string) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "poweradmin_zone" "test" {
  name = "test-dnssec-key-acc.example.com"
  type = "MASTER"
}

resource "poweradmin_dnssec_key" "test" {
  zone_id   = poweradmin_zone.test.id
  type      = "csk"
  algorithm = "ecdsa256"
  bits      = 256
  %s
}
`, activeLine)
}
