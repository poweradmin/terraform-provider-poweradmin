// Copyright Poweradmin Development Team 2025, 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestNormalizeRRSetRecords(t *testing.T) {
	configured := []RRSetRecordModel{
		{Content: types.StringValue("mail1.example.com."), Disabled: types.BoolValue(false), Priority: types.Int64Value(10)},
		{Content: types.StringValue("mail2.example.com"), Disabled: types.BoolValue(false), Priority: types.Int64Value(20)},
	}
	// API strips the trailing dot and returns records in a different order
	fromAPI := []RRSetRecord{
		{Content: "mail2.example.com", Disabled: false, Priority: 20},
		{Content: "mail1.example.com", Disabled: false, Priority: 10},
	}

	got := normalizeRRSetRecords(configured, fromAPI)

	if len(got) != 2 {
		t.Fatalf("expected 2 records, got %d", len(got))
	}
	if got[0].Content.ValueString() != "mail2.example.com" {
		t.Errorf("expected exact match preserved, got %q", got[0].Content.ValueString())
	}
	if got[1].Content.ValueString() != "mail1.example.com." {
		t.Errorf("expected configured trailing dot preserved, got %q", got[1].Content.ValueString())
	}
	if got[0].Priority.ValueInt64() != 20 || got[1].Priority.ValueInt64() != 10 {
		t.Errorf("expected API priorities kept, got %d and %d", got[0].Priority.ValueInt64(), got[1].Priority.ValueInt64())
	}
}

func TestNormalizeRRSetRecords_ContentCollision(t *testing.T) {
	// Same target with different priorities: dot spelling must stay with its own element
	configured := []RRSetRecordModel{
		{Content: types.StringValue("mail.example.com."), Disabled: types.BoolValue(false), Priority: types.Int64Value(10)},
		{Content: types.StringValue("mail.example.com"), Disabled: types.BoolValue(false), Priority: types.Int64Value(20)},
	}
	fromAPI := []RRSetRecord{
		{Content: "mail.example.com", Disabled: false, Priority: 20},
		{Content: "mail.example.com", Disabled: false, Priority: 10},
	}

	got := normalizeRRSetRecords(configured, fromAPI)

	if len(got) != 2 {
		t.Fatalf("expected 2 records, got %d", len(got))
	}
	if got[0].Content.ValueString() != "mail.example.com" || got[0].Priority.ValueInt64() != 20 {
		t.Errorf("expected undotted priority-20 record, got %q p%d", got[0].Content.ValueString(), got[0].Priority.ValueInt64())
	}
	if got[1].Content.ValueString() != "mail.example.com." || got[1].Priority.ValueInt64() != 10 {
		t.Errorf("expected dotted priority-10 record, got %q p%d", got[1].Content.ValueString(), got[1].Priority.ValueInt64())
	}
}

func TestNormalizeRRSetRecords_ExternalChangeSurfaces(t *testing.T) {
	configured := []RRSetRecordModel{
		{Content: types.StringValue("old.example.com.")},
	}
	fromAPI := []RRSetRecord{
		{Content: "new.example.com"},
	}

	got := normalizeRRSetRecords(configured, fromAPI)

	if got[0].Content.ValueString() != "new.example.com" {
		t.Errorf("expected external change to surface, got %q", got[0].Content.ValueString())
	}
}

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
