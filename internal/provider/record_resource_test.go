// Copyright Poweradmin Development Team 2025, 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccRecordResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create zone and record
			{
				Config: testAccRecordResourceConfig("test-record-acc.example.com", "www", "A", "192.0.2.100", 3600),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("poweradmin_record.test", "name", "www"),
					resource.TestCheckResourceAttr("poweradmin_record.test", "type", "A"),
					resource.TestCheckResourceAttr("poweradmin_record.test", "content", "192.0.2.100"),
					resource.TestCheckResourceAttr("poweradmin_record.test", "ttl", "3600"),
					resource.TestCheckResourceAttrSet("poweradmin_record.test", "id"),
					resource.TestCheckResourceAttrSet("poweradmin_record.test", "zone_id"),
				),
			},
			// ImportState testing — record import needs zone_id/record_id format
			{
				ResourceName:      "poweradmin_record.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs, ok := s.RootModule().Resources["poweradmin_record.test"]
					if !ok {
						return "", fmt.Errorf("resource not found")
					}
					return fmt.Sprintf("%s/%s", rs.Primary.Attributes["zone_id"], rs.Primary.Attributes["id"]), nil
				},
			},
			// Update and Read testing
			{
				Config: testAccRecordResourceConfig("test-record-acc.example.com", "www", "A", "192.0.2.101", 7200),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("poweradmin_record.test", "content", "192.0.2.101"),
					resource.TestCheckResourceAttr("poweradmin_record.test", "ttl", "7200"),
				),
			},
		},
	})
}

func TestAccRecordResource_MX(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create MX record with priority
			{
				Config: testAccRecordResourceConfigMX("test-mx-acc.example.com", "@", "mail.example.com.", 10, 3600),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("poweradmin_record.test", "name", "@"),
					resource.TestCheckResourceAttr("poweradmin_record.test", "type", "MX"),
					resource.TestCheckResourceAttr("poweradmin_record.test", "content", "mail.example.com."),
					resource.TestCheckResourceAttr("poweradmin_record.test", "priority", "10"),
					resource.TestCheckResourceAttr("poweradmin_record.test", "ttl", "3600"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "poweradmin_record.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs, ok := s.RootModule().Resources["poweradmin_record.test"]
					if !ok {
						return "", fmt.Errorf("resource not found")
					}
					return fmt.Sprintf("%s/%s", rs.Primary.Attributes["zone_id"], rs.Primary.Attributes["id"]), nil
				},
				// Content may differ by trailing dot after import since state is lost
				ImportStateVerifyIgnore: []string{"content"},
			},
		},
	})
}

func TestAccRecordResource_CNAME(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccRecordResourceConfig("test-cname-acc.example.com", "blog", "CNAME", "www.example.com.", 3600),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("poweradmin_record.test", "name", "blog"),
					resource.TestCheckResourceAttr("poweradmin_record.test", "type", "CNAME"),
					resource.TestCheckResourceAttr("poweradmin_record.test", "content", "www.example.com."),
				),
			},
		},
	})
}

func testAccRecordResourceConfig(zoneName, recordName, recordType, content string, ttl int) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "poweradmin_zone" "test" {
  name = %[1]q
  type = "MASTER"
}

resource "poweradmin_record" "test" {
  zone_id = poweradmin_zone.test.id
  name    = %[2]q
  type    = %[3]q
  content = %[4]q
  ttl     = %[5]d
}
`, zoneName, recordName, recordType, content, ttl)
}

func testAccRecordResourceConfigMX(zoneName, recordName, content string, priority, ttl int) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "poweradmin_zone" "test" {
  name = %[1]q
  type = "MASTER"
}

resource "poweradmin_record" "test" {
  zone_id  = poweradmin_zone.test.id
  name     = %[2]q
  type     = "MX"
  content  = %[3]q
  priority = %[4]d
  ttl      = %[5]d
}
`, zoneName, recordName, content, priority, ttl)
}

// Unicode targets, rrset names and zone lookups must not drift when the
// server stores them as punycode (Poweradmin 4.5.2+). Writes are chained:
// concurrent record writes to one zone can deadlock on MySQL.
func TestAccRecordResource_IDNTargets(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig() + `
resource "poweradmin_zone" "test" {
  name = "acc-idn-bücher.example"
  type = "MASTER"
}

resource "poweradmin_record" "cname" {
  zone_id = poweradmin_zone.test.id
  name    = "blog"
  type    = "CNAME"
  content = "www.bücher.example."
  ttl     = 3600
}

resource "poweradmin_record" "mx" {
  zone_id  = poweradmin_zone.test.id
  name     = "@"
  type     = "MX"
  content  = "Mail.Bücher.Example."
  priority = 10
  ttl      = 3600

  depends_on = [poweradmin_record.cname]
}

resource "poweradmin_rrset" "test" {
  zone_id = poweradmin_zone.test.id
  name    = "münchen"
  type    = "CNAME"
  ttl     = 3600
  records = [{ content = "host.bücher.example." }]

  depends_on = [poweradmin_record.mx]
}

data "poweradmin_zone" "test" {
  name       = "acc-idn-bücher.example"
  depends_on = [poweradmin_zone.test]
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("poweradmin_record.cname", "content", "www.bücher.example."),
					resource.TestCheckResourceAttr("poweradmin_record.mx", "content", "Mail.Bücher.Example."),
					resource.TestCheckResourceAttr("poweradmin_record.mx", "priority", "10"),
					resource.TestCheckResourceAttr("poweradmin_rrset.test", "name", "münchen"),
					resource.TestCheckResourceAttrPair("data.poweradmin_zone.test", "id", "poweradmin_zone.test", "id"),
				),
			},
		},
	})
}

// The server stores ech=X as ech="X"; an unquoted ech in config must not drift.
func TestAccRecordResource_HTTPSUnquotedEch(t *testing.T) {
	const content = "1 . alpn=h2 ech=AEn+DQBFKwAgACABWIHUGj4u+PIggYXcR5JF0gYk3dCRioBW8uJq9H4mKAAIAAEAAQABAANAEnB1YmxpYy50bHMtZWNoLmRldgAA"
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig() + fmt.Sprintf(`
resource "poweradmin_zone" "test" {
  name = "acc-ech.example"
  type = "MASTER"
}

resource "poweradmin_record" "https" {
  zone_id = poweradmin_zone.test.id
  name    = "svc"
  type    = "HTTPS"
  content = %[1]q
  ttl     = 3600
}

resource "poweradmin_rrset" "svcb" {
  zone_id = poweradmin_zone.test.id
  name    = "_8443._https.svc"
  type    = "SVCB"
  ttl     = 3600
  records = [{ content = %[1]q }]

  depends_on = [poweradmin_record.https]
}
`, content),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("poweradmin_record.https", "content", content),
					resource.TestCheckResourceAttr("poweradmin_rrset.svcb", "records.0.content", content),
				),
			},
		},
	})
}
