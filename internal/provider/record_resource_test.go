// Copyright (c) Poweradmin Development Team
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccRecordResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create zone and record
			{
				Config: testAccRecordResourceConfig("test-record.example.com", "www", "A", "192.0.2.100", 3600),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("poweradmin_record.test", "name", "www"),
					resource.TestCheckResourceAttr("poweradmin_record.test", "type", "A"),
					resource.TestCheckResourceAttr("poweradmin_record.test", "content", "192.0.2.100"),
					resource.TestCheckResourceAttr("poweradmin_record.test", "ttl", "3600"),
					resource.TestCheckResourceAttrSet("poweradmin_record.test", "id"),
					resource.TestCheckResourceAttrSet("poweradmin_record.test", "zone_id"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "poweradmin_record.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update and Read testing
			{
				Config: testAccRecordResourceConfig("test-record.example.com", "www", "A", "192.0.2.101", 7200),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("poweradmin_record.test", "content", "192.0.2.101"),
					resource.TestCheckResourceAttr("poweradmin_record.test", "ttl", "7200"),
				),
			},
			// Delete testing automatically occurs in TestCase
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
				Config: testAccRecordResourceConfigMX("test-mx.example.com", "@", "mail.example.com.", 10, 3600),
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
			},
		},
	})
}

func TestAccRecordResource_CNAME(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create CNAME record
			{
				Config: testAccRecordResourceConfig("test-cname.example.com", "blog", "CNAME", "www.example.com.", 3600),
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
