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

func TestNormalizeRecordName(t *testing.T) {
	tests := []struct {
		name       string
		configured string
		fromAPI    string
		zoneName   string
		want       string
	}{
		{"identical relative", "www", "www", "example.com", "www"},
		{"identical apex", "@", "@", "example.com", "@"},
		{"fqdn preserved", "www.example.com", "www", "example.com", "www.example.com"},
		{"fqdn with trailing dot preserved", "www.example.com.", "www", "example.com", "www.example.com."},
		{"fqdn case-insensitive match", "WWW.Example.COM", "www", "example.com", "WWW.Example.COM"},
		{"multi-label fqdn preserved", "sub.www.example.com", "sub.www", "example.com", "sub.www.example.com"},
		{"zone fqdn preserved for apex", "example.com", "@", "example.com", "example.com"},
		{"zone fqdn with dot preserved for apex", "example.com.", "@", "example.com", "example.com."},
		{"external rename surfaces", "www", "mail", "example.com", "mail"},
		{"external rename to apex surfaces", "www", "@", "example.com", "@"},
		{"relative multi-label drift surfaces", "sub.www", "sub", "example.com", "sub"},
		{"foreign fqdn not preserved for apex", "other.org", "@", "example.com", "@"},
		{"similar prefix is not a match", "wwwx.example.com", "www", "example.com", "www"},
		{"unknown zone name trusts configured", "www.example.com", "www", "", "www.example.com"},
		{"unknown zone name keeps identical value", "www", "www", "", "www"},
		{"unknown zone name with empty configured takes api value", "", "www", "", "www"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := normalizeRecordName(tt.configured, tt.fromAPI, tt.zoneName); got != tt.want {
				t.Errorf("normalizeRecordName(%q, %q, %q) = %q, want %q", tt.configured, tt.fromAPI, tt.zoneName, got, tt.want)
			}
		})
	}
}

func TestNormalizeRecordContent(t *testing.T) {
	tests := []struct {
		name       string
		configured string
		fromAPI    string
		want       string
	}{
		{"trailing dot preserved", "mail.example.com.", "mail.example.com", "mail.example.com."},
		{"identical values", "mail.example.com", "mail.example.com", "mail.example.com"},
		{"api keeps dot", "mail.example.com.", "mail.example.com.", "mail.example.com."},
		{"external change surfaces", "mail.example.com.", "other.example.com", "other.example.com"},
		{"empty configured takes api value", "", "mail.example.com", "mail.example.com"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := normalizeRecordContent(tt.configured, tt.fromAPI); got != tt.want {
				t.Errorf("normalizeRecordContent(%q, %q) = %q, want %q", tt.configured, tt.fromAPI, got, tt.want)
			}
		})
	}
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
